package loader

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cezarguimaraes/tkn-dash/pkg/cache"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func loadFile(path, kind string) (cache.Store, error) {
	switch kind {
	case "taskrun":
		return cache.FromFile[*pipelinev1.TaskRun](path)
	case "pipelinerun":
		return cache.FromFile[*pipelinev1.PipelineRun](path)
	default:
		return nil, fmt.Errorf("unknown kind: %q", kind)
	}
}

func LoadLocalLists(paths ...string) (map[string]cache.Store, error) {
	storeMap := map[string][]cache.Store{}

	for _, p := range paths {

		f, err := os.OpenFile(p, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return nil, err
		}

		tmp := map[string]interface{}{}

		dec := json.NewDecoder(f)
		if err := dec.Decode(&tmp); err != nil {
			return nil, err
		}

		out := unstructured.Unstructured{}
		out.SetUnstructuredContent(tmp)

		var kind string
		out.EachListItem(func(obj runtime.Object) error {
			kind = obj.GetObjectKind().GroupVersionKind().Kind
			// return an error to stop iteration
			return errors.New("")
		})

		kind = strings.ToLower(kind)
		str, err := loadFile(p, kind)
		if err != nil {
			return nil, err
		}

		storeMap[kind] = append(storeMap[kind], str)
	}

	unionMap := map[string]cache.Store{}
	for k, stores := range storeMap {
		unionMap[k] = cache.Union(stores...)
	}

	return unionMap, nil
}
