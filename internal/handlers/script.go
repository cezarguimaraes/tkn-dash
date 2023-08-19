package handlers

import (
	"net/http"

	"github.com/cezarguimaraes/tkn-dash/internal/syntax"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/labstack/echo/v4"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func StepScript(chromaStyle string) echo.HandlerFunc {
	return func(c echo.Context) error {
		tc := c.(*tekton.Context)
		td := &tekton.TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
		}

		var foundStep pipelinev1.Step
		for _, step := range td.TaskRun.Status.TaskSpec.Steps {
			if step.Name == td.Step {
				foundStep = step
			}
		}

		c.Response().WriteHeader(http.StatusOK)
		return syntax.FormatHTML(
			c.Response().Writer,
			foundStep.Script,
			syntax.WithStyle(chromaStyle),
			syntax.WithLinkPrefix("script-"),
		)
	}
}
