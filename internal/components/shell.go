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
				Link(
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
                        /* BS default blue is not at all readable with
                           red and green text */ 
                        background-color: var(--bs-secondary-bg-subtle) !important;
                        border-color: var(--bs-secondary-bg-subtle) !important;
                    } 
                `),
				),
			},
			Body: append(
				g.Map(content, func(tc handlers.TektonComponent) g.Node {
					return tc(td)
				}),
				DataAttr("bs-theme", "dark"),
				Script(
					Src("/_static/bootstrap.bundle.min.js"),
					g.Attr(
						"integrity",
						"sha384-HwwvtgBNo3bZJJLYd8oVXjrBZt8cqVSpeBNS5n7C8IVInixGAoxmnlMuBnhbgrkm",
					),
				),
			),
		})
	}
}
