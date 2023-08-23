package components

import (
	"github.com/cezarguimaraes/tkn-dash/internal/model"
	g "github.com/maragudk/gomponents"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
)

func Shell(content ...model.TektonComponent) model.TektonComponent {
	return func(td *model.TemplateData) g.Node {
		return c.HTML5(c.HTML5Props{
			Title: "tkn-dash",
			Head: []g.Node{
				Script(
					Src("/_static/htmx.min.js"),
					g.Attr(
						"integrity",
						"sha384-zUfuhFKKZCbHTY6aRR46gxiqszMk5tcHjsVFxnUo8VMus4kHGVdIYVbOYYNlKmHV",
					),
				),
				Link(
					Href("/_static/daisyui.css"),
					Rel("stylesheet"),
					Type("text/css"),
				),
				Script(
					Src("/_static/tailwindcss.js"),
				),
			},
			Body: append(
				g.Map(content, func(tc model.TektonComponent) g.Node {
					return tc(td)
				}),
				DataAttr("theme", "night"),
			),
		})
	}
}
