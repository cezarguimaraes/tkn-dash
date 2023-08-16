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
)

type foo struct {
	Name       string
	BigVarName int
}

func main() {
	e := echo.New()
	f, err := os.OpenFile("trs.json", os.O_RDONLY, os.ModePerm)
	if err != nil {
		e.Logger.Fatal(err)
	}
	var trs tekton.TaskRunList
	dec := json.NewDecoder(f)
	if err := dec.Decode(&trs); err != nil {
		e.Logger.Fatal(err)
	}

	fmt.Println(len(trs.Items))
	sort.Slice(trs.Items, func(i, j int) bool {
		return trs.Items[i].GetCreationTimestamp().
			Sub(trs.Items[j].GetCreationTimestamp().Time) > 0
	})

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	e.Renderer = renderer
	e.Debug = true
	e.Logger.SetLevel(log.DEBUG)

	// Named route "foobar"
	e.GET("/something", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", map[string]interface{}{})
	}).Name = "list"
	e.GET("/something/:name", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", map[string]interface{}{
			"Name": c.Param("name"),
		})
	}).Name = "list-w-details"

	e.GET("/details/:name", func(c echo.Context) error {
		name := c.Param("name")
		var found tekton.TaskRun
		for _, tr := range trs.Items {
			if tr.GetName() == name {
				found = tr
				break
			}
		}
		if found.GetName() == "" {
			return c.String(http.StatusNotFound, name+" not found")
		}
		// c.Response().Header().Set()
		return c.Render(http.StatusOK, "details.html", found)
	})

	e.GET("/details/:taskName/step/:stepName", func(c echo.Context) error {
		taskName := c.Param("taskName")
		var found tekton.TaskRun
		for _, tr := range trs.Items {
			if tr.GetName() == taskName {
				found = tr
				break
			}
		}
		if found.GetName() == "" {
			return c.String(http.StatusNotFound, taskName+" not found")
		}

		/*
			stepName := c.Param("stepName")
			var foundStep tekton.Step
			for _, step := range found.Status.TaskSpec.Steps {
				if step.Name == stepName {
					foundStep = step
				}
			}
		*/

		return c.Render(http.StatusOK, "step-details", map[string]string{
			"TaskName": taskName,
			"StepName": c.Param("stepName"),
		})
	})

	e.GET("/log/:taskName/step/:stepName", func(c echo.Context) error {
		taskName := c.Param("taskName")
		var found tekton.TaskRun
		for _, tr := range trs.Items {
			if tr.GetName() == taskName {
				found = tr
				break
			}
		}
		if found.GetName() == "" {
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

	e.GET("/taskruns", func(c echo.Context) error {
		pageSize := 100
		page := 0
		if pageStr := c.QueryParam("page"); pageStr != "" {
			page, _ = strconv.Atoi(pageStr)
		}

		from := page * pageSize
		to := min(len(trs.Items), (page+1)*pageSize)

		if from >= len(trs.Items) {
			return c.String(http.StatusOK, "")
		}

		type item struct {
			Name     string
			Age      string
			NextPage string
		}
		items := make([]item, 0, (to - from))
		now := time.Now()
		for i, tr := range trs.Items[from:to] {
			nextPage := ""
			if i+1 == pageSize {
				nextPage = "/taskruns?page=" + strconv.Itoa(page+1)
			}
			items = append(items, item{
				Name:     tr.GetObjectMeta().GetName(),
				NextPage: nextPage,
				Age:      now.Sub(tr.GetCreationTimestamp().Time).String(),
			})
		}

		return c.Render(http.StatusOK, "taskruns.html", items)
	}).Name = "taskruns"

	e.Logger.Fatal(e.Start(":8000"))
}

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}

	return t.templates.ExecuteTemplate(w, name, data)
}
