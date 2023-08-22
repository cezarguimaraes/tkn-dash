package handlers

import (
	"github.com/cezarguimaraes/tkn-dash/internal/model"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/labstack/echo/v4"
)

func Component(c model.TektonComponent) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		tc := ctx.(*tekton.Context)
		td := &model.TemplateData{}
		if err := tc.BindTemplateData(td); err != nil {
			return err
		}
		return c(td).Render(ctx.Response())
	}
}
