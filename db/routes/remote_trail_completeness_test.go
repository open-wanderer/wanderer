package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestTrailChildProofKeepsParserErrorsAndExistingKeysInconclusive(t *testing.T) {
	f := newRemoteSyncFixture(t)
	for _, c := range []*core.Collection{f.waypoints, f.logs} {
		c.ListRule = types.Pointer("")
		if err := f.app.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	trail := f.save(f.trails, map[string]any{})
	for _, tc := range []string{"parser error", "unsaved", "existing keys", "cancelled query"} {
		t.Run(tc, func(t *testing.T) {
			r := trail.Fresh()
			e := &core.RequestEvent{App: f.app}
			e.Request = httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"?expand=waypoints_via_trail,summit_logs_via_trail", nil)
			switch tc {
			case "parser error":
				e.Request = httptest.NewRequest(http.MethodGet, e.Request.URL.String(), strings.NewReader("{invalid"))
				e.Request.Header.Set("Content-Type", "application/json")
			case "unsaved":
				r.Id = ""
			case "existing keys":
				r.SetExpand(map[string]any{"waypoints_via_trail": []*core.Record{}, "summit_logs_via_trail": nil})
			case "cancelled query":
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				e.Request = e.Request.WithContext(ctx)
			}
			completeRequestedTrailChildren(e, r)
			expands := r.Expand()
			if tc == "existing keys" {
				if len(expands) != 2 || expands["summit_logs_via_trail"] != nil {
					t.Fatalf("overwrote existing expand keys: %+v", expands)
				}
			} else if len(expands) != 0 {
				t.Fatalf("inconclusive proof became empty: %+v", expands)
			}
		})
	}
}

func TestTrailChildProofDoesNotMutateParentRequestInfo(t *testing.T) {
	f := newRemoteSyncFixture(t)
	for _, c := range []*core.Collection{f.waypoints, f.logs} {
		c.ListRule = types.Pointer("")
		if err := f.app.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	e := &core.RequestEvent{App: f.app}
	e.Request = httptest.NewRequest(http.MethodGet, "/remote/trail/id?expand=waypoints_via_trail&filter=caller&share=&extra=parent", nil)
	e.Set(core.RequestEventKeyInfoContext, "node")
	info, err := e.RequestInfo()
	if err != nil {
		t.Fatal(err)
	}
	trail := f.save(f.trails, map[string]any{})
	completeRequestedTrailChildren(e, trail)
	if len(info.Query) != 4 || info.Query["filter"] != "caller" || info.Query["extra"] != "parent" || info.Query["expand"] != "waypoints_via_trail" || info.Context != "node" {
		t.Fatalf("proof mutated parent request info: %+v", info)
	}
	if _, ok := info.Query["share"]; !ok || len(trail.Expand()) != 1 {
		t.Fatal("share presence lost or proof did not run")
	}
}

func TestRemoteTrailChildProofCachedHandleUsesActualStoredID(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.trails.Fields.Add(&core.DateField{Name: "updated"})
	f.trails.ViewRule = types.Pointer("")
	for _, c := range []*core.Collection{f.trails, f.waypoints, f.logs} {
		c.ListRule = types.Pointer("")
		if err := f.app.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	iri := "https://remote.example/api/v1/trail/origintrail0001"
	trail := f.save(f.trails, map[string]any{"iri": iri, "author": f.remoteActor.Id, "updated": time.Now(), "full_sync_completed": true})
	// A failed expansion with a list-visible child must remain missing. A proof
	// against the origin path ID instead of the stored cache ID would report [].
	f.save(f.waypoints, map[string]any{"trail": trail.Id, "name": "Stored child"})
	previous := remoteHandleGetActor
	t.Cleanup(func() { remoteHandleGetActor = previous })
	remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) { return f.remoteActor, nil }
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/remote/trail/{id}", RemoteTrailGet)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/remote/trail/origintrail0001?handle=@riley@remote.example&expand=waypoints_via_trail,summit_logs_via_trail", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"`+trail.Id+`"`) || strings.Contains(w.Body.String(), `"waypoints_via_trail"`) || !strings.Contains(w.Body.String(), `"summit_logs_via_trail":[]`) {
		t.Fatalf("cached handle proof used the wrong trail identity: status=%d body=%s", w.Code, w.Body.String())
	}
}
