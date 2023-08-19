package cache

import (
	"errors"
	"sort"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type sharedInformerCache struct {
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
) (*sharedInformerCache, func()) {
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
	return &sharedInformerCache{lw, si, closeCh}, stopFunc
}

func (s *sharedInformerCache) Get(namespace, name string) (interface{}, error) {
	it, exists, err := s.si.GetStore().GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("key does not exist")
	}
	return it, nil
}

func (s *sharedInformerCache) Search(opts *SearchOptions) ([]interface{}, ContinueToken, error) {
	if opts.Limit == 0 {
		opts.Limit = 100
	}
	if opts.LabelSelector == nil {
		opts.LabelSelector = labels.Everything()
	}

	// TODO: continue from based on item key/order
	// not position as items might change inbetween calls
	from := 0
	if opts.ContinueFrom != nil {
		var err error
		from, err = strconv.Atoi(*opts.ContinueFrom)
		if err != nil {
			return nil, nil, err
		}
	}

	items := s.si.GetStore().List()

	if from >= len(items) {
		return nil, nil, nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].(metav1.Object).GetCreationTimestamp().
			Sub(items[j].(metav1.Object).GetCreationTimestamp().Time) > 0
	})

	var at int
	res := make([]interface{}, 0, opts.Limit)
	for at = from; at < len(items); at++ {
		if len(res) >= opts.Limit {
			break
		}

		obj := items[at].(metav1.Object)
		if opts.Namespace != nil && *opts.Namespace != obj.GetNamespace() {
			continue
		}

		if !opts.LabelSelector.Matches(labels.Set(obj.GetLabels())) {
			continue
		}

		res = append(res, obj)
	}

	continueFrom := strconv.Itoa(at)
	return res, &continueFrom, nil
}
