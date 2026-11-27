package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func newRemoteCleanupFixture(t *testing.T, target string) *remoteChildFixture {
	t.Helper()
	f := newRemoteChildFixture(t, target)
	f.remoteActor.Set("iri", "https://a.example/users/alice")
	f.remoteActor.Set("domain", "a.example")
	if err := f.app.Save(f.remoteActor); err != nil {
		t.Fatal(err)
	}
	f.trail.Set("iri", "https://a.example/api/v1/trail/trail")
	if err := f.app.Save(f.trail); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestRemoteSyncPrunesOnlyMissingOriginChildren(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		t.Run(target, func(t *testing.T) {
			f := newRemoteCleanupFixture(t, target)
			kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
			relation := map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[target]
			removed := f.child("https://a.example/api/v1/"+kind+"/removed", f.remoteActor.Id, f.trail.Id)
			kept := f.child("https://a.example/api/v1/"+kind+"/kept", f.remoteActor.Id, f.trail.Id)
			local := f.child("https://b.example/api/v1/"+kind+"/local", f.localActor.Id, f.trail.Id)
			foreign := f.child("https://a.example:8443/api/v1/"+kind+"/contribution", f.remoteActor.Id, f.trail.Id)
			localAuthor := f.child("https://a.example/api/v1/"+kind+"/local-author", f.localActor.Id, f.trail.Id)
			otherTrail := f.save(f.trails, map[string]any{"author": f.remoteActor.Id})
			other := f.child("https://a.example/api/v1/"+kind+"/other", f.remoteActor.Id, otherTrail.Id)
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			entries := []any{map[string]any{"id": "kept", "iri": kept.GetString("iri"), "name": "Updated", "text": "Updated"}}
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{relation: entries}}}}
			}
			for round := 0; round < 2; round++ {
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatal(err)
				}
				if _, err := f.app.FindRecordById(f.children, removed.Id); !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("missing origin child survived: %v", err)
				}
				if round == 0 {
					if _, err := f.app.FindRecordById(f.children, kept.Id); err != nil {
						t.Fatalf("listed child disappeared: %v", err)
					}
				} else if _, err := f.app.FindRecordById(f.children, kept.Id); !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("last origin child survived explicit empty proof: %v", err)
				}
				for _, record := range []*core.Record{local, foreign, localAuthor, other} {
					if _, err := f.app.FindRecordById(f.children, record.Id); err != nil {
						t.Fatalf("independent child deleted: %v", err)
					}
				}
				entries = []any{}
			}
		})
	}
}

func TestRemoteSyncKeepsChildrenWhenExpandIsInconclusive(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		for _, scenario := range []string{"omitted", "failed", "malformed", "missing-id", "oversized-id", "relative-iri", "malformed-iri", "non-http-iri", "userinfo-iri", "hostless-iri", "saturated", "capped"} {
			t.Run(target+"/"+scenario, func(t *testing.T) {
				f := newRemoteCleanupFixture(t, target)
				kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
				relation := map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[target]
				retained := f.child("https://a.example/api/v1/"+kind+"/retained", f.remoteActor.Id, f.trail.Id)
				payload := map[string]any{"id": "trail", "public": true}
				switch scenario {
				case "malformed":
					payload["expand"] = map[string]any{relation: map[string]any{}}
				case "missing-id":
					payload["expand"] = map[string]any{relation: []any{map[string]any{"name": "No id", "text": "No id"}}}
				case "oversized-id":
					payload["expand"] = map[string]any{relation: []any{map[string]any{"id": strings.Repeat("a", 1000)}}}
				case "relative-iri", "malformed-iri", "non-http-iri", "userinfo-iri", "hostless-iri":
					iri := map[string]string{
						"relative-iri":  "/child",
						"malformed-iri": "https://a.example/%",
						"non-http-iri":  "ftp://a.example/child",
						"userinfo-iri":  "https://user@a.example/child",
						"hostless-iri":  "https:///child",
					}[scenario]
					payload["expand"] = map[string]any{relation: []any{map[string]any{"id": "present", "iri": iri}}}
				case "saturated", "capped":
					size := 1000
					if scenario == "capped" {
						size++
					}
					entries := make([]any, size)
					for i := range entries {
						entries[i] = map[string]any{"id": "present", "name": "Present", "text": "Present"}
					}
					payload["expand"] = map[string]any{relation: entries}
				}
				original := newRemoteSyncHTTPClient
				t.Cleanup(func() { newRemoteSyncHTTPClient = original })
				newRemoteSyncHTTPClient = func() *http.Client {
					if scenario == "failed" {
						return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
							return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
						}}}
					}
					return &http.Client{Transport: richTextSyncTransport{payload}}
				}
				_, err := performFullSync(f.app, t.Context(), nil, f.trail)
				if (err != nil) != (scenario == "failed") {
					t.Fatalf("sync result=%v", err)
				}
				if _, err := f.app.FindRecordById(f.children, retained.Id); err != nil {
					t.Fatalf("inconclusive response deleted a cached child: %v", err)
				}
			})
		}
	}
}

func TestRemoteTrailChildIDsAcceptValidIRIs(t *testing.T) {
	for _, target := range []string{"waypoint", "summit_log"} {
		f := newRemoteCleanupFixture(t, target)
		kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
		relation := map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[target]
		for _, test := range []struct{ name, iri string }{
			{"fallback", ""},
			{"origin", "https://a.example/api/v1/" + kind + "/present"},
			{"foreign-origin", "https://a.example:8443/api/v1/" + kind + "/present"},
		} {
			t.Run(target+"/"+test.name, func(t *testing.T) {
				data := map[string]any{"expand": map[string]any{relation: []any{map[string]any{"id": "present", "iri": test.iri}}}}
				listed := remoteTrailChildIDs(f.app, data, "https://a.example", relation, f.children.Name)
				expected := test.iri
				if expected == "" {
					expected = "https://a.example/api/v1/" + kind + "/present"
				}
				if _, present := listed[expected]; len(listed) != 1 || !present {
					t.Fatalf("valid child identity made the relation inconclusive: %v", listed)
				}
			})
		}
	}
}

func TestRemoteChildCleanupWaitsForAllWriteGroups(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "waypoint")
	retained := f.child("https://a.example/api/v1/waypoint/retained", f.remoteActor.Id, f.trail.Id)
	entries := make([]any, remoteSyncWriteBatch+1)
	for i := range entries {
		entries[i] = map[string]any{"id": fmt.Sprint(i), "name": "Present"}
	}
	entries[len(entries)-1].(map[string]any)["name"] = "Unavailable"
	abort := errors.New("child write unavailable")
	f.app.OnRecordCreate("waypoints").BindFunc(func(event *core.RecordEvent) error {
		if event.Record.GetString("name") == "Unavailable" {
			return abort
		}
		return event.Next()
	})
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{"waypoints_via_trail": entries}}}}
	}
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); !errors.Is(err, abort) {
		t.Fatalf("sync should stop at failed child write: %v", err)
	}
	if _, err := f.app.FindRecordById(f.children, retained.Id); err != nil {
		t.Fatalf("failed write group deleted unlisted child: %v", err)
	}
}

func TestRemoteSyncKeepsChildrenWhenPocketBaseOmitsEmptyExpands(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "waypoint")
	retained := f.child("https://a.example/api/v1/waypoint/retained", f.remoteActor.Id, f.trail.Id)
	remote := core.NewRecord(f.trails)
	remote.Id = "trail"
	remote.Set("public", true)
	if failures := f.app.ExpandRecord(remote, []string{"waypoints_via_trail", "summit_logs_via_trail"}, nil); len(failures) != 0 {
		t.Fatalf("expand failed: %v", failures)
	}
	wire, err := json.Marshal(remote)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("empty trail response: %s", wire)
	var payload map[string]any
	if err := json.Unmarshal(wire, &payload); err != nil {
		t.Fatal(err)
	}
	expand, _ := payload["expand"].(map[string]any)
	if _, present := expand["waypoints_via_trail"]; present {
		t.Fatalf("PocketBase emitted an empty backrelation: %s", wire)
	}
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.FindRecordById(f.children, retained.Id); err != nil {
		t.Fatalf("omitted backrelation deleted cached child: %v", err)
	}
}

func TestRemoteSyncKeepsEntriesStoredDuringFetch(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	removed := f.child("https://a.example/api/v1/summit-log/removed", f.remoteActor.Id, f.trail.Id)
	kept := f.child("https://a.example/api/v1/summit-log/kept", f.remoteActor.Id, f.trail.Id)
	var added *core.Record
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if added == nil {
				added = f.child("https://a.example/api/v1/summit-log/added", f.remoteActor.Id, f.trail.Id)
			}
			return richTextSyncTransport{map[string]any{"id": "trail", "public": true,
				"expand": map[string]any{"summit_logs_via_trail": []any{
					map[string]any{"id": "kept", "iri": kept.GetString("iri"), "text": "Kept"},
				}}}}.RoundTrip(request)
		}}}
	}
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.FindRecordById(f.children, removed.Id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing preexisting entry survived: %v", err)
	}
	if added == nil {
		t.Fatal("origin document was not fetched")
	}
	if _, err := f.app.FindRecordById(f.children, added.Id); err != nil {
		t.Fatalf("entry stored while the sync was in flight was deleted: %v", err)
	}
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.FindRecordById(f.children, added.Id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("later complete response did not remove the missing entry: %v", err)
	}
}

func TestRemoteSummitLogCleanupWaitsForAllWriteGroups(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	retained := f.child("https://a.example/api/v1/summit-log/retained", f.remoteActor.Id, f.trail.Id)
	entries := make([]any, remoteSyncWriteBatch+1)
	for i := range entries {
		entries[i] = map[string]any{"id": fmt.Sprint(i), "text": "Present"}
	}
	entries[len(entries)-1].(map[string]any)["text"] = "Unavailable"
	abort := errors.New("child write unavailable")
	f.app.OnRecordCreate("summit_logs").BindFunc(func(event *core.RecordEvent) error {
		if event.Record.GetString("text") == "Unavailable" {
			return abort
		}
		return event.Next()
	})
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{"summit_logs_via_trail": entries}}}}
	}
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); !errors.Is(err, abort) {
		t.Fatalf("sync should stop at failed child write: %v", err)
	}
	if _, err := f.app.FindRecordById(f.children, retained.Id); err != nil {
		t.Fatalf("failed write group deleted unlisted entry: %v", err)
	}
}
