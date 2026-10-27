package routes

import (
	"net/http"
	"testing"
	"time"
)

func TestCanonicalSummitLogAcceptsOnlySameParentAcrossSchemeChanges(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, candidate := range []struct {
		name, parent string
		allowed      bool
	}{
		{"same", "https://remote.example/api/v1/trail/trail", true},
		{"scheme", "http://remote.example/api/v1/trail/trail", true},
		{"different-path", "https://remote.example/api/v1/trail/other", false},
		{"different-host", "https://another.example/api/v1/trail/trail", false},
		{"different-port", "https://remote.example:8443/api/v1/trail/trail", false},
	} {
		t.Run(candidate.name, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			iri := "https://third.example/api/v1/summit-log/entry"
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					data := map[string]any{"id": "trail", "name": "Updated", "public": true, "expand": map[string]any{"summit_logs_via_trail": []any{map[string]any{"id": "untrusted", "iri": iri, "text": "Cached by parent"}}}}
					if request.URL.Host == "third.example" {
						data = map[string]any{"id": "entry", "iri": iri, "text": "Confirmed contribution", "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": candidate.parent}}}
					}
					return richTextSyncTransport{data}.RoundTrip(request)
				}}}
			}
			if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
				t.Fatal(err)
			}
			logs, err := f.app.FindAllRecords("summit_logs")
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if candidate.allowed {
				want = 1
			}
			if len(logs) != want {
				t.Fatalf("canonical parent binding imported%d contributions, want%d", len(logs), want)
			}
			if want == 1 && (logs[0].GetString("text") != "Confirmed contribution" || logs[0].GetString("author") != author.Id) {
				t.Fatal("canonical fields/author were not retained")
			}
		})
	}
}
