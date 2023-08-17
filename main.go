package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func parseJSON(name string, out any) error {
	f, err := os.OpenFile(name, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
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
	e := echo.New()

	var prs tekton.PipelineRunList
	if err := parseJSON("prs.json", &prs); err != nil {
		e.Logger.Fatal(err)
	}

	var trs tekton.TaskRunList
	if err := parseJSON("trs.json", &trs); err != nil {
		e.Logger.Fatal(err)
	}

	getTaskRun := func(name string) *tekton.TaskRun {
		for i, tr := range trs.Items {
			if tr.GetName() == name {
				return &trs.Items[i]
			}
		}
		return nil
	}

	getPipelineRun := func(name string) *tekton.PipelineRun {
		for i, pr := range prs.Items {
			if pr.GetName() == name {
				return &prs.Items[i]
			}
		}
		return nil
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

		return td
	}
	fmt.Println(len(prs.Items), len(trs.Items))

	sort.Slice(prs.Items, func(i, j int) bool {
		return prs.Items[i].GetCreationTimestamp().
			Sub(prs.Items[j].GetCreationTimestamp().Time) > 0
	})

	sort.Slice(trs.Items, func(i, j int) bool {
		return trs.Items[i].GetCreationTimestamp().
			Sub(trs.Items[j].GetCreationTimestamp().Time) > 0
	})

	t := template.New("all").Funcs(map[string]any{
		"step_url": func(data *templData, taskRun string, step string) string {
			frag := fmt.Sprintf("/taskruns/%s/step/%s", taskRun, step)
			if data.PipelineRun != nil {
				frag = fmt.Sprintf(
					"/pipelineruns/%s%s",
					data.PipelineRun.GetName(),
					frag,
				)
			}
			return frag
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
	})
	e.GET("/:resource/:pipelineRun/taskruns/:taskRun/step/:step", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", resolveTemplDataFromContext(c))
	})

	e.GET("/:resource/:name/details", func(c echo.Context) error {
		td := resolveTemplDataFromContext(c)
		if step := c.QueryParam("step"); step != "" {
			td.Step = step
		}

		if task := c.QueryParam("task"); task != "" {
			td.TaskRun = getTaskRun(task)
			td.TaskRuns = []*tekton.TaskRun{td.TaskRun}
		}

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

		var resources []metav1.Object
		switch resource {
		case "taskruns":
			for i := range trs.Items {
				tr := &trs.Items[i]
				resources = append(resources, tr)
			}
		case "pipelineruns":
			for i := range prs.Items {
				pr := &prs.Items[i]
				resources = append(resources, pr)
			}
		}

		pageSize := 100
		page := 0
		if pageStr := c.QueryParam("page"); pageStr != "" {
			page, _ = strconv.Atoi(pageStr)
		}

		from := page * pageSize
		to := min(len(resources), (page+1)*pageSize)

		if from >= len(resources) {
			return c.String(http.StatusOK, "")
		}

		type item struct {
			Name     string
			Age      string
			NextPage string
		}
		items := make([]item, 0, (to - from))
		now := time.Now()
		for i, r := range resources[from:to] {
			nextPage := ""
			if i+1 == pageSize {
				nextPage = c.Echo().Reverse("items", resource) +
					"?page=" +
					strconv.Itoa(page+1)
			}
			items = append(items, item{
				Name:     r.GetName(),
				NextPage: nextPage,
				Age: ageString(
					now.Sub(r.GetCreationTimestamp().Time),
				) + " ago",
			})
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
