package components

import (
	"github.com/cezarguimaraes/tkn-dash/internal/handlers"
	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	g "github.com/maragudk/gomponents"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
)

func Shell(content ...handlers.TektonComponent) handlers.TektonComponent {
	return func(td *tekton.TemplateData) g.Node {
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
				//<link href="https://cdn.jsdelivr.net/npm/daisyui@3.5.1/dist/full.css" rel="stylesheet" type="text/css" />
				Link(
					Href("https://cdn.jsdelivr.net/npm/daisyui@3.5.1/dist/full.css"),
					Rel("stylesheet"),
					Type("text/css"),
				),
				Script(
					Src("https://cdn.tailwindcss.com"),
				),
				/*Link(
									Href("/_static/bootstrap.min.css"),
									Rel("stylesheet"),
									g.Attr(
										"integrity",
										"sha384-4bw+/aepP/YC94hEpVNVgiZdgIC5+VKNBQNGCHeKRQN+PtmoHDEXuppvnDJzQIu9",
									),
								),
								StyleEl(
									g.Text(`
				                    .list-group-item.active {
				                        background-color: var(--bs-secondary-bg-subtle) !important;
				                        border-color: var(--bs-secondary-bg-subtle) !important;
				                    }
				                `),
								),
				*/
			},
			Body: append(
				g.Map(content, func(tc handlers.TektonComponent) g.Node {
					return tc(td)
				}),
				DataAttr("theme", "night"),
				/*DataAttr("bs-theme", "dark"),
				Script(
					Src("/_static/bootstrap.bundle.min.js"),
					g.Attr(
						"integrity",
						"sha384-HwwvtgBNo3bZJJLYd8oVXjrBZt8cqVSpeBNS5n7C8IVInixGAoxmnlMuBnhbgrkm",
					),
				),*/
			),
		})
	}
}
