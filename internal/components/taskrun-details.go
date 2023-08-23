package components

import (
	"io"
	"strconv"
	"strings"

	"github.com/cezarguimaraes/tkn-dash/internal/model"
	g "github.com/maragudk/gomponents"
	htmx "github.com/maragudk/gomponents-htmx"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type stepDetail struct {
	Name   string
	Active bool
}

func StepDetailsTabs(td *model.TemplateData, active string, outOfBand bool) g.Node {
	stepDetails := []*stepDetail{
		{
			Name: "Log",
		},
		{
			Name: "Script",
		},
		{
			Name: "Manifest",
		},
	}

	noneActive := true
	for _, sd := range stepDetails {
		if strings.ToLower(sd.Name) == active {
			sd.Active = true
			noneActive = false
		}
	}
	if noneActive {
		stepDetails[0].Active = true
	}

	return Div(
		Class("tabs tabs-boxed"),
		ID("step-details-tabs"),
		g.If(
			outOfBand,
			htmx.SwapOOB("true"),
		),
		g.Group(g.Map(stepDetails, func(sd *stepDetail) g.Node {
			route := strings.ToLower(sd.Name)
			return A(
				c.Classes{
					"tab":        true,
					"tab-active": sd.Active,
				},
				htmx.Get(td.URLFor(route, td.Namespace, td.TaskRun.GetName(), td.Step)),
				htmx.Target("#step-details-content"),
				g.If(!outOfBand && sd.Active, htmx.Trigger("load")),
				g.If(outOfBand || !sd.Active, htmx.PushURL(stepTabURL(td, td.TaskRun.GetName(), td.Step, route))),
				g.Text(sd.Name),
			)
		})),
	)
}

type rGroup struct {
	children []g.Node
}

var _ g.Node = &rGroup{}

func (rg *rGroup) Render(w io.Writer) error {
	for _, c := range rg.children {
		if c == nil {
			continue
		}
		err := c.Render(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func RGroup(children ...g.Node) g.Node {
	return &rGroup{children}
}

type breadcrumb struct {
	kind, name string
}

func breadcrumbs(td *model.TemplateData) []breadcrumb {
	var x []breadcrumb
	if td.PipelineRun != nil {
		x = append(x, breadcrumb{name: td.PipelineRun.GetName(), kind: "PR"})
	}
	if td.TaskRun != nil {
		x = append(x, breadcrumb{name: td.TaskRun.GetName(), kind: "TR"})
	}
	if td.Step != "" {
		x = append(x, breadcrumb{name: td.Step, kind: "STEP"})
	}
	return x
}

func TaskRunDetails(outOfBand bool) func(*model.TemplateData) g.Node {
	return func(td *model.TemplateData) g.Node {
		if td.TaskRun == nil {
			return g.Text("you are doing something wrong")
		}
		return RGroup(
			g.If(outOfBand, taskRun(td, true)(td.TaskRun)),
			Div(
				Div(
					Class("text-sm breadcrumbs"),
					Ul(
						g.Map(
							breadcrumbs(td),
							func(p breadcrumb) g.Node {
								return Li(Span(
									Class("font-semibold"),
									Div(Class("badge badge-info me-2"), g.Text(p.kind)),
									g.Text(p.name),
								))
							},
						)...,
					),
				),
				Div(
					StyleAttr("max-height: 30vh; overflow-y: auto"),
					Table(
						Class("table table-zebra table-pin-rows"),
						THead(Tr(Th(g.Text("Name")), Th(g.Text("Value")))),
						TBody(
							g.Map(
								td.TaskRun.Spec.Params,
								func(p pipelinev1beta1.Param) g.Node {
									return Tr(
										Td(
											Class("select-all"),
											g.Text(p.Name),
										),
										Td(
											Class("select-all"),
											g.Text(p.Value.StringVal),
										),
									)
								},
							)...,
						),
					),
				),
				Div(
					StyleAttr("flex-grow: 1;"),
					StepDetailsTabs(td, td.Tab, false),
					Div(
						Class("rounded-lg overflow-clip mt-2"),
						ID("step-details-content"),
					),
				),
			),
		)
	}
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
