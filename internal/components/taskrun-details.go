package components

import (
	"strconv"
	"strings"

	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	g "github.com/maragudk/gomponents"
	htmx "github.com/maragudk/gomponents-htmx"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type stepDetail struct {
	idx     int
	Name    string
	Content g.Node
	Active  bool
}

func detailsHTMX(td *tekton.TemplateData, route string) g.Node {
	return g.Group(
		[]g.Node{
			htmx.Get(td.URLFor(route, td.Namespace, td.TaskRun.GetName(), td.Step)),
			htmx.Trigger("revealed"),
		},
	)
}

func TaskRunDetails(td *tekton.TemplateData) g.Node {
	stepDetails := []*stepDetail{
		{
			Name: "Log",
			Content: Pre(
				Class("text-bg-secondary"),
				Code(
					ID("log-code"),
					detailsHTMX(td, "log"),
				),
			),
		},
		{
			Name: "Script",
			Content: Div(
				detailsHTMX(td, "script"),
			),
		},
		{
			Name: "Manifest",
			Content: Div(
				detailsHTMX(td, "manifest"),
			),
		},
	}

	for idx, sd := range stepDetails {
		sd.idx = idx
	}
	stepDetails[0].Active = true

	return Div(
		Table(
			Class("table table-dark table-striped"),
			THead(Tr(Th(g.Text("Name")), Th(g.Text("Value")))),
			TBody(
				g.Map(
					td.TaskRun.Spec.Params,
					func(p pipelinev1beta1.Param) g.Node {
						return Tr(
							Td(g.Text(p.Name)), Td(g.Text(p.Value.StringVal)),
						)
					},
				)...,
			),
		),
		Div(
			StyleAttr("flex-grow: 1;"),
			H3(g.Text(td.Step)),
			Div(
				Class("tabs tabs-boxed"),
				ID("step-details-tabs"),
				g.Group(g.Map(stepDetails, func(sd *stepDetail) g.Node {
					route := strings.ToLower(sd.Name)
					return A(
						c.Classes{
							"tab":        true,
							"tab-active": sd.Active,
						},
						htmx.Get(td.URLFor(route, td.Namespace, td.TaskRun.GetName(), td.Step)),
						htmx.Target("#step-details-content"),
						g.Text(sd.Name),
					)
				})),
			),
			Div(
				ID("step-details-content"),
			),
		),
		/*Div(
			StyleAttr("flex-grow: 1;"),
			H3(g.Text(td.Step)),
			Ul(
				Class("nav nav-tabs"),
				ID("myTab"),
				Role("tablist"),
				g.Group(g.Map(stepDetails, func(sd *stepDetail) g.Node {
					return stepTab(
						sd.idx,
						strings.ToLower(sd.Name),
						sd.idx == 0,
						g.Text(sd.Name),
					)
				})),
			),
			Div(
				Class("tab-content"),
				g.Group(g.Map(stepDetails, func(sd *stepDetail) g.Node {
					return stepContent(
						sd.idx,
						strings.ToLower(sd.Name),
						sd.idx == 0,
						sd.Content,
					)
				})),
			),
		),*/
	)
}

func stepTab(idx int, id string, active bool, content g.Node) g.Node {
	return Li(
		Class("nav-item"), Role("presentation"),
		Button(
			c.Classes{
				"nav-link": true,
				"active":   active,
			},
			ID(id+"tab"),
			DataAttr("bs-toggle", "tab"),
			DataAttr("bs-target", "#"+id+"pane"),
			Type("button"),
			Role("tab"),
			Aria("controls", id+"pane"),
			Aria("selected", strconv.FormatBool(active)),
			content,
		),
	)
}

func stepContent(idx int, id string, active bool, content g.Node) g.Node {
	return Div(
		c.Classes{
			"tab-pane": true,
			"active":   active,
		},
		ID(id+"pane"), Role("tabpanel"), Aria("labelledby", id+"tab"),
		TabIndex("0"),
		content,
	)
}
