package util

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type trailObjectBodyTransport struct {
	body   *trailObjectBody
	status int
}

func (t trailObjectBodyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: t.status, ContentLength: -1, Header: make(http.Header), Body: t.body, Request: r}, nil
}

type trailObjectBody struct {
	io.Reader
	read   int64
	closed bool
}

func (b *trailObjectBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.read += int64(n)
	return n, e
}
func (b *trailObjectBody) Close() error { b.closed = true; return nil }

func TestTrailObjectFromIRIBoundsWholeResponse(t *testing.T) {
	for _, oversize := range []bool{false, true} {
		document := `{"id":"https://remote.example/api/v1/trail/t","type":"Place"}`
		if oversize {
			document += strings.Repeat(" ", int(FederationObjectBodyLimit))
		}
		body := &trailObjectBody{Reader: strings.NewReader(document)}
		original := newTrailObjectHTTPClient
		newTrailObjectHTTPClient = func() *http.Client { return &http.Client{Transport: trailObjectBodyTransport{body: body, status: 200}} }
		object, err := TrailObjectFromIRI("https://remote.example/api/v1/trail/t")
		newTrailObjectHTTPClient = original
		if !body.closed {
			t.Fatal("response was not closed")
		}
		if oversize {
			if !errors.Is(err, ErrFederationBodyTooLarge) || object != nil || body.read != FederationObjectBodyLimit+1 {
				t.Fatalf("oversize object=%v read=%d err=%v", object, body.read, err)
			}
		} else if err != nil || object == nil {
			t.Fatalf("small object refused: %v", err)
		}
	}
}
