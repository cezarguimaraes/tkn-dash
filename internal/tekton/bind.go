package tekton

import (
	"golang.org/x/exp/slices"

	"github.com/cezarguimaraes/tkn-dash/internal/model"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func (c *Context) BindTemplateData(td *model.TemplateData) error {
	td.URLFor = c.Context.Echo().Reverse

	// TODO: use echo Bind() for param extraction
	// TODO: maybe run this on the middleware when all routes use template data
	td.Namespaces = c.opts.namespaces

	for _, pn := range c.ParamNames() {
		switch pn {
		case "namespace":
			td.Namespace = c.Param(pn)
			// Allows one to choose a namespace not found through
			// the k8s client. Probably only useful for fileStorage
			// usage in which case this could be loaded from the parsed
			// files.
			if !slices.Contains(td.Namespaces, td.Namespace) {
				td.Namespaces = append(td.Namespaces, td.Namespace)
			}
		case "resource":
			// TODO: validate resource
			td.Resource = c.Param(pn)
		case "name":
			if td.Resource == "taskruns" {
				tr := c.GetTaskRun(td.Namespace, c.Param(pn))
				td.TaskRun = tr
				if tr != nil {
					td.TaskRuns = append(td.TaskRuns, tr)
				}
				break
			}

			fallthrough
		case "pipelineRun":
			prName := c.Param(pn)
			td.PipelineRun = c.GetPipelineRun(td.Namespace, prName)
			td.TaskRuns = c.GetPipelineTaskRuns(td.Namespace, prName)
		case "taskRun":
			tr := c.GetTaskRun(td.Namespace, c.Param(pn))
			td.TaskRun = tr
			if tr != nil && len(td.TaskRuns) == 0 {
				td.TaskRuns = append(td.TaskRuns, tr)
			}
		case "step":
			td.Step = c.Param(pn)
		case "tab":
			td.Tab = c.Param(pn)
		}
	}

	for pn := range c.QueryParams() {
		switch pn {
		case "namespace":
			td.Namespace = c.QueryParam(pn)
		case "step":
			td.Step = c.QueryParam(pn)
		case "task":
			td.TaskRun = c.GetTaskRun(td.Namespace, c.QueryParam(pn))
			td.TaskRuns = []*pipelinev1beta1.TaskRun{td.TaskRun}
		case "pipelineRun":
			prName := c.QueryParam(pn)
			td.PipelineRun = c.GetPipelineRun(td.Namespace, prName)
			td.TaskRuns = c.GetPipelineTaskRuns(td.Namespace, prName)
		}
	}

	// auto select first taskRun / step
	if td.Step == "" && len(td.TaskRuns) > 0 {
		if td.TaskRun == nil {
			td.TaskRun = td.TaskRuns[0]
		}
		td.Step = td.TaskRuns[0].Status.TaskSpec.Steps[0].Name
	}

	if log := c.Log.V(4); log.Enabled() {
		tr := ""
		if td.TaskRun != nil {
			tr = td.TaskRun.GetName()
		}
		pr := ""
		if td.PipelineRun != nil {
			pr = td.PipelineRun.GetName()
		}

		trs := []string{}
		for _, t := range td.TaskRuns {
			if t == nil {
				trs = append(trs, "<nil>")
				continue
			}
			trs = append(trs, t.GetName())
		}

		c.Log.V(4).Info("bound values",
			"resource", td.Resource,
			"namespaces", td.Namespaces,
			"taskRun", tr,
			"pipelineRun", pr,
			"taskRuns", trs,
		)
	}

	return nil
}
