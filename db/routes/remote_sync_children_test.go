package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

func childRefreshFixture(t *testing.T) *remoteSyncFixture {
	f := newRemoteSyncFixture(t)
	f.trails.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	f.lists.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	f.trails.ViewRule = types.Pointer("public=true")
	f.lists.ViewRule = types.Pointer("public=true")
	f.waypoints.Fields.GetByName("name").(*core.TextField).Max = 5000
	f.waypoints.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 99, MaxSize: 20 << 20, MimeTypes: []string{"image/jpeg", "image/png"}})
	f.logs.Fields.Add(&core.NumberField{Name: "distance", Min: types.Pointer(float64(0))}, &core.FileField{Name: "photos", MaxSelect: 99, MaxSize: 20 << 20, MimeTypes: []string{"image/jpeg", "image/png"}})
	for _, col := range []*core.Collection{f.trails, f.lists, f.waypoints, f.logs} {
		if err := f.app.Save(col); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func refreshTransport(t *testing.T, payload func(*http.Request) map[string]any) *atomic.Int32 {
	t.Helper()
	calls := &atomic.Int32{}
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return richTextSyncTransport{payload(req)}.RoundTrip(req)
		}}}
	}
	return calls
}

func checkCompletedChildView(t *testing.T, f *remoteSyncFixture, kind string, record *core.Record) {
	t.Helper()
	old := remoteHandleGetActor
	defer func() { remoteHandleGetActor = old }()
	remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) { return f.remoteActor, nil }
	e := &core.RequestEvent{App: f.app}
	e.Request = httptest.NewRequest(http.MethodGet, "/remote/"+kind+"/copy?handle=@riley@remote.example", nil)
	e.Request.SetPathValue("id", "copy")
	response := httptest.NewRecorder()
	e.Response = response
	handler := RemoteTrailGet
	if kind == "list" {
		handler = RemoteListGet
	}
	if err := handler(e); err != nil || response.Code != 200 {
		t.Fatalf("completed view failed: status%d error%v", response.Code, err)
	}
	if err := waitForBackgroundSync(record, 3*time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteSyncSkipsInvalidChildrenAndRetriesLater(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, variant := range []string{"name-limit", "photo-content", "third-party-distance"} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing%t", variant, existing), func(t *testing.T) {
				f := childRefreshFixture(t)
				parent := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/copy", "name": "Cached", "public": true, "author": f.remoteActor.Id, "full_sync_completed": true, "needs_full_sync": true})
				kind, col, author, origin := "waypoint", f.waypoints, f.remoteActor, "https://remote.example"
				if variant == "third-party-distance" {
					kind, col, origin = "summit-log", f.logs, "https://contributor.example"
					author = f.save(f.actors, map[string]any{"iri": origin + "/actor/riley"})
				}
				iri := origin + "/api/v1/" + kind + "/child"
				var previous *core.Record
				if existing {
					previous = f.save(col, map[string]any{"iri": iri, "name": "Saved child", "text": "Saved child", "author": author.Id, "trail": parent.Id})
				}
				repaired := false
				calls := refreshTransport(t, func(req *http.Request) map[string]any {
					bad := map[string]any{"id": "child", "iri": iri, "name": "Updated child", "text": "Updated child", "distance": 1}
					if !repaired {
						switch variant {
						case "name-limit":
							bad["name"] = strings.Repeat("a", 5001)
						case "third-party-distance":
							bad["distance"] = -1
						default:
							bad["photos"] = []any{"photo.jpg"}
						}
					}
					if req.URL.Host == "contributor.example" {
						bad["expand"] = map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": parent.GetString("iri")}}
						return bad
					}
					relation := "waypoints_via_trail"
					if kind == "summit-log" {
						relation = "summit_logs_via_trail"
						bad = map[string]any{"id": "child", "iri": iri}
					}
					valid := map[string]any{"id": "valid", "name": "Valid sibling", "text": "Valid sibling"}
					return map[string]any{"id": "copy", "iri": parent.GetString("iri"), "name": "Refreshed", "public": true, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, relation: []any{bad, valid}}}
				})
				oldActor := remoteSyncGetActorByIRI
				t.Cleanup(func() { remoteSyncGetActorByIRI = oldActor })
				remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, iri string, _ bool) (*core.Record, error) {
					return app.FindFirstRecordByData("activitypub_actors", "iri", iri)
				}
				oldMedia := fetchRemoteSyncMedia
				t.Cleanup(func() { fetchRemoteSyncMedia = oldMedia })
				fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
					file, err := filesystem.NewFileFromBytes([]byte("<!doctype html><html>Photo unavailable</html>"), "photo.jpg")
					return file, func() {}, err
				}
				got, err := performFullSync(f.app, t.Context(), nil, parent)
				if err != nil || got.GetBool("needs_full_sync") || !got.GetBool("full_sync_completed") || got.GetString("name") != "Refreshed" {
					t.Fatalf("invalid child prevented completion: record%v error%v", got, err)
				}
				if n, err := f.app.CountRecords(col.Name); err != nil || n != map[bool]int64{false: 1, true: 2}[existing] {
					t.Fatalf("valid sibling lost: count%d error%v", n, err)
				}
				if existing {
					fields := []string{"author", "trail", "iri", "photos"}
					if col == f.waypoints {
						fields = append(fields, "name")
					} else {
						fields = append(fields, "text")
					}
					f.assertUnchanged(previous, fields...)
				}
				before := calls.Load()
				checkCompletedChildView(t, f, "trail", got)
				if calls.Load() != before {
					t.Fatal("fresh view retried the invalid child immediately")
				}
				repaired = true
				got, err = f.app.FindRecordById("trails", got.Id)
				if err != nil {
					t.Fatal(err)
				}
				got, err = performFullSync(f.app, t.Context(), nil, got)
				if err != nil {
					t.Fatal(err)
				}
				child, err := f.app.FindFirstRecordByData(col.Name, "iri", iri)
				if err != nil {
					t.Fatal(err)
				}
				if col == f.waypoints && child.GetString("name") != "Updated child" || col == f.logs && child.GetFloat("distance") != 1 {
					t.Fatal("later corrected child was not imported")
				}
			})
		}
	}
}

func TestRemoteListSyncKeepsInvalidExistingChildAndValidSiblings(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := childRefreshFixture(t)
	f.trails.Fields.GetByName("name").(*core.TextField).Max = 5000
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	previous := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/child", "name": "Saved child", "author": f.remoteActor.Id})
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/copy", "author": f.remoteActor.Id, "trails": []string{previous.Id}, "public": true, "full_sync_completed": true, "needs_full_sync": true})
	refreshTransport(t, func(*http.Request) map[string]any {
		return map[string]any{"id": "copy", "iri": list.GetString("iri"), "name": "Refreshed", "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, "trails": []any{map[string]any{"id": "child", "name": strings.Repeat("a", 5001)}, map[string]any{"id": "invalidnew", "name": strings.Repeat("b", 5001)}, map[string]any{"id": "valid", "name": "Valid sibling"}}}}
	})
	got, err := performFullListSync(f.app, t.Context(), nil, list)
	if err != nil {
		t.Fatal(err)
	}
	if ids := got.GetStringSlice("trails"); len(ids) != 2 || !slices.Contains(ids, previous.Id) {
		t.Fatalf("invalid child replaced good list relations: %v", ids)
	}
	f.assertUnchanged(previous, "name", "iri", "author")
}

func TestRemoteSyncBudgetAfterCommittedMetadataKeepsCopiesReadable(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, mode := range []string{"trail/metadata", "trail/children", "list/children"} {
		t.Run(mode, func(t *testing.T) {
			f := childRefreshFixture(t)
			kind := "trail"
			col := f.trails
			children := f.waypoints
			relation := "waypoints_via_trail"
			if strings.HasPrefix(mode, "list/") {
				kind, col, children, relation = "list", f.lists, f.trails, "trails"
				f.lists.Fields.GetByName("trails").(*core.RelationField).MaxSelect = 200
				if err := f.app.Save(f.lists); err != nil {
					t.Fatal(err)
				}
			}
			parent := f.save(col, map[string]any{"iri": "https://remote.example/api/v1/" + kind + "/copy", "author": f.remoteActor.Id, "name": "Cached", "public": true, "full_sync_completed": true, "needs_full_sync": true})
			childKind := "waypoint"
			if kind == "list" {
				childKind = "trail"
			}
			later := f.save(children, map[string]any{"iri": "https://remote.example/api/v1/" + childKind + "/child100", "author": f.remoteActor.Id, "trail": parent.Id, "name": "Saved later child"})
			if kind == "list" {
				parent.Set("trails", []string{later.Id})
				if err := f.app.Save(parent); err != nil {
					t.Fatal(err)
				}
			}
			values := make([]any, remoteSyncWriteBatch+1)
			for i := range values {
				values[i] = map[string]any{"id": fmt.Sprintf("child%d", i), "name": "Refreshed child"}
			}
			refreshTransport(t, func(*http.Request) map[string]any {
				return map[string]any{"id": "copy", "iri": parent.GetString("iri"), "name": "Refreshed", "public": true, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, relation: values}}
			})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if mode == "trail/metadata" {
				f.app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
					if e.Record.Id == parent.Id && e.Record.GetBool("needs_full_sync") {
						cancel()
					}
					return e.Next()
				})
			} else {
				f.app.OnRecordAfterCreateSuccess(children.Name).BindFunc(func(e *core.RecordEvent) error { cancel(); return e.Next() })
			}
			var got *core.Record
			var err error
			if kind == "trail" {
				got, err = performFullSync(f.app, ctx, nil, parent)
			} else {
				got, err = performFullListSync(f.app, ctx, nil, parent)
			}
			if err != nil || ctx.Err() == nil || got.GetBool("needs_full_sync") || !got.GetBool("full_sync_completed") {
				t.Fatalf("interrupted children left blocking markers: record%v error%v ctx%v", got, err, ctx.Err())
			}
			f.assertUnchanged(later, "name", "author", "iri")
			if kind == "list" {
				if ids := got.GetStringSlice("trails"); len(ids) != 101 || !slices.Contains(ids, later.Id) {
					t.Fatalf("unprocessed existing list child disappeared: %v", ids)
				}
			}
			checkCompletedChildView(t, f, kind, got)
		})
	}
}

func TestRemoteSyncUnexpectedChildWriteErrorStillPropagates(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := childRefreshFixture(t)
	parent := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/copy", "author": f.remoteActor.Id, "name": "Cached", "public": true, "full_sync_completed": true, "needs_full_sync": true})
	writeErr := fmt.Errorf("child storage unavailable: %w", os.ErrNotExist)
	f.app.OnRecordCreateExecute("waypoints").BindFunc(func(*core.RecordEvent) error { return writeErr })
	refreshTransport(t, func(*http.Request) map[string]any {
		return map[string]any{"id": "copy", "iri": parent.GetString("iri"), "name": "Refreshed", "public": true, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, "waypoints_via_trail": []any{map[string]any{"id": "child", "name": "Child"}}}}
	})
	if _, err := performFullSync(f.app, t.Context(), nil, parent); !errors.Is(err, writeErr) {
		t.Fatalf("storage error hidden: %v", err)
	}
	checkCompletedChildView(t, f, "trail", parent)
}
