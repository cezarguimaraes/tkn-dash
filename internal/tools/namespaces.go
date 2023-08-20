package tools

import (
	"context"

	"github.com/cezarguimaraes/tkn-dash/pkg/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceLister interface {
	List(context.Context) ([]string, error)
}

type storeLister struct {
	stores []cache.Store
}

func NamespaceListerFromStore(stores ...cache.Store) *storeLister {
	return &storeLister{stores}
}

func (l *storeLister) List(ctx context.Context) ([]string, error) {
	set := map[string]interface{}{}
	for _, s := range l.stores {
		ls, _, err := s.Search(&cache.SearchOptions{
			Limit: -1,
		})
		if err != nil {
			return nil, err
		}
		for _, i := range ls {
			m := i.(metav1.Object)
			set[m.GetNamespace()] = struct{}{}
		}
	}
	ns := make([]string, 0, len(set))
	for k := range set {
		ns = append(ns, k)
	}
	return ns, nil
}
