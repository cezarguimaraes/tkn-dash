package cache

import (
	"encoding/base64"
	"encoding/json"
)

type union struct {
	stores []Store
}

var _ Store = &union{}

func Union(stores ...Store) *union {
	return &union{stores}
}

func (u *union) Get(namespace, name string) (interface{}, error) {
	var err error
	for _, str := range u.stores {
		var i interface{}
		i, err = str.Get(namespace, name)
		if err == nil {
			return i, nil
		}
	}
	return nil, err
}

func (u *union) Search(opts *SearchOptions) ([]interface{}, ContinueToken, error) {
	ct := make([]ContinueToken, len(u.stores))
	if opts.ContinueFrom != nil {
		var err error
		ct, err = decodeContinueToken(opts.ContinueFrom)
		if err != nil {
			return nil, nil, err
		}
	}

	var agg []interface{}
	cont := false
	for idx, str := range u.stores {
		// not the first page and an individual store
		// already got to nil, so we ignore it instead
		// of restarting its search
		if ct[idx] == nil && opts.ContinueFrom != nil {
			continue
		}

		tmp, nxtCt, err := str.Search(&SearchOptions{
			LabelSelector: opts.LabelSelector,
			Limit:         opts.Limit,
			ContinueFrom:  ct[idx],
			Namespace:     opts.Namespace,
		})
		if err != nil {
			return agg, nil, err
		}

		ct[idx] = nxtCt
		if nxtCt != nil {
			cont = true
		}

		agg = append(agg, tmp...)
	}

	var aggCt ContinueToken
	if cont {
		var err error
		aggCt, err = encodeContinueToken(ct)
		if err != nil {
			return agg, nil, err
		}
	}

	return agg, aggCt, nil
}

func decodeContinueToken(tok ContinueToken) ([]ContinueToken, error) {
	if tok == nil {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(string(*tok))
	if err != nil {
		return nil, err
	}

	var out []ContinueToken
	err = json.Unmarshal(b, &out)
	return out, err
}

func encodeContinueToken(toks []ContinueToken) (ContinueToken, error) {
	js, err := json.Marshal(toks)
	if err != nil {
		return nil, err
	}
	str := base64.StdEncoding.EncodeToString(js)
	return &str, nil
}
