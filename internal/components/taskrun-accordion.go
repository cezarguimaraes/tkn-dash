package components

import (
	"io"

	"github.com/cezarguimaraes/tkn-dash/internal/model"
	g "github.com/maragudk/gomponents"
	htmx "github.com/maragudk/gomponents-htmx"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type renders[T any] func(T) g.Node

func taskRun(td *model.TemplateData, outOfBand bool) renders[*pipelinev1beta1.TaskRun] {
	return func(tr *pipelinev1beta1.TaskRun) g.Node {
		if tr == nil {
			return g.Text("this should not happen")
		}
		return Li(
			ID(tr.GetName()),
			g.If(outOfBand, htmx.SwapOOB("true")),
			Details(
				g.Attr("open"),
				Summary(
					Class("font-semibold"),
					g.Text(tr.Spec.TaskRef.Name),
				),
				Ul(
					g.Group(g.Map(tr.Status.Steps, step(td, tr))),
				),
			),
		)
	}
}

func step(
	td *model.TemplateData,
	tr *pipelinev1beta1.TaskRun,
) renders[pipelinev1beta1.StepState] {
	return func(ss pipelinev1beta1.StepState) g.Node {
		success := ss.Terminated != nil &&
			ss.Terminated.ExitCode == 0
		active := td.TaskRun.GetName() == tr.GetName() &&
			td.Step == ss.Name
		var indicator g.Node
		if success {
			indicator = iconFor("Succeeded")
		} else {
			indicator = iconFor("Failed")
		}
		detailsURL := td.URLFor(
			"details-w-step",
			tr.GetNamespace(),
			tr.GetName(),
			ss.Name,
		)
		if td.PipelineRun != nil {
			detailsURL = detailsURL + "?pipelineRun=" + td.PipelineRun.GetName()
		}
		return Li(
			A(
				c.Classes{
					"active": active,
					"my-1":   true,
				},
				htmx.Get(detailsURL),
				htmx.Target("#taskrun-details"),
				htmx.PushURL(stepURL(td, tr.GetName(), ss.Name)),
				htmx.Swap("innerHTML"),
				Span(
					indicator,
				),
				g.Text(ss.Name),
			),
		)
	}
}

type wrap struct {
	f func() g.Node
}

func (ww *wrap) Render(w io.Writer) error {
	return ww.f().Render(w)
}

func TaskRuns(td *model.TemplateData) g.Node {
	return Div(
		ID("details"), StyleAttr("display: flex;"),

		g.If(td.TaskRun != nil, &wrap{func() g.Node {
			return RGroup(
				Div(
					ID("tasks"), Class("ms-3 mt-3"),
					StyleAttr("flex-shrink: 0; min-width: 300px;"),
					Ul(
						Class("menu bg-base-200 rounded-box"),
						g.Group(g.Map(td.TaskRuns, taskRun(td, false))),
					),
				),
				Div(
					ID("taskrun-details"),
					Class("ms-3 mt-3"),
					StyleAttr("flex-grow: 5; height: 100%; display: flex; flex-direction: column;"),
					g.If(td.TaskRun != nil, &wrap{func() g.Node { return TaskRunDetails(false)(td) }}),
				),
			)
		}}),
	)
}

func stepTabURL(data *model.TemplateData, taskRun, step, tab string) string {
	if data.PipelineRun != nil {
		return data.URLFor(
			"list-w-pipe-details-tab",
			data.PipelineRun.GetNamespace(),
			"pipelineruns",
			data.PipelineRun.GetName(),
			taskRun,
			step,
			tab,
		)
	}
	return data.URLFor(
		"list-w-task-details-tab",
		data.TaskRun.GetNamespace(),
		"taskruns",
		taskRun,
		step,
		tab,
	)
}

func stepURL(data *model.TemplateData, taskRun string, step string) string {
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
