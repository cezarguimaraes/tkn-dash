package handlers

import (
	"bytes"
	"html"
	"io"
	"net/http"

	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/labstack/echo/v4"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// TODO: poll from htmx until container finishes
func StepLog(cs *clientset.Clientset) echo.HandlerFunc {
	return func(c echo.Context) error {
		tc := c.(*tekton.Context)
		td := &tekton.TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
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

		return c.String(http.StatusOK, html.EscapeString(buf.String()))
	}
}
