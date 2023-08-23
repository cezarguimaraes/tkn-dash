package components

import (
	"strings"

	"github.com/cezarguimaraes/tkn-dash/internal/model"
	g "github.com/maragudk/gomponents"
	htmx "github.com/maragudk/gomponents-htmx"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
)

const navBarContent = "navbar-content"

func NavBar(td *model.TemplateData) g.Node {
	return Div(
		Class("navbar bg-base-100"),
		Div(
			Class("navbar-start"),
			A(
				Href("https://github.com/cezarguimaraes/tkn-dash"),
				Target("_blank"),
				Class("btn btn-ghost normal-case text-xl"),
				g.Text("tkn-dash"),
			),
		),
		Div(
			Class("navbar-center flex"),
			Ul(
				Class("menu menu-horizontal grap-1"),
				g.Group(g.Map([]string{"PipelineRuns", "TaskRuns"}, func(r string) g.Node {
					active := strings.ToLower(r) == td.Resource
					return Li(
						Class("px-2"),
						A(
							c.Classes{
								"active": active,
							},
							Href(td.URLFor(
								"list",
								td.Namespace,
								strings.ToLower(r),
							)),
							g.Text(r),
						),
					)
				})),
			),
		),
	)
}

func Namespaces(td *model.TemplateData) g.Node {
	return Div(
		Class("mx-2 form-floating"),
		StyleAttr("flex-grow: 1;"),
		Select(
			Name("namespace"), ID("namespace"), Class("select select-primary w-full"),
			StyleAttr("flex-grow: 1;"),
			AutoComplete("off"),
			htmx.Get(td.URLFor("items", td.Resource)),
			htmx.Target("#items"),
			htmx.Swap("innerHTML"),
			htmx.Include("#search"),
			Option(
				Disabled(),
				g.Text("Namespace"),
			),
			g.Group(g.Map(td.Namespaces, func(ns string) g.Node {
				return Option(g.If(ns == td.Namespace, Selected()), g.Text(ns))
			})),
		),
	)
}

func Search(td *model.TemplateData) g.Node {
	return Div(
		g.Raw(`<div class="bg-gradient-to-r from-indigo-500 from-10% via-sky-500 via-30% to-emerald-500 to-90% "></div>`),
		ID("search"), Class("container-fluid my-3"),
		StyleAttr("display: flex;"),
		Div(
			Class("mx-2"),
			StyleAttr("flex-grow: 5;"),
			Input(
				Name("search"), ID("search-text"),
				Class("input input-bordered input-primary w-full"),
				Type("search"),
				Placeholder("Write a label selector (e.g: \"label=foo\" or \"label1 in (foo, bar), label2 != baz\")"),
				htmx.Get(td.URLFor("items", td.Resource)),
				htmx.Target("#items"),
				htmx.Swap("innerHTML"),
				htmx.Trigger("keyup changed delay:500ms, search"),
				htmx.Include("#search"),
			),
			Div(
				Class("pt-1 pe-2 text-xs text-end"),
				g.Text("For more information, check "),
				A(
					Class("link link-info"),
					Target("_blank"),
					Href("https://pkg.go.dev/k8s.io/apimachinery/pkg/labels#Parse"),
					g.Text("https://pkg.go.dev/k8s.io/apimachinery/pkg/labels#Parse"),
				),
				g.Text(" docs."),
			),
		),
		Namespaces(td),
	)
}

func ExplorerList(td *model.TemplateData) g.Node {
	return Table(
		Class("table table-zebra table-pin-rows"),
		THead(Tr(
			htmx.Get(td.URLFor("items", td.Resource)),
			htmx.Target("#items"),
			htmx.Include("#search"),
			htmx.Trigger("revealed"),
			htmx.Swap("innerHTML"),
			Th(g.Text("Name")),
			Th(g.Text("Age")),
		)),
		TBody(ID("items")),
	)
}

func iconFor(status string) g.Node {
	switch status {
	case "Failed":
		return g.Raw(`
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="text-error bi bi-x-circle-fill" viewBox="0 0 16 16">
                <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zM5.354 4.646a.5.5 0 1 0-.708.708L7.293 8l-2.647 2.646a.5.5 0 0 0 .708.708L8 8.707l2.646 2.647a.5.5 0 0 0 .708-.708L8.707 8l2.647-2.646a.5.5 0 0 0-.708-.708L8 7.293 5.354 4.646z"/>
            </svg>
        `)
	case "Succeeded":
		return g.Raw(`
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="text-success bi bi-check-circle-fill" viewBox="0 0 16 16">
                <path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0zm-3.97-3.03a.75.75 0 0 0-1.08.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05z"/>
            </svg>
        `)
	case "Running":
		return g.Raw(`
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="text-warning bi bi-circle-fill" viewBox="0 0 16 16">
                <circle cx="8" cy="8" r="8"/>
            </svg>
        `)
	}
	return g.Text("")
}

func ExplorerListItems(sr model.SearchResults) []g.Node {
	return g.Map(sr.Items, func(it model.SearchItem) g.Node {
		return Tr(
			g.If(
				it.NextPage != "",
				g.Group([]g.Node{
					htmx.Get(it.NextPage),
					htmx.Trigger("intersect once"),
					htmx.Swap("afterend"),
				}),
			),
			Td(
				A(
					Href("#"),
					Class("inline-flex"),
					htmx.Get(sr.URLFor(
						"details",
						it.Namespace,
						sr.Resource,
						it.Name,
					)),
					htmx.Target("#details"),
					htmx.Swap("outerHTML"),
					htmx.PushURL(sr.URLFor(
						"list-w-details",
						it.Namespace,
						sr.Resource,
						it.Name,
					)),
					Span(
						Class("pe-2"),
						iconFor(it.Status),
					),
					g.Text(it.Name),
				),
			),
			Td(Span(g.Text(it.Age))),
		)
	})
}

func Explorer(td *model.TemplateData) g.Node {
	return Div(
		Class("h-screen"), StyleAttr("display: flex; flex-direction: column;"),
		NavBar(td),
		Search(td),
		Div(
			ID("container"), Class("container-fluid"),
			StyleAttr("display: flex; overflow: auto;"),
			htmx.HistoryElt(""),
			Div(
				ID("left"), Class("mt-3"),
				StyleAttr("flex-shrink: 0; overflow: auto;"),
				ExplorerList(td),
			),
			Div(
				ID("right"), Class("pe-3"),
				StyleAttr("flex-grow: 1;"),
				TaskRuns(td),
			),
		),
	)
}
