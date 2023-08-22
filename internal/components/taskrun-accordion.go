package components

import (
	"io"

	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	g "github.com/maragudk/gomponents"
	htmx "github.com/maragudk/gomponents-htmx"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type renders[T any] func(T) g.Node

func taskRun(td *tekton.TemplateData) renders[*pipelinev1beta1.TaskRun] {
	return func(tr *pipelinev1beta1.TaskRun) g.Node {
		return Li(
			Details(
				g.Attr("open"),
				Summary(
					g.Text(tr.Spec.TaskRef.Name),
				),
				Ul(
					g.Group(g.Map(tr.Status.Steps, step(td, tr))),
				),
			),
		)
		return Div(
			Class("accordion-item"),
			H2(
				Class("accordion-header"),
				Button(
					c.Classes{
						"accordion-button": true,
						"collapsed":        td.TaskRun.GetName() != tr.GetName(),
					},
					Type("button"),
					DataAttr("bs-toggle", "collapse"),
					DataAttr("bs-target", "#"+tr.GetName()),
					Span(
						Class("me-2"),
						g.Text(tr.Spec.TaskRef.Name),
					),
				),
			),
			Div(
				ID(tr.GetName()),
				c.Classes{
					"accordion-collapse": true,
					"collapse":           true,
					"show":               td.TaskRun.GetName() == tr.GetName(),
				},
				DataAttr("bs-parent", "#accordion"),
				Ul(
					Class("list-group-flush list-group"),
					g.Group(g.Map(tr.Status.Steps, step(td, tr))),
				),
			),
		)
	}
}

func step(
	td *tekton.TemplateData,
	tr *pipelinev1beta1.TaskRun,
) renders[pipelinev1beta1.StepState] {
	return func(ss pipelinev1beta1.StepState) g.Node {
		success := ss.Terminated != nil &&
			ss.Terminated.ExitCode == 0
		active := td.TaskRun.GetName() == tr.GetName() &&
			td.Step == ss.Name
		return Li(
			A(
				c.Classes{
					"active": active,
				},
				htmx.Get(
					td.URLFor(
						"details-w-step",
						tr.GetNamespace(),
						tr.GetName(),
						ss.Name,
					),
				),
				htmx.Target("#taskrun-details"),
				htmx.PushURL(stepURL(td, tr.GetName(), ss.Name)),
				htmx.Swap("innerHTML"),
				g.Text(ss.Name),
			),
		)
		return A(
			Href("#"),
			c.Classes{
				"list-group-item":        true,
				"list-group-item-action": true,
				"text-bold":              true,
				"text-success":           success,
				"text-danger":            !success,
				"active":                 active,
			},
			DataAttr("bs-toggle", "list"),
			htmx.Get(
				td.URLFor(
					"details-w-step",
					tr.GetNamespace(),
					tr.GetName(),
					ss.Name,
				),
			),
			htmx.Target("#taskrun-details"),
			htmx.PushURL(stepURL(td, tr.GetName(), ss.Name)),
			htmx.Swap("innerHTML"),
			g.Text(ss.Name),
		)
	}
}

type wrap struct {
	f func() g.Node
}

func (ww *wrap) Render(w io.Writer) error {
	return ww.f().Render(w)
}

func TaskRuns(td *tekton.TemplateData) g.Node {
	return Div(
		ID("details"), StyleAttr("display: flex;"),

		Div(
			ID("tasks"), Class("ms-3 mt-3"),
			StyleAttr("flex-shrink: 0; min-width: 300px;"),
			Ul(
				Class("menu bg-base-200 rounded-box"),
				g.Group(g.Map(td.TaskRuns, taskRun(td))),
			),
		),
		Div(
			ID("taskrun-details"),
			Class("ms-3 mt-3"),
			StyleAttr("flex-grow: 5; height: 100%; display: flex; flex-direction: column;"),
		),
	)
	return Div(
		ID("details"), StyleAttr("display: flex;"),
		Div(
			ID("tasks"), Class("ms-3 mt-3"),
			StyleAttr("flex-shrink: 0; min-width: 300px;"),
			Div(
				Class("accordion"), ID("taskrun-accordion"),
				g.Group(g.Map(td.TaskRuns, taskRun(td))),
			),
		),
		Div(
			ID("taskrun-details"),
			Class("ms-3 mt-3"),
			StyleAttr("flex-grow: 5; height: 100%; display: flex; flex-direction: column;"),
			g.If(
				td.TaskRun != nil && td.Step != "",
				&wrap{func() g.Node {
					return Div(
						htmx.Get(
							td.URLFor(
								"details-w-step",
								td.TaskRun.GetNamespace(),
								td.TaskRun.GetName(),
								td.Step,
							),
						),
						htmx.Trigger("load"),
						htmx.Swap("outerHTML"),
					)
				}},
			),
		),
	)
}

func stepURL(data *tekton.TemplateData, taskRun string, step string) string {
	if data.PipelineRun != nil {
		return data.URLFor(
			"list-w-pipe-details",
			data.PipelineRun.GetNamespace(),
			"pipelineruns",
			data.PipelineRun.GetName(),
			taskRun,
			step,
		)
	}
	return data.URLFor(
		"list-w-task-details",
		data.TaskRun.GetNamespace(),
		"taskruns",
		taskRun,
		step,
	)
}
