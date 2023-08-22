package model

import (
	"github.com/maragudk/gomponents"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type TektonComponent func(*TemplateData) gomponents.Node

type TemplateData struct {
	// Namespaces lists all namespaces found.
	Namespaces []string

	// Namespace specifies which namespace we are working in currently.
	Namespace string

	// Resource is the root object for this page, taskruns/pipelineruns
	Resource string

	// PipelineRun is resolved from the :pipelineRun url param
	PipelineRun *pipelinev1beta1.PipelineRun

	// TaskRun is resolved from the :taskRun url param
	TaskRun *pipelinev1beta1.TaskRun

	// TaskRuns is the list of taskRuns that should be rendered
	// in the middle "step view". It is either a list containing
	// a single taskRun in taskRun view, or the list of taskRuns
	// pertaining to a pipelineRUn
	TaskRuns []*pipelinev1beta1.TaskRun

	// Step is the name of the step resolved from the :step url param
	Step string

	URLFor func(name string, args ...interface{}) string
}

type SearchItem struct {
	Namespace string
	Name      string
	Age       string
	Status    string
	NextPage  string
}

type SearchResults struct {
	Resource string
	Items    []SearchItem
	URLFor   func(string, ...interface{}) string
}
