package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cezarguimaraes/tkn-dash/internal/tekton"
	"github.com/cezarguimaraes/tkn-dash/pkg/cache"
	"github.com/labstack/echo/v4"
	"github.com/maragudk/gomponents"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/apis"
)

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

type renderer func(SearchResults) []gomponents.Node

func Search(r renderer) echo.HandlerFunc {
	return func(c echo.Context) error {
		tc := c.(*tekton.Context)

		ns := c.QueryParam("namespace")
		resource := c.Param("resource")

		if c.Request().Header.Get("HX-Trigger-Name") == "namespace" {
			// ensure we update the user's URL and history
			// when they select a namespace
			c.Response().Header().Set(
				"HX-Push-Url",
				c.Echo().Reverse("list", ns, resource),
			)
		}

		str := tc.GetStoreFor(resource)

		var ls labels.Selector
		if search := c.QueryParam("search"); search != "" {
			var err error
			ls, err = labels.Parse(search)
			if err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		}

		opts := &cache.SearchOptions{
			Limit:         100,
			LabelSelector: ls,
		}
		if ns != "" {
			opts.Namespace = &ns
		}
		if pageStr := c.QueryParam("page"); pageStr != "" {
			opts.ContinueFrom = &pageStr
		}

		results, continueFrom, err := str.Search(opts)
		if err != nil {
			return err
		}

		items := make([]SearchItem, 0, len(results))

		now := time.Now()
		for i, r := range results {
			nextPage := ""
			if i+1 == len(results) && continueFrom != nil {
				// include continue token plus any incoming
				// search params
				qs := c.QueryParams()
				qs.Set("page", *continueFrom)

				nextPage = c.Echo().Reverse("items", resource) +
					"?" + qs.Encode()
			}

			var status string
			if st, ok := r.(statusConditionAccessor); ok {
				cond := st.GetStatusCondition().
					GetCondition(apis.ConditionSucceeded)

				if cond != nil {
					if cond.IsUnknown() {
						status = "Running"
					}
					if cond.IsFalse() {
						status = "Failed"
					}
					if cond.IsTrue() {
						status = "Succeeded"
					}
				}
			}

			obj := r.(metav1.Object)
			items = append(items, SearchItem{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
				NextPage:  nextPage,
				Status:    status,
				Age: ageString(
					now.Sub(obj.GetCreationTimestamp().Time),
				) + " ago",
			})

		}

		sr := SearchResults{
			Resource: resource,
			Items:    items,
			URLFor:   c.Echo().Reverse,
		}

		for _, it := range r(sr) {
			if err := it.Render(c.Response()); err != nil {
				return err
			}
		}
		return nil
	}
}

var (
	thresholds = [4]time.Duration{time.Second, time.Minute, time.Hour, 24 * time.Hour}
	suffixes   = [4]string{"s", "m", "h", "d"}
)

func ageString(d time.Duration) (res string) {
	res = "1" + suffixes[0]
	for i, dur := range thresholds {
		if dur > d {
			return
		}
		res = strconv.Itoa(int((d+dur-1)/dur)) + suffixes[i] // round up
	}
	return
}

// statusConditionAccessor allows casting both PipelineRun and StatusRun
// down from metav1.Object to a shared interface which allows us to extract
// their Status condition.
type statusConditionAccessor interface {
	GetStatusCondition() apis.ConditionAccessor
}
