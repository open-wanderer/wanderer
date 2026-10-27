package routes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteTopLevelSyncUsesCanonicalAuthor(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		for _, existing := range []bool{false, true} {
			name := kind + "/new_shell"
			if existing {
				name = kind + "/correct_existing_handle_attribution"
			}
			t.Run(name, func(t *testing.T) {
				f, record := topLevelAuthorFixture(t, kind, existing)
				canonical := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/alice", "last_fetched": time.Now()})
				payload := topLevelAuthorPayload(record, canonical.GetString("iri"))
				calls := serveTopLevelAuthorPayload(t, payload)
				if _, err := syncTopLevelAuthorFixture(t, f, kind, record); err != nil {
					t.Fatal(err)
				}
				stored, err := f.app.FindRecordById(record.Collection(), record.Id)
				if err != nil || stored.GetString("author") != canonical.Id || stored.GetString("name") != "Canonical name" || !stored.GetBool("full_sync_completed") || *calls != 1 {
					t.Fatalf("canonical sync lost its author or fields: record=%v calls=%d error=%v", stored, *calls, err)
				}
			})
		}
	}
}

func TestRemoteTopLevelSyncKeepsExistingAuthorWithoutExpansion(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f, record := topLevelAuthorFixture(t, kind, true)
			serveTopLevelAuthorPayload(t, map[string]any{"id": "canonical-id", "name": "Legacy name"})
			if _, err := syncTopLevelAuthorFixture(t, f, kind, record); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(record.Collection(), record.Id)
			if err != nil || stored.GetString("author") != f.remoteActor.Id || stored.GetString("name") != "Legacy name" {
				t.Fatalf("verified legacy author was replaced: record=%v error=%v", stored, err)
			}
		})
	}
}

func TestRemoteTopLevelSyncRejectsUnboundAttributionBeforeWrites(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		for _, scenario := range []string{"missing_new_author", "foreign_author", "local_author", "foreign_object", "different_object", "missing_correction_identity", "foreign_stored_author_without_identity", "foreign_stored_author_with_foreign_author"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				existing := scenario != "missing_new_author"
				f, record := topLevelAuthorFixture(t, kind, existing)
				canonical := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/alice", "last_fetched": time.Now()})
				foreign := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
				payload := topLevelAuthorPayload(record, canonical.GetString("iri"))
				switch scenario {
				case "missing_new_author":
					delete(payload, "expand")
				case "foreign_author":
					payload["expand"] = map[string]any{"author": map[string]any{"iri": foreign.GetString("iri")}}
				case "local_author":
					payload["expand"] = map[string]any{"author": map[string]any{"iri": f.localActor.GetString("iri")}}
				case "foreign_object":
					payload["iri"] = "https://third.example/api/v1/" + kind + "/canonical-id"
				case "different_object":
					payload["iri"] = "https://remote.example/api/v1/" + kind + "/another-id"
				case "missing_correction_identity":
					delete(payload, "iri")
				case "foreign_stored_author_without_identity", "foreign_stored_author_with_foreign_author":
					record.Set("author", foreign.Id)
					if err := f.app.Save(record); err != nil {
						t.Fatal(err)
					}
					if scenario == "foreign_stored_author_without_identity" {
						delete(payload, "iri")
					} else {
						payload["expand"] = map[string]any{"author": map[string]any{"iri": foreign.GetString("iri")}}
					}
				}
				serveTopLevelAuthorPayload(t, payload)
				writes := 0
				f.app.OnRecordCreate().BindFunc(func(e *core.RecordEvent) error { writes++; return e.Next() })
				f.app.OnRecordUpdate().BindFunc(func(e *core.RecordEvent) error { writes++; return e.Next() })
				beforeAuthor := record.GetString("author")
				if _, err := syncTopLevelAuthorFixture(t, f, kind, record); err == nil {
					t.Fatal("unbound remote attribution was accepted")
				}
				if writes != 0 || record.GetString("author") != beforeAuthor || record.GetString("name") != "Original name" {
					t.Fatalf("rejected sync changed records: writes=%d author=%s name=%s", writes, record.GetString("author"), record.GetString("name"))
				}
				if existing {
					f.assertUnchanged(record, "iri", "author", "name", "full_sync_completed")
				}
			})
		}
	}
}

func TestRemoteTopLevelSyncProtectsCurrentAndHistoricalLocalOwners(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		for _, origin := range []string{"https://local.example", "https://previous.example"} {
			t.Run(kind+"/"+origin, func(t *testing.T) {
				f, record := topLevelAuthorFixture(t, kind, true)
				record.Set("author", f.localActor.Id)
				record.Set("iri", origin+"/api/v1/"+kind+"/"+record.Id)
				if err := f.app.Save(record); err != nil {
					t.Fatal(err)
				}
				calls := serveTopLevelAuthorPayload(t, topLevelAuthorPayload(record, f.remoteActor.GetString("iri")))
				if _, err := syncTopLevelAuthorFixture(t, f, kind, record); err != nil {
					t.Fatal(err)
				}
				if *calls != 0 {
					t.Fatal("locally authored content was fetched from another origin")
				}
				f.assertUnchanged(record, "iri", "author", "name", "full_sync_completed")
			})
		}
	}
}

func topLevelAuthorFixture(t *testing.T, kind string, existing bool) (*remoteSyncFixture, *core.Record) {
	t.Helper()
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.lists.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.BoolField{Name: "needs_full_sync"}, &core.BoolField{Name: "full_sync_completed"})
	if err := f.app.Save(f.lists); err != nil {
		t.Fatal(err)
	}
	collection := f.trails
	if kind == "list" {
		collection = f.lists
	}
	record := core.NewRecord(collection)
	record.Set("iri", "https://remote.example/api/v1/"+kind+"/canonical-id")
	record.Set("author", f.remoteActor.Id)
	record.Set("name", "Original name")
	record.Set("needs_full_sync", true)
	if existing {
		if err := f.app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	return f, record
}

func topLevelAuthorPayload(record *core.Record, authorIRI string) map[string]any {
	return map[string]any{"id": "canonical-id", "iri": record.GetString("iri"), "name": "Canonical name", "expand": map[string]any{"author": map[string]any{"iri": authorIRI}}}
}

func serveTopLevelAuthorPayload(t *testing.T, payload map[string]any) *int {
	t.Helper()
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	calls := 0
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			calls++
			if !slices.Contains(strings.Split(request.URL.Query().Get("expand"), ","), "author") {
				t.Error("canonical author was not requested from the origin")
			}
			body, err := json.Marshal(payload)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}, err
		}}}
	}
	return &calls
}

func syncTopLevelAuthorFixture(t *testing.T, f *remoteSyncFixture, kind string, record *core.Record) (*core.Record, error) {
	t.Helper()
	requestURL := &url.URL{RawQuery: "fields=id,name"}
	if kind == "list" {
		return performFullListSync(f.app, context.Background(), requestURL, record)
	}
	return performFullSync(f.app, context.Background(), requestURL, record)
}

func TestRemoteTopLevelSyncReplacesForeignStoredAuthorWithCanonicalAuthor(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f, record := topLevelAuthorFixture(t, kind, true)
			canonical := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/alice", "last_fetched": time.Now()})
			foreign := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
			record.Set("author", foreign.Id)
			if err := f.app.Save(record); err != nil {
				t.Fatal(err)
			}
			serveTopLevelAuthorPayload(t, topLevelAuthorPayload(record, canonical.GetString("iri")))
			if _, err := syncTopLevelAuthorFixture(t, f, kind, record); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(record.Collection(), record.Id)
			if err != nil || stored.GetString("author") != canonical.Id || stored.GetString("name") != "Canonical name" {
				t.Fatalf("legacy author from another host was not replaced: record=%v error=%v", stored, err)
			}
		})
	}
}

// A refresh failure must not fail a sync whose author the origin confirms
// or whose canonical author is already cached with a consistent identity.
func TestRemoteTopLevelSyncSurvivesFailedAuthorRefresh(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "")
	for _, kind := range []string{"trail", "list"} {
		for _, scenario := range []string{"confirmed_stored_author", "cached_canonical_author", "inconsistent_cached_author"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f, record := topLevelAuthorFixture(t, kind, true)
				f.actors.Fields.Add(&core.TextField{Name: "domain"})
				if err := f.app.Save(f.actors); err != nil {
					t.Fatal(err)
				}
				stale := time.Now().Add(-3 * time.Hour)
				remoteActor, err := f.app.FindRecordById(f.actors, f.remoteActor.Id)
				if err != nil {
					t.Fatal(err)
				}
				remoteActor.Set("domain", "remote.example")
				remoteActor.Set("last_fetched", stale)
				if err := f.app.Save(remoteActor); err != nil {
					t.Fatal(err)
				}
				domain := "remote.example"
				if scenario == "inconsistent_cached_author" {
					domain = "alternate.example"
				}
				canonical := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/alice", "domain": domain, "last_fetched": stale})
				authorIRI := canonical.GetString("iri")
				if scenario == "confirmed_stored_author" {
					authorIRI = f.remoteActor.GetString("iri")
				}
				serveTopLevelAuthorPayload(t, topLevelAuthorPayload(record, authorIRI))
				_, err = syncTopLevelAuthorFixture(t, f, kind, record)
				stored, _ := f.app.FindRecordById(record.Collection(), record.Id)
				switch scenario {
				case "confirmed_stored_author":
					if err != nil || stored.GetString("author") != f.remoteActor.Id || stored.GetString("name") != "Canonical name" {
						t.Fatalf("confirmed author refresh decided the sync: record=%v error=%v", stored, err)
					}
				case "cached_canonical_author":
					if err != nil || stored.GetString("author") != canonical.Id || stored.GetString("name") != "Canonical name" {
						t.Fatalf("cached canonical author was not used: record=%v error=%v", stored, err)
					}
				case "inconsistent_cached_author":
					if err == nil || stored.GetString("author") != f.remoteActor.Id || stored.GetString("name") != "Original name" {
						t.Fatalf("cached author with another host's identity was trusted: record=%v error=%v", stored, err)
					}
				}
			})
		}
	}
}

func TestResolveRemoteSyncAuthorUsesConsistentCachedActorAfterFailedRefresh(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "")
	for _, scenario := range []string{"consistent", "inconsistent"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			f.actors.Fields.Add(&core.TextField{Name: "domain"})
			if err := f.app.Save(f.actors); err != nil {
				t.Fatal(err)
			}
			domain := "remote.example"
			if scenario == "inconsistent" {
				domain = "alternate.example"
			}
			bob := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/bob", "domain": domain, "last_fetched": time.Now().Add(-3 * time.Hour)})
			raw := map[string]any{"expand": map[string]any{"author": map[string]any{"iri": bob.GetString("iri")}}}
			author, allowed, err := resolveRemoteSyncAuthor(f.app, context.Background(), f.remoteActor.Id, "https://remote.example/api/v1/summit-log/sl1", raw)
			if scenario == "consistent" && (err != nil || !allowed || author != bob.Id) {
				t.Fatalf("consistent cached child author was not used: author=%s allowed=%v err=%v", author, allowed, err)
			}
			if scenario == "inconsistent" && (err == nil || allowed) {
				t.Fatalf("child author with another host's identity was trusted: author=%s allowed=%v", author, allowed)
			}
		})
	}
}

func TestRemoteTopLevelSyncAcceptsEmptyIdentityOfTheRequestedRecord(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f, record := topLevelAuthorFixture(t, kind, false)
			canonical := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/alice", "last_fetched": time.Now()})
			payload := topLevelAuthorPayload(record, canonical.GetString("iri"))
			payload["iri"] = ""
			serveTopLevelAuthorPayload(t, payload)
			if _, err := syncTopLevelAuthorFixture(t, f, kind, record); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(record.Collection(), record.Id)
			if err != nil || stored.GetString("author") != canonical.Id || stored.GetString("name") != "Canonical name" {
				t.Fatalf("v0.19 origin response lost its canonical author: record=%v error=%v", stored, err)
			}
		})
	}
}

func TestRemoteTopLevelSyncRejectsAmbiguousLegacyObjectIdentity(t *testing.T) {
	for _, suffix := range []string{"?", "?untrusted=1", "#fragment", "/"} {
		t.Run(suffix, func(t *testing.T) {
			f, record := topLevelAuthorFixture(t, "trail", false)
			record.Set("iri", record.GetString("iri")+suffix)
			payload := topLevelAuthorPayload(record, f.remoteActor.GetString("iri"))
			payload["iri"] = ""
			serveTopLevelAuthorPayload(t, payload)
			if _, err := syncTopLevelAuthorFixture(t, f, "trail", record); err == nil {
				t.Fatal("legacy response accepted an ambiguous object identity")
			}
			stored, err := f.app.FindAllRecords("trails")
			if err != nil || len(stored) != 0 {
				t.Fatalf("unverified legacy object was persisted: count=%d error=%v", len(stored), err)
			}
		})
	}
}
