package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/alecthomas/chroma/quick"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektoncs "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"knative.dev/pkg/apis"
)

const namespace = "tmp"

var (
	thresholds = [4]time.Duration{time.Second, time.Minute, time.Hour, 24 * time.Hour}
	suffixes   = [4]string{"s", "m", "h", "d"}
)

func ageString(d time.Duration) (res string) {
	res = "1" + suffixes[0]
	for i, dur := range thresholds {
		if dur > d {
			return
		}
		res = strconv.Itoa(int((d+dur-1)/dur)) + suffixes[i] // round up
	}
	return
}

type templData struct {
	// Resource is the root object for this page, taskruns/pipelineruns
	Resource string

	// PipelineRun is resolved from the :pipelineRun url param
	PipelineRun *tekton.PipelineRun

	// TaskRun is resolved from the :taskRun url param
	TaskRun *tekton.TaskRun

	// TaskRuns is the list of taskRuns that should be rendered
	// in the middle "step view". It is either a list containing
	// a single taskRun in taskRun view, or the list of taskRuns
	// pertaining to a pipelineRUn
	TaskRuns []*tekton.TaskRun

	// Step is the name of the step resolved from the :step url param
	Step string
}

func main() {
	storage := "file"
	storage = "sharedInformer"

	var trs, prs Storage
	switch storage {
	case "file":
		var err error
		prs, err = NewFileStorage[*tekton.PipelineRun]("prs.json")
		if err != nil {
			panic(err)
		}

		trs, err = NewFileStorage[*tekton.TaskRun]("trs.json")
		if err != nil {
			panic(err)
		}
	case "sharedInformer":
		var err error
		config, err := clientcmd.BuildConfigFromFlags("", "/home/nonseq/.kube/config")
		if err != nil {
			panic(err.Error())
		}

		cs, err := tektoncs.NewForConfig(config)
		if err != nil {
			panic(err)
		}

		var stopFn func()
		prs, stopFn = NewSharedInformerStorage(
			cs.TektonV1().RESTClient(),
			namespace,
			"pipelineruns",
			&tekton.PipelineRun{},
		)
		defer stopFn()

		trs, stopFn = NewSharedInformerStorage(
			cs.TektonV1().RESTClient(),
			namespace,
			"taskruns",
			&tekton.TaskRun{},
		)
		defer stopFn()
	}

	e := echo.New()

	getTaskRun := func(name string) *tekton.TaskRun {
		tr, err := trs.Get(name)
		if err != nil {
			return nil
		}
		return tr.(*tekton.TaskRun)
	}

	getPipelineRun := func(name string) *tekton.PipelineRun {
		pr, err := prs.Get(name)
		if err != nil {
			return nil
		}
		return pr.(*tekton.PipelineRun)
	}

	getPipelineTaskRuns := func(name string) []*tekton.TaskRun {
		pr := getPipelineRun(name)
		if pr == nil {
			return nil
		}
		var trs []*tekton.TaskRun
		for _, cr := range pr.Status.ChildReferences {
			trs = append(trs, getTaskRun(cr.Name))
		}
		return trs
	}

	resolveTemplDataFromContext := func(c echo.Context) *templData {
		td := &templData{}
		for _, pn := range c.ParamNames() {
			switch pn {
			case "resource":
				// TODO: validate resource
				td.Resource = c.Param(pn)
			case "name":
				if td.Resource == "taskruns" {
					tr := getTaskRun(c.Param(pn))
					td.TaskRun = tr
					if tr != nil {
						td.TaskRuns = append(td.TaskRuns, tr)
					}
					break
				}

				fallthrough
			case "pipelineRun":
				prName := c.Param(pn)
				td.PipelineRun = getPipelineRun(prName)
				td.TaskRuns = getPipelineTaskRuns(prName)
			case "taskRun":
				tr := getTaskRun(c.Param(pn))
				td.TaskRun = tr
				if tr != nil && len(td.TaskRuns) == 0 {
					td.TaskRuns = append(td.TaskRuns, tr)
				}
			case "step":
				td.Step = c.Param(pn)
			}
		}

		for pn := range c.QueryParams() {
			switch pn {
			case "step":
				td.Step = c.QueryParam(pn)
			case "task":
				td.TaskRun = getTaskRun(c.QueryParam(pn))
				td.TaskRuns = []*tekton.TaskRun{td.TaskRun}
			}
		}

		// auto select first taskRun / step
		if td.Step == "" && len(td.TaskRuns) > 0 {
			if td.TaskRun == nil {
				td.TaskRun = td.TaskRuns[0]
			}
			td.Step = td.TaskRuns[0].Status.TaskSpec.Steps[0].Name
		}

		return td
	}

	t := template.New("all").Funcs(map[string]any{
		"obj_name": func(o metav1.Object) string {
			return o.GetName()
		},
		"step_url": func(data *templData, taskRun string, step string) string {
			if data.PipelineRun != nil {
				return e.Reverse(
					"list-w-pipe-details",
					"pipelineruns",
					data.PipelineRun.GetName(),
					taskRun,
					step,
				)
			}
			return e.Reverse(
				"list-w-task-details",
				"taskruns",
				taskRun,
				step,
			)
		},
		"url_for": func(name string, args ...interface{}) string {
			return e.Reverse(name, args...)
		},
	})

	renderer := &TemplateRenderer{
		templates: template.Must(t.ParseGlob("templates/*.html")),
	}

	e.Renderer = renderer
	e.Debug = true
	e.Logger.SetLevel(log.DEBUG)

	/*
	   :ns/taskruns/:taskrunname/
	*/

	supportedResources := map[string]struct{}{
		"taskruns":     {},
		"pipelineruns": {},
	}
	e.GET("/:resource", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", resolveTemplDataFromContext(c))
	}).Name = "list"
	e.GET("/:resource/:name", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", resolveTemplDataFromContext(c))
	}).Name = "list-w-details"
	e.GET("/:resource/:taskRun/step/:step", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", resolveTemplDataFromContext(c))
	}).Name = "list-w-task-details"
	e.GET("/:resource/:pipelineRun/taskruns/:taskRun/step/:step", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", resolveTemplDataFromContext(c))
	}).Name = "list-w-pipe-details"

	e.GET("/:resource/:name/details", func(c echo.Context) error {
		td := resolveTemplDataFromContext(c)

		return c.Render(http.StatusOK, "details.html", td)
	}).Name = "details"

	e.GET("/details/:taskName/step/:stepName", func(c echo.Context) error {
		taskName := c.Param("taskName")
		found := getTaskRun(taskName)
		if found == nil {
			return c.String(http.StatusNotFound, taskName+" not found")
		}

		return c.Render(http.StatusOK, "step-details", map[string]interface{}{
			"TaskName": taskName,
			"StepName": c.Param("stepName"),
			"Task":     found,
		})
	}).Name = "details-w-step"

	e.GET("/log/:taskName/step/:stepName", func(c echo.Context) error {
		taskName := c.Param("taskName")
		found := getTaskRun(taskName)
		if found == nil {
			return c.String(http.StatusNotFound, taskName+" not found")
		}

		stepName := c.Param("stepName")
		var foundStep tekton.Step
		for _, step := range found.Status.TaskSpec.Steps {
			if step.Name == stepName {
				foundStep = step
			}
		}

		var buf bytes.Buffer
		if err := quick.Highlight(
			&buf,
			foundStep.Script,
			"bash", "html", "monokai",
		); err == nil {
			return c.HTML(http.StatusOK, buf.String())
		}

		return c.String(http.StatusOK, foundStep.Script)
	})

	// TODO: pagination by ordered key
	e.GET("/:resource/items", func(c echo.Context) error {
		resource := c.Param("resource")
		if _, ok := supportedResources[resource]; !ok {
			return c.String(http.StatusNotFound, "Not Found")
		}

		var str Storage
		switch resource {
		case "taskruns":
			str = trs
		case "pipelineruns":
			str = prs
		}

		opts := &SearchOptions{
			Limit: 100,
		}
		if pageStr := c.QueryParam("page"); pageStr != "" {
			opts.ContinueFrom = &pageStr
		}

		results, continueFrom, err := str.Search(opts)
		if err != nil {
			return err
		}

		type item struct {
			Name     string
			Age      string
			Status   string
			NextPage string
		}
		items := make([]item, 0, len(results))

		now := time.Now()
		for i, r := range results {
			nextPage := ""
			if i+1 == len(results) && continueFrom != nil {
				nextPage = c.Echo().Reverse("items", resource) +
					"?page=" + *continueFrom
			}
			obj := r.(metav1.Object)

			items = append(items, item{
				Name:     obj.GetName(),
				NextPage: nextPage,
				Age: ageString(
					now.Sub(obj.GetCreationTimestamp().Time),
				) + " ago",
			})

			if pr, ok := r.(*tekton.PipelineRun); ok {
				cond := pr.GetStatusCondition().GetCondition(apis.ConditionSucceeded)
				if cond != nil {
					if cond.IsUnknown() {
						items[i].Status = "Running"
					}
					if cond.IsFalse() {
						items[i].Status = "Failed"
					}
					if cond.IsTrue() {
						items[i].Status = "Succeeded"
					}
				}
			}
		}

		return c.Render(http.StatusOK, "taskruns.html", map[string]interface{}{
			"Resource": resource,
			"Items":    items,
		})
	}).Name = "items"

	e.Logger.Fatal(e.Start(":8000"))
}

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type ContinueToken *string

type SearchOptions struct {
	ContinueFrom ContinueToken
	Limit        int
}

type Storage interface {
	Get(name string) (interface{}, error)

	Search(*SearchOptions) ([]interface{}, ContinueToken, error)
}

type fileStorage[T metav1.Object] struct {
	items   []interface{}
	nameMap map[string]interface{}
}

func NewFileStorage[T metav1.Object](path string) (*fileStorage[T], error) {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out struct {
		// parse as concrete type T but store as interface{},
		// otherwise we need to create interface{} slices
		// on every search
		Items []T `json:"items"`
	}

	dec := json.NewDecoder(f)
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}

	sort.Slice(out.Items, func(i, j int) bool {
		return out.Items[i].GetCreationTimestamp().
			Sub(out.Items[j].GetCreationTimestamp().Time) > 0
	})

	nameMap := make(map[string]interface{}, len(out.Items))
	items := make([]interface{}, 0, len(out.Items))
	for _, it := range out.Items {
		nameMap[it.GetName()] = it
		items = append(items, it)
	}

	return &fileStorage[T]{items, nameMap}, nil
}

func (s *fileStorage[T]) Get(name string) (interface{}, error) {
	found, ok := s.nameMap[name]
	if !ok {
		return nil, errors.New("key not found")
	}
	return found, nil
}

func (s *fileStorage[T]) Search(opts *SearchOptions) ([]interface{}, ContinueToken, error) {
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	from := 0
	if opts.ContinueFrom != nil {
		var err error
		from, err = strconv.Atoi(*opts.ContinueFrom)
		if err != nil {
			return nil, nil, err
		}
	}

	if from >= len(s.items) {
		return nil, nil, nil
	}

	to := min(len(s.items), from+opts.Limit)
	continueFrom := strconv.Itoa(to)
	return s.items[from:to], &continueFrom, nil
}

type sharedInformerStorage struct {
	lw      cache.ListerWatcher
	si      cache.SharedInformer
	closeCh chan struct{}
	// TODO: check if empty namespace works for all
	namespace string
}

func NewSharedInformerStorage(getter cache.Getter, namespace string, resource string, exampleObject runtime.Object) (*sharedInformerStorage, func()) {
	lw := cache.NewListWatchFromClient(
		getter,
		resource,
		namespace,
		fields.Everything(),
	)
	si := cache.NewSharedInformer(lw, exampleObject, 5*time.Minute)
	closeCh := make(chan struct{})
	go si.Run(closeCh)

	stopFunc := func() {
		close(closeCh)
	}
	return &sharedInformerStorage{lw, si, closeCh, namespace}, stopFunc
}

func (s *sharedInformerStorage) Get(name string) (interface{}, error) {
	it, exists, err := s.si.GetStore().GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("key does not exist")
	}
	return it, nil
}

func (s *sharedInformerStorage) Search(opts *SearchOptions) ([]interface{}, ContinueToken, error) {
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	// TODO: continue from based on item key/order
	// not position as items might change inbetween calls
	from := 0
	if opts.ContinueFrom != nil {
		var err error
		from, err = strconv.Atoi(*opts.ContinueFrom)
		if err != nil {
			return nil, nil, err
		}
	}

	items := s.si.GetStore().List()

	if from >= len(items) {
		return nil, nil, nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].(metav1.Object).GetCreationTimestamp().
			Sub(items[j].(metav1.Object).GetCreationTimestamp().Time) > 0
	})

	to := min(len(items), from+opts.Limit)
	continueFrom := strconv.Itoa(to)
	return items[from:to], &continueFrom, nil
}
