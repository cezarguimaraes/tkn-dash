package tekton

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/labstack/echo/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TemplateHandler(t *template.Template, name string) echo.HandlerFunc {
	return func(c echo.Context) error {
		tc := c.(*Context)
		td := &TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
		}

		c.Response().Header().Set(
			echo.HeaderContentType,
			echo.MIMETextHTMLCharsetUTF8,
		)

		c.Response().WriteHeader(http.StatusOK)
		return t.ExecuteTemplate(c.Response(), name, td)
	}
}

//go:embed templates/*
var templates embed.FS

func LoadTemplates(e *echo.Echo) (*template.Template, error) {
	t := template.New("all").Funcs(map[string]any{
		"obj_name": func(o metav1.Object) string {
			return o.GetName()
		},
		"step_url": func(data *TemplateData, taskRun string, step string) string {
			if data.PipelineRun != nil {
				return e.Reverse(
					"list-w-pipe-details",
					data.PipelineRun.GetNamespace(),
					"pipelineruns",
					data.PipelineRun.GetName(),
					taskRun,
					step,
				)
			}
			return e.Reverse(
				"list-w-task-details",
				data.TaskRun.GetNamespace(),
				"taskruns",
				taskRun,
				step,
			)
		},
		"url_for": func(name string, args ...interface{}) string {
			return e.Reverse(name, args...)
		},
	})

	return t.ParseFS(templates, "templates/*.html")
}
