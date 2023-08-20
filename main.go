package main

import (
	"context"
	"flag"
	"net/http"
	"time"

	"github.com/cezarguimaraes/tkn-dash/internal/handlers"
	"github.com/cezarguimaraes/tkn-dash/internal/loader"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/cezarguimaraes/tkn-dash/internal/tools"
	"github.com/cezarguimaraes/tkn-dash/pkg/cache"
	"github.com/go-logr/logr"

	"github.com/labstack/echo/v4"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektoncs "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	kubeconfig  = flag.String("kubeconfig", "", "(optional) path to kubeconfig")
	chromaStyle = flag.String("syntax-style", "github-dark", "a valid style name from https://xyproto.github.io/splash/docs/")
)

func main() {
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

	tknMiddleware := tekton.NewMiddleware(prs, trs, namespaces)

	e := echo.New()

	e.Use(tknMiddleware)

	ts, err := tekton.LoadTemplates(e)
	if err != nil {
		log.Error(err, "error loading embedded templates")
		klog.FlushAndExit(10*time.Second, 1)
	}

	e.GET("/*", func(c echo.Context) error {
		return c.Redirect(
			http.StatusFound,
			c.Echo().Reverse("list", namespaces[0], "taskruns"),
		)
	})

	templateRoutes := []struct {
		name, route, template string
	}{
		{
			route:    "/:namespace/:resource",
			name:     "list",
			template: "index.html",
		},
		{
			route:    "/:namespace/:resource/:name",
			name:     "list-w-details",
			template: "index.html",
		},
		{
			route:    "/:namespace/:resource/:taskRun/step/:step",
			name:     "list-w-task-details",
			template: "index.html",
		},
		{
			route:    "/:namespace/:resource/:pipelineRun/taskruns/:taskRun/step/:step",
			name:     "list-w-pipe-details",
			template: "index.html",
		},
		{
			route:    "/:namespace/:resource/:name/details",
			name:     "details",
			template: "details.html",
		},
		{
			route:    "/:namespace/details/:taskRun/step/:step",
			name:     "details-w-step",
			template: "step-details",
		},
	}

	for _, rt := range templateRoutes {
		e.GET(
			rt.route,
			tekton.TemplateHandler(ts, rt.template),
		).Name = rt.name
	}

	e.GET("/:namespace/log/:taskRun/step/:step",
		handlers.StepLog(kubeclientset),
	).Name = "log"

	e.GET("/:namespace/script/:taskRun/step/:step",
		handlers.StepScript(*chromaStyle),
	).Name = "script"

	e.GET("/:namespace/manifest/:taskRun",
		handlers.Manifest(*chromaStyle),
	).Name = "manifest"

	e.GET("/:resource/items", handlers.Search(ts)).Name = "items"

	e.Logger.Fatal(e.Start(":8000"))
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
		tcs.TektonV1().RESTClient(),
		"taskruns",
		&pipelinev1.TaskRun{},
	)
	prInformer, prStopFn := cache.NewSharedInformerCache(
		tcs.TektonV1().RESTClient(),
		"pipelineruns",
		&pipelinev1.PipelineRun{},
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
