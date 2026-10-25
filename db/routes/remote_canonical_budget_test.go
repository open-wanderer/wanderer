package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type canonicalCountedBody struct {
	io.Reader
	bytes, closed *atomic.Int64
}

func (b canonicalCountedBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.bytes.Add(int64(n))
	return n, err
}
func (b canonicalCountedBody) Close() error { b.closed.Add(1); return nil }

func TestCanonicalContributionsBoundBodiesAndRetainedSchema(t *testing.T) {
	for _, kind := range []string{"summit-log", "comment", "note"} {
		for _, scenario := range []string{"body-limit", "schema", "aggregate"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				t.Setenv("ORIGIN", "https://local.example")
				f := newRemoteChildFixture(t, "summit_log")
				actor := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
				count := 1
				if scenario == "aggregate" {
					count = 30
				}
				padding := make([]any, 40000)
				for i := range padding {
					padding[i] = map[string]any{}
				}
				var bytes, closed, responses atomic.Int64
				original := newRemoteSyncHTTPClient
				t.Cleanup(func() { newRemoteSyncHTTPClient = original })
				newRemoteSyncHTTPClient = func() *http.Client {
					return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
						responses.Add(1)
						id := strings.TrimPrefix(request.URL.Path, "/api/v1/"+kind+"/")
						iri := "https://third.example" + request.URL.Path
						payload := map[string]any{"id": id, "iri": iri, "text": "Confirmed", "padding": padding, "expand": map[string]any{"author": map[string]any{"iri": actor.GetString("iri")}, "trail": map[string]any{"iri": f.trail.GetString("iri")}}}
						if kind == "note" {
							payload = map[string]any{"id": iri, "type": "Note", "content": "Confirmed", "attributedTo": actor.GetString("iri"), "inReplyTo": f.trail.GetString("iri"), "to": "https://www.w3.org/ns/activitystreams#Public", "padding": padding}
						}
						wire, err := json.Marshal(payload)
						if err != nil {
							return nil, err
						}
						if scenario == "body-limit" {
							wire = append(wire, []byte(strings.Repeat(" ", int(canonicalRemoteBodyLimit)))...)
						}
						return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: canonicalCountedBody{Reader: strings.NewReader(string(wire)), bytes: &bytes, closed: &closed}}, nil
					}}}
				}
				entries := []any{}
				for i := range count {
					iri := fmt.Sprintf("https://third.example/api/v1/%s/%d", kind, i)
					entries = append(entries, map[string]any{"id": fmt.Sprint(i), "iri": iri})
				}
				data := []map[string]any{}
				if kind == "summit-log" {
					for _, value := range fetchCanonicalRemoteSummitLogs(f.app, t.Context(), map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}, f.trail, "https://remote.example") {
						data = append(data, value.data)
					}
				} else {
					references := []canonicalCommentReference{}
					for _, entry := range entries {
						references = append(references, canonicalCommentReference{iri: entry.(map[string]any)["iri"].(string)})
					}
					for _, value := range fetchCanonicalRemoteComments(f.app, t.Context(), newRemoteSyncHTTPClient(), f.trail, references) {
						data = append(data, value.data)
					}
				}
				if responses.Load() != closed.Load() {
					t.Fatalf("response body leak: responses%d closed%d", responses.Load(), closed.Load())
				}
				switch scenario {
				case "body-limit":
					if len(data) != 0 || bytes.Load() > canonicalRemoteBodyLimit+1 {
						t.Fatalf("oversized contribution accepted or over-read: count%d bytes%d", len(data), bytes.Load())
					}
				case "schema":
					if len(data) != 1 || data[0]["text"] != "Confirmed" {
						t.Fatalf("valid contribution lost: %#v", data)
					}
				case "aggregate":
					if len(data) == 0 || len(data) >= count || bytes.Load() > canonicalRemoteByteBudget+1 {
						t.Fatalf("aggregate budget ineffective: count%d bytes%d", len(data), bytes.Load())
					}
				}
				for _, record := range data {
					if _, retained := record["padding"]; retained {
						t.Fatal("retained unrelated nested response objects")
					}
				}
			})
		}
	}
}
