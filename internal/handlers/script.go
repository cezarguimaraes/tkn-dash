package handlers

import (
	"net/http"

	"github.com/cezarguimaraes/tkn-dash/internal/components"
	"github.com/cezarguimaraes/tkn-dash/internal/model"
	"github.com/cezarguimaraes/tkn-dash/internal/syntax"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/labstack/echo/v4"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func StepScript(chromaStyle string) echo.HandlerFunc {
	return func(c echo.Context) error {
		tc := c.(*tekton.Context)
		td := &model.TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
		}

		var foundStep pipelinev1beta1.Step
		for _, step := range td.TaskRun.Status.TaskSpec.Steps {
			if step.Name == td.Step {
				foundStep = step
			}
		}

		c.Response().WriteHeader(http.StatusOK)
		components.StepDetailsTabs(td, "script", true).
			Render(c.Response().Writer)

		return syntax.FormatHTML(
			c.Response().Writer,
			foundStep.Script,
			syntax.WithStyle(chromaStyle),
			syntax.WithLinkPrefix("script-"),
		)
	}
}
