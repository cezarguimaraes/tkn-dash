package handlers

import (
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/labstack/echo/v4"
	g "github.com/maragudk/gomponents"
)

type TektonComponent func(*tekton.TemplateData) g.Node

func Component(c TektonComponent) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		tc := ctx.(*tekton.Context)
		td := &tekton.TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
		}
		return c(td).Render(ctx.Response())
	}
}
