package cache

import (
	"errors"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type SharedInformerCache struct {
	lw      cache.ListerWatcher
	si      cache.SharedInformer
	closeCh chan struct{}
}

// TODO: warning whenever there is a List() before
// the shared informer HasSynced()

func NewSharedInformerCache(
	getter cache.Getter,
	resource string,
	exampleObject runtime.Object,
) (*SharedInformerCache, func()) {
	lw := cache.NewListWatchFromClient(
		getter,
		resource,
		"",
		fields.Everything(),
	)

	si := cache.NewSharedInformer(lw, exampleObject, 5*time.Minute)
	closeCh := make(chan struct{})
	go si.Run(closeCh)

	stopFunc := func() {
		close(closeCh)
	}
	return &SharedInformerCache{lw, si, closeCh}, stopFunc
}

func (s *SharedInformerCache) HasSynced() bool {
	return s.si.HasSynced()
}

func (s *SharedInformerCache) Get(namespace, name string) (interface{}, error) {
	it, exists, err := s.si.GetStore().GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("key does not exist")
	}
	return it, nil
}

func (s *SharedInformerCache) Search(opts *SearchOptions) ([]interface{}, ContinueToken, error) {
	items := s.si.GetStore().List()

	sort.Slice(items, func(i, j int) bool {
		return items[i].(metav1.Object).GetCreationTimestamp().
			Sub(items[j].(metav1.Object).GetCreationTimestamp().Time) > 0
	})

	return sliceSearch(items, opts)
}
