package cache

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func sliceSearch(items []interface{}, opts *SearchOptions) ([]interface{}, ContinueToken, error) {
	if opts.Limit == 0 {
		opts.Limit = 100
	}
	if opts.LabelSelector == nil {
		opts.LabelSelector = labels.Everything()
	}

	// TODO: continue from based on item key/order
	// not position as items might change between calls
	from := 0
	if opts.ContinueFrom != nil {
		var err error
		from, err = strconv.Atoi(*opts.ContinueFrom)
		if err != nil {
			return nil, nil, err
		}
	}

	if from >= len(items) {
		return nil, nil, nil
	}

	resCap := 64
	if opts.Limit > 0 {
		resCap = opts.Limit
	}
	res := make([]interface{}, 0, resCap)
	var at int
	for at = from; at < len(items); at++ {
		if opts.Limit > 0 && len(res) >= opts.Limit {
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
