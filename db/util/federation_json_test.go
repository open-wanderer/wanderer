package util

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type federationCountingReader struct {
	io.Reader
	read int64
}

func (r *federationCountingReader) Read(p []byte) (int, error) {
	n, e := r.Reader.Read(p)
	r.read += int64(n)
	return n, e
}

type federationDecodeSpy struct{ decoded bool }

func (s *federationDecodeSpy) UnmarshalJSON([]byte) error { s.decoded = true; return nil }

func TestFederationJSONRejectsOversizeBeforeAllocatingObjects(t *testing.T) {
	for _, body := range []string{`{"values":[` + strings.Repeat("0,", 1<<20) + `0]}`, `{}` + strings.Repeat(" ", 2<<20)} {
		reader := &federationCountingReader{Reader: strings.NewReader(body)}
		spy := &federationDecodeSpy{}
		err := DecodeFederationJSON(reader, FederationActorBodyLimit, spy)
		if !errors.Is(err, ErrFederationBodyTooLarge) || spy.decoded || reader.read != FederationActorBodyLimit+1 {
			t.Fatalf("oversize decoded=%v read=%d err=%v", spy.decoded, reader.read, err)
		}
	}
}

func TestFederationJSONRequiresCompleteSingleDocument(t *testing.T) {
	for _, tc := range []struct {
		body  string
		limit int64
		valid bool
	}{
		{body: `{"id":"ok"}`, limit: 11, valid: true},
		{body: `{"id":"ok"}`, limit: 10},
		{body: `{"id":"ok"} {}`, limit: 20},
		{body: `{"id":"ok"`, limit: 20},
		{body: `{"id":"ok"}`, limit: 0},
		{body: `{"id":"ok"}`, limit: -1},
	} {
		var data map[string]any
		err := DecodeFederationJSON(strings.NewReader(tc.body), tc.limit, &data)
		if tc.valid != (err == nil) {
			t.Fatalf("body %q limit %d: %v", tc.body, tc.limit, err)
		}
	}
}
