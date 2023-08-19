package cache

import (
	"reflect"
	"testing"
)

func TestUnionContinueTokenDecoding(t *testing.T) {
	s := "WyIyIixudWxsLCIzIl0="
	var tok ContinueToken = &s
	toks, err := decodeContinueToken(tok)
	if err != nil {
		t.Errorf("decodeContinueToken(%q) = (%v, %v), want err to be nil", *tok, toks, err)
	}

	// convert *string to string so we can compare
	gotToks := []string{}
	nilString := "<nil-token>"
	for _, tok := range toks {
		if tok == nil {
			gotToks = append(gotToks, nilString)
			continue
		}
		gotToks = append(gotToks, *tok)
	}

	wantToks := []string{"2", nilString, "3"}
	if !reflect.DeepEqual(wantToks, gotToks) {
		t.Errorf("decodeContinueToken(%q) got %v; want %v", *tok, gotToks, wantToks)
	}
}

func TestUnionContinueTokenEncoding(t *testing.T) {
	two := "2"
	three := "3"
	toks := []ContinueToken{&two, nil, &three}
	s, err := encodeContinueToken(toks)
	if err != nil {
		t.Errorf("encodeContinueToken(), got err %v, want nil", err)
	}
	wantS := "WyIyIixudWxsLCIzIl0="
	if *s != wantS {
		t.Errorf("encodeContinueToken() got %q, want %q", *s, wantS)
	}
}
