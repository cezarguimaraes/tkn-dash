package cache

import "k8s.io/apimachinery/pkg/labels"

type Store interface {
	Get(namespace, name string) (interface{}, error)

	Search(*SearchOptions) ([]interface{}, ContinueToken, error)
}

type ContinueToken *string

type SearchOptions struct {
	ContinueFrom  ContinueToken
	Limit         int
	LabelSelector labels.Selector
	Namespace     *string
}
