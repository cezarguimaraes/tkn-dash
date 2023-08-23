package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/cezarguimaraes/tkn-dash/internal/components"
	"github.com/cezarguimaraes/tkn-dash/internal/handlers"
	"github.com/cezarguimaraes/tkn-dash/internal/loader"
	"github.com/cezarguimaraes/tkn-dash/internal/model"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/cezarguimaraes/tkn-dash/internal/tools"
	"github.com/cezarguimaraes/tkn-dash/pkg/cache"
	"github.com/go-logr/logr"
	"github.com/pkg/browser"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektoncs "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

//go:embed static/*
var static embed.FS

var (
	kubeconfig  = flag.String("kubeconfig", "", "(optional) path to kubeconfig")
	chromaStyle = flag.String("syntax-style", "github-dark", "a valid style name from https://xyproto.github.io/splash/docs/")
	addr        = flag.String("addr", ":", "[address]:port to listen on")
	openBrowser = flag.Bool("browser", false, "whether to try and open a browser to the dashboard")
)

func main() {
	klog.InitFlags(nil)

	flag.Parse()

	log := klog.NewKlogr()

	var trs, prs cache.Store
	var kubeclientset *clientset.Clientset

	if args := flag.Args(); len(args) > 0 {
		log.Info("loading tekton resources from files", "files", args)
		stores, err := loader.LoadLocalLists(args...)
		if err != nil {
			log.Error(err, "error loading tekton resources from files")
			klog.FlushAndExit(10*time.Second, 1)
		}
		trs = stores["taskrun"]
		prs = stores["pipelinerun"]
	} else {
		kubecfg, err := loadKubeConfig()
		if err != nil {
			log.Error(err, "error loading kubeconfig")
			klog.FlushAndExit(10*time.Second, 1)
		}

		kubeclientset, err = clientset.NewForConfig(kubecfg)
		if err != nil {
			log.Error(err, "error initializing kubernetes clientset")
			klog.FlushAndExit(10*time.Second, 1)
		}

		log.Info("loading tekton resources from cluster")
		tcs, err := tektoncs.NewForConfig(kubecfg)
		if err != nil {
			log.Error(err, "error loading tekton resources from cluster")
			klog.FlushAndExit(10*time.Second, 1)
		}

		var stopFn func()
		trs, prs, stopFn = initializeStores(log, tcs)
		defer stopFn()
	}

	nsLister := tools.NamespaceListerFromStore(trs, prs)
	namespaces, err := nsLister.List(context.Background())
	if err != nil {
		log.Error(err, "error listing namespaces")
		klog.FlushAndExit(10*time.Second, 1)
	}

	tknMiddleware := tekton.NewMiddleware(
		prs, trs,
		tekton.WithNamespaces(namespaces),
		tekton.WithLogger(log),
	)

	e := echo.New()

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:           true,
		LogError:         true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogMethod:        true,
		LogLatency:       true,
		LogResponseSize:  true,
		LogContentLength: true,
		LogStatus:        true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			keysAndValues := []interface{}{
				"method", v.Method,
				"URI", v.URI,
				"status", v.Status,
				"remote_ip", v.RemoteIP,
				"host", v.Host,
				"latency", v.Latency,
				"request_content_length", v.ContentLength,
				"response_size", v.ResponseSize,
			}
			if v.Error != nil {
				log.Error(v.Error, "error handling request", keysAndValues...)
			} else {
				log.V(2).Info("request responded", keysAndValues...)
			}
			return nil
		},
	}))

	// workaround to add middleware to .StaticFS
	staticGrp := e.Group("/_static",
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set(
					echo.HeaderCacheControl,
					"public, max-age=31536000",
				)
				return next(c)
			}
		},
	)
	staticGrp.StaticFS("/", echo.MustSubFS(static, "static"))

	e.Use(tknMiddleware)
	e.Use(middleware.Gzip())

	e.GET("/*", func(c echo.Context) error {
		return c.Redirect(
			http.StatusFound,
			c.Echo().Reverse("list", namespaces[0], "taskruns"),
		)
	})

	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.String(http.StatusNotFound, "not found")
	})

	componentRoutes := []struct {
		name, route string
		component   model.TektonComponent
	}{
		{
			route:     "/:namespace/:resource",
			name:      "list",
			component: components.Shell(components.Explorer),
		},
		{
			route:     "/:namespace/:resource/:name",
			name:      "list-w-details",
			component: components.Shell(components.Explorer),
		},
		{
			route:     "/:namespace/:resource/:taskRun/step/:step",
			name:      "list-w-task-details",
			component: components.Shell(components.Explorer),
		},
		{
			route:     "/:namespace/:resource/:taskRun/step/:step/:tab",
			name:      "list-w-task-details-tab",
			component: components.Shell(components.Explorer),
		},
		{
			route:     "/:namespace/:resource/:pipelineRun/taskruns/:taskRun/step/:step",
			name:      "list-w-pipe-details",
			component: components.Shell(components.Explorer),
		},
		{
			route:     "/:namespace/:resource/:pipelineRun/taskruns/:taskRun/step/:step/:tab",
			name:      "list-w-pipe-details-tab",
			component: components.Shell(components.Explorer),
		},
		{
			route:     "/:namespace/:resource/:name/details",
			name:      "details",
			component: components.TaskRuns,
		},
		{
			route:     "/:namespace/details/:taskRun/step/:step",
			name:      "details-w-step",
			component: components.TaskRunDetails(true),
		},
	}

	for _, ct := range componentRoutes {
		e.GET(
			ct.route,
			handlers.Component(ct.component),
		).Name = ct.name
	}

	e.GET("/log/:namespace/:taskRun/step/:step",
		handlers.StepLog(kubeclientset),
	).Name = "log"

	e.GET("/script/:namespace/:taskRun/step/:step",
		handlers.StepScript(*chromaStyle),
	).Name = "script"

	e.GET("/manifest/:namespace/:taskRun/step/:step",
		handlers.Manifest(*chromaStyle),
	).Name = "manifest"

	e.GET("/:resource/items",
		handlers.Search(
			components.ExplorerListItems,
		),
	).Name = "items"

	lc := net.ListenConfig{
		KeepAlive: 3 * time.Minute,
	}
	l, err := lc.Listen(context.Background(), "tcp4", *addr)
	if err != nil {
		log.Error(err, "failed to listen on specified address", *addr)
		klog.FlushAndExit(10*time.Second, 1)
	}

	tcpAddr := l.Addr().(*net.TCPAddr)
	localURL := fmt.Sprintf("http://127.0.0.1:%d", tcpAddr.Port)
	log.Info("server is listening", "address", l.Addr(), "URL", localURL)

	if *openBrowser {
		err := browser.OpenURL(localURL)
		if err != nil {
			log.Error(err, "failed to open browser", "URL", localURL)
		}
	}

	e.HideBanner = true
	e.HidePort = true
	e.Listener = l
	if err := e.Start(*addr); err != nil {
		log.Error(err, "error starting server")
	}
}

func loadKubeConfig() (*rest.Config, error) {
	lr := clientcmd.NewDefaultClientConfigLoadingRules()
	if *kubeconfig != "" {
		lr.ExplicitPath = *kubeconfig
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		lr,
		&clientcmd.ConfigOverrides{},
	)

	return clientConfig.ClientConfig()
}

func initializeStores(
	log logr.Logger,
	tcs *tektoncs.Clientset,
) (trs cache.Store, prs cache.Store, stopFn func()) {
	trInformer, trStopFn := cache.NewSharedInformerCache(
		tcs.TektonV1beta1().RESTClient(),
		"taskruns",
		&pipelinev1beta1.TaskRun{},
	)
	prInformer, prStopFn := cache.NewSharedInformerCache(
		tcs.TektonV1beta1().RESTClient(),
		"pipelineruns",
		&pipelinev1beta1.PipelineRun{},
	)

	stopFn = func() {
		prStopFn()
		trStopFn()
	}

	log.Info("waiting until shared informers have synced")
	for {
		time.Sleep(1 * time.Second)
		if !prInformer.HasSynced() {
			continue
		}
		if !trInformer.HasSynced() {
			continue
		}
		break
	}
	log.Info("shared informers have synced")

	return trInformer, prInformer, stopFn
}
