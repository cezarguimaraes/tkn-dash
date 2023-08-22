package handlers

import (
	"net/http"

	"github.com/cezarguimaraes/tkn-dash/internal/components"
	"github.com/cezarguimaraes/tkn-dash/internal/model"
	"github.com/cezarguimaraes/tkn-dash/internal/syntax"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v2"
)

const (
	cliLastApply = "kubectl.kubernetes.io/last-applied-configuration"
)

func Manifest(chromaStyle string) echo.HandlerFunc {
	return func(c echo.Context) error {
		tc := c.(*tekton.Context)
		td := &model.TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
		}

		// omit commonly large fields
		tr := td.TaskRun.DeepCopy()
		tr.ObjectMeta.ManagedFields = nil
		delete(tr.ObjectMeta.Annotations, cliLastApply)

		yml, err := yaml.Marshal(tr)
		if err != nil {
			return err
		}

		c.Response().WriteHeader(http.StatusOK)
		components.StepDetailsTabs(td, "manifest", true).
			Render(c.Response().Writer)
		return syntax.FormatHTML(
			c.Response().Writer,
			string(yml),
			syntax.WithStyle(chromaStyle),
			syntax.WithLinkPrefix("manifest-"),
			syntax.WithFallbackLanguage("yaml"),
		)
	}
}
