package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fileCache[T metav1.Object] struct {
	items   []interface{}
	nameMap map[string]interface{}
}

func FromFile[T metav1.Object](path string) (*fileCache[T], error) {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out struct {
		// parse as concrete type T but store as interface{},
		// otherwise we need to create interface{} slices
		// on every search
		Items []T `json:"items"`
	}

	dec := json.NewDecoder(f)
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}

	sort.Slice(out.Items, func(i, j int) bool {
		return out.Items[i].GetCreationTimestamp().
			Sub(out.Items[j].GetCreationTimestamp().Time) > 0
	})

	nameMap := make(map[string]interface{}, len(out.Items))
	items := make([]interface{}, 0, len(out.Items))
	for _, it := range out.Items {
		key := fmt.Sprintf("%s/%s", it.GetNamespace(), it.GetName())
		nameMap[key] = it
		items = append(items, it)
	}

	return &fileCache[T]{items, nameMap}, nil
}

func (s *fileCache[T]) Get(namespace, name string) (interface{}, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	found, ok := s.nameMap[key]
	if !ok {
		return nil, errors.New("key not found")
	}
	return found, nil
}

func (s *fileCache[T]) Search(opts *SearchOptions) ([]interface{}, ContinueToken, error) {
	return sliceSearch(s.items, opts)
}
