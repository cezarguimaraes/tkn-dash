package tekton

import (
	"github.com/cezarguimaraes/tkn-dash/pkg/cache"
	"github.com/labstack/echo/v4"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type Context struct {
	echo.Context

	pr, tr cache.Store

	namespaces []string
}

// TODO: remove namespaces param

func NewMiddleware(pr, tr cache.Store, namespaces []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &Context{
				Context:    c,
				pr:         pr,
				tr:         tr,
				namespaces: namespaces,
			}
			return next(cc)
		}
	}
}

func (c *Context) GetStoreFor(resource string) cache.Store {
	switch resource {
	case "taskruns":
		return c.tr
	case "pipelineruns":
		return c.pr
	}
	return nil
}

func (c *Context) GetTaskRun(namespace, name string) *pipelinev1beta1.TaskRun {
	tr, err := c.tr.Get(namespace, name)
	if err != nil {
		return nil
	}
	return tr.(*pipelinev1beta1.TaskRun)
}

func (c *Context) GetPipelineRun(namespace, name string) *pipelinev1beta1.PipelineRun {
	pr, err := c.pr.Get(namespace, name)
	if err != nil {
		return nil
	}
	return pr.(*pipelinev1beta1.PipelineRun)
}

func (c *Context) GetPipelineTaskRuns(namespace, name string) []*pipelinev1beta1.TaskRun {
	pr := c.GetPipelineRun(namespace, name)
	if pr == nil {
		return nil
	}
	var trs []*pipelinev1beta1.TaskRun
	for _, cr := range pr.Status.ChildReferences {
		trs = append(trs, c.GetTaskRun(namespace, cr.Name))
	}
	// support for < v0.45
	if trMap := pr.Status.TaskRuns; len(trs) == 0 && len(trMap) > 0 {
		taskOrder := map[string]int{}
		for ord, task := range pr.Status.PipelineSpec.Tasks {
			taskOrder[task.Name] = ord
		}
		trs = make([]*pipelinev1beta1.TaskRun, len(taskOrder))
		for name, tr := range trMap {
			ord := taskOrder[tr.PipelineTaskName]
			trs[ord] = c.GetTaskRun(namespace, name)
		}
	}
	return trs
}
