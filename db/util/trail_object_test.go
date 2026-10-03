package util

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// useTrailObjectPlainClient lets TrailObjectFromIRI reach httptest servers on
// loopback, which SafeHTTPClient refuses.
func useTrailObjectPlainClient(t *testing.T) {
	t.Helper()
	orig := trailObjectHTTPClient
	trailObjectHTTPClient = func() *http.Client { return &http.Client{} }
	t.Cleanup(func() { trailObjectHTTPClient = orig })
}

// The trail fetch uses the caller's context.
func TestTrailObjectFromIRIHonoursContext(t *testing.T) {
	useTrailObjectPlainClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := TrailObjectFromIRI(ctx, server.URL+"/api/v1/trail/slow")
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("TrailObjectFromIRI returned after %v, want within 2s of a 100ms deadline", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

// paddedTrailObject is a valid trail object JSON document of exactly size bytes.
func paddedTrailObject(size int64) string {
	const doc = `{"id":"https://remote.example/api/v1/trail/big","type":"Note"}`
	return doc + strings.Repeat(" ", int(size)-len(doc))
}

// The body is read through a size limit.
func TestTrailObjectFromIRIRefusesOversizedBody(t *testing.T) {
	useTrailObjectPlainClient(t)
	var size int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(paddedTrailObject(size)))
	}))
	defer server.Close()

	size = TrailObjectMaxBytes
	object, err := TrailObjectFromIRI(context.Background(), server.URL+"/api/v1/trail/big")
	if err != nil {
		t.Fatalf("body of exactly TrailObjectMaxBytes refused: %v", err)
	}
	if object.ID.String() != "https://remote.example/api/v1/trail/big" {
		t.Errorf("id = %q", object.ID)
	}

	size = TrailObjectMaxBytes + 1
	if _, err := TrailObjectFromIRI(context.Background(), server.URL+"/api/v1/trail/big"); err == nil {
		t.Error("body of TrailObjectMaxBytes+1 accepted, want it refused")
	}
}
