package routes

import (
	"net/http"
	"testing"
	"time"
)

func TestRemoteTrailLoadsNewThirdInstanceLogsOnlyFromTheirCanonicalOrigin(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, scenario := range []string{"canonical", "wrong-parent", "wrong-author-origin", "known-copy", "legacy-contributor", "local-iri"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			third := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			iri := "https://third.example/api/v1/summit-log/original"
			if scenario == "local-iri" {
				iri = "https://local.example/api/v1/summit-log/original"
			}
			if scenario == "legacy-contributor" {
				f.child(iri, f.remoteActor.Id, f.trail.Id)
			}
			if scenario == "known-copy" {
				f.child(iri, third.Id, f.trail.Id)
			}
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			canonicalFetches := 0
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					payload := map[string]any{"id": "trail", "name": "Origin trail", "public": true, "expand": map[string]any{
						"summit_logs_via_trail": []any{map[string]any{"id": "untrusted-copy-id", "iri": iri, "text": "Untrusted copy", "expand": map[string]any{"author": map[string]any{"iri": third.GetString("iri")}}}},
					}}
					if request.URL.Host == "third.example" {
						canonicalFetches++
						if request.URL.Path != "/api/v1/summit-log/original" {
							t.Fatalf("wrong canonical object path: %s", request.URL)
						}
						parentIRI, authorIRI := f.trail.GetString("iri"), third.GetString("iri")
						if scenario == "wrong-parent" {
							parentIRI = "https://remote.example/api/v1/trail/another"
						}
						if scenario == "wrong-author-origin" {
							authorIRI = f.remoteActor.GetString("iri")
						}
						payload = map[string]any{"id": "original", "iri": iri, "text": "Canonical content", "expand": map[string]any{"trail": map[string]any{"iri": parentIRI}, "author": map[string]any{"iri": authorIRI}}}
					}
					return richTextSyncTransport{payload}.RoundTrip(request)
				}}}
			}
			if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
				t.Fatal(err)
			}
			logs, err := f.app.FindAllRecords("summit_logs")
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "canonical", "legacy-contributor", "known-copy":
				if len(logs) != 1 || logs[0].GetString("text") != "Canonical content" || logs[0].GetString("author") != third.Id || logs[0].GetString("iri") != iri || canonicalFetches != 1 {
					t.Fatalf("canonical contributor log missing: logs=%v fetches=%d", logs, canonicalFetches)
				}
			case "local-iri":
				if len(logs) != 0 || canonicalFetches != 0 {
					t.Fatal("local object echo was imported or fetched")
				}
			default:
				if len(logs) != 0 || canonicalFetches != 1 {
					t.Fatalf("invalid canonical entry imported: logs=%v fetches=%d", logs, canonicalFetches)
				}
			}
		})
	}
}
