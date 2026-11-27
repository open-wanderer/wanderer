package routes

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestRemoteFullSyncKeepsOmittedLogWhenItsAuthorChangedAfterConfirmation(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, scenario := range []string{"changed-author", "local-author"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			f.remoteActor.Set("iri", "https://a.example/api/v1/actor/alice")
			f.remoteActor.Set("domain", "a.example")
			if err := f.app.Save(f.remoteActor); err != nil {
				t.Fatal(err)
			}
			f.trail.Set("iri", "https://a.example/api/v1/trail/trail")
			if err := f.app.Save(f.trail); err != nil {
				t.Fatal(err)
			}
			author := f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/carol", "last_fetched": time.Now()})
			newAuthor := f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/erin", "last_fetched": time.Now()})
			iri := "https://a.example:8443/api/v1/summit-log/omitted"
			log := f.child(iri, author.Id, f.trail.Id)
			var confirmations atomic.Int32
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "a.example" {
						return richTextSyncTransport{map[string]any{"id": "trail", "name": "Origin", "public": true,
							"expand": map[string]any{"summit_logs_via_trail": []any{}}}}.RoundTrip(request)
					}
					confirmations.Add(1)
					stored, err := f.app.FindRecordById(f.children, log.Id)
					if err != nil {
						return nil, err
					}
					if scenario == "changed-author" {
						stored.Set("author", newAuthor.Id)
						if err := f.app.Save(stored); err != nil {
							return nil, err
						}
					} else {
						author.Set("is_local", true)
						if err := f.app.Save(author); err != nil {
							return nil, err
						}
					}
					return &http.Response{StatusCode: http.StatusGone, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
				}}}
			}
			if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(f.children, log.Id)
			expectedAuthor := newAuthor.Id
			if scenario == "local-author" {
				expectedAuthor = author.Id
			}
			if err != nil || stored.GetString("author") != expectedAuthor || stored.GetString("trail") != f.trail.Id {
				t.Fatalf("canonical removal deleted a log whose author changed before the sync transaction: record=%v error=%v", stored, err)
			}
			if confirmations.Load() != 1 {
				t.Fatalf("canonical confirmation calls=%d want1", confirmations.Load())
			}
		})
	}
}

func TestRemoteFullSyncKeepsOmittedLogAfterParentChanges(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	author := f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/carol", "last_fetched": time.Now()})
	iri := "https://a.example:8443/api/v1/summit-log/omitted"
	log := f.child(iri, author.Id, f.trail.Id)
	otherTrail := f.save(f.trails, map[string]any{"author": f.remoteActor.Id})
	var confirmations atomic.Int32
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if request.URL.Host == "a.example" {
				return richTextSyncTransport{map[string]any{"id": "trail", "name": "Origin", "public": true,
					"expand": map[string]any{"summit_logs_via_trail": []any{}}}}.RoundTrip(request)
			}
			confirmations.Add(1)
			stored, err := f.app.FindRecordById(f.children, log.Id)
			if err != nil {
				return nil, err
			}
			stored.Set("trail", otherTrail.Id)
			if err := f.app.Save(stored); err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusGone, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
		}}}
	}
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	stored, err := f.app.FindRecordById(f.children, log.Id)
	if err != nil || stored.GetString("trail") != otherTrail.Id || stored.GetString("author") != author.Id {
		t.Fatalf("confirmation deleted an entry moved to another trail: record=%v error=%v", stored, err)
	}
	if confirmations.Load() != 1 {
		t.Fatalf("confirmation calls=%d want1", confirmations.Load())
	}
}
