package handlers

import (
	"bytes"
	"html"
	"io"
	"net/http"

	"github.com/cezarguimaraes/tkn-dash/internal/components"
	"github.com/cezarguimaraes/tkn-dash/internal/model"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/labstack/echo/v4"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// TODO: poll from htmx until container finishes
func StepLog(cs *clientset.Clientset) echo.HandlerFunc {
	return func(c echo.Context) error {
		tc := c.(*tekton.Context)
		td := &model.TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
		}

		components.StepDetailsTabs(td, "log", true).
			Render(c.Response())

		if cs == nil {
			return c.String(
				http.StatusOK,
				"logs unavailable: tkn-dash initialized from files",
			)
		}

		req := cs.CoreV1().Pods(td.Namespace).GetLogs(
			td.TaskRun.Status.PodName,
			&v1.PodLogOptions{
				Container: "step-" + td.Step,
			},
		)
		rc, err := req.Stream(c.Request().Context())
		if err != nil {
			return err
		}
		defer rc.Close()

		var buf bytes.Buffer
		_, err = io.Copy(&buf, rc)
		if err != nil {
			return err
		}

		log := bytes.NewBufferString(html.EscapeString(buf.String()))
		_, err = io.Copy(c.Response(), log)
		return err
	}
}
