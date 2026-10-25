package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

func localHandleObjectFixture(t *testing.T) (*pbtests.TestApp, *core.Record, *core.Record) {
	t.Helper()
	t.Setenv("ORIGIN", "https://www.local.example")
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	drainTestAppStartupLogs(t, app)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "user"}, &core.BoolField{Name: "is_local"},
		&core.TextField{Name: "preferred_username"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "iri"})
	settings := core.NewBaseCollection("settings")
	settings.Fields.Add(&core.TextField{Name: "user"}, &core.TextField{Name: "bio"})
	follows := core.NewBaseCollection("follows")
	follows.Fields.Add(&core.TextField{Name: "follower"}, &core.TextField{Name: "followee"}, &core.TextField{Name: "status"})
	for _, collection := range []*core.Collection{actors, settings, follows} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"trails", "lists"} {
		collection := core.NewBaseCollection(name)
		collection.ViewRule = types.Pointer("public=true")
		collection.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "name"},
			&core.BoolField{Name: "public"}, &core.TextField{Name: "author"},
			&core.BoolField{Name: "needs_full_sync"}, &core.DateField{Name: "updated"})
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	if err := app.SaveNoValidate(user); err != nil {
		t.Fatal(err)
	}
	preferences := core.NewRecord(settings)
	preferences.Set("user", user.Id)
	if err := app.Save(preferences); err != nil {
		t.Fatal(err)
	}
	local := core.NewRecord(actors)
	local.Set("user", user.Id)
	local.Set("is_local", true)
	local.Set("preferred_username", "alice")
	local.Set("domain", "local.example")
	local.Set("iri", "https://www.local.example/api/v1/activitypub/user/alice")
	if err := app.Save(local); err != nil {
		t.Fatal(err)
	}
	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	list := core.NewRecord(lists)
	list.Set("author", local.Id)
	list.Set("public", true)
	list.Set("name", "Local list")
	list.Set("updated", time.Now())
	if err := app.Save(list); err != nil {
		t.Fatal(err)
	}
	list.Set("iri", "https://www.local.example/api/v1/list/"+list.Id)
	list.Set("needs_full_sync", true)
	if err := app.Save(list); err != nil {
		t.Fatal(err)
	}
	return app, local, list
}

func cancelledLocalObjectEvent(app core.App, handle, id string) *core.RequestEvent {
	request := httptest.NewRequest(http.MethodGet, "/remote/list/"+id+"?handle="+url.QueryEscape(handle), nil)
	request.SetPathValue("id", id)
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	event := &core.RequestEvent{App: app}
	event.Request, event.Response = request.WithContext(ctx), httptest.NewRecorder()
	return event
}

func TestLocalHandleObjectResolversKeepTheStoredAuthor(t *testing.T) {
	app, local, _ := localHandleObjectFixture(t)
	for _, handle := range []string{"@ALICE", "@alice@www.local.example", "@ALICE@WWW.LOCAL.EXAMPLE", "@Alice@LOCAL.EXAMPLE"} {
		for name, resolve := range map[string]func(*core.RequestEvent, context.Context, string, string) (*core.Record, error){
			"trail": findLocalTrailByRemoteInfo, "list": findLocalListByRemoteInfo,
		} {
			t.Run(name+"/"+handle, func(t *testing.T) {
				event := cancelledLocalObjectEvent(app, handle, "missing00000001")
				// No existing object can mask a failed actor lookup through the
				// ID fallback; the shell must use the resolved local author.
				object, err := resolve(event, event.Request.Context(), handle, "missing00000001")
				if err != nil || object == nil {
					t.Fatalf("local %s resolution failed: object=%v error=%v", name, object, err)
				}
				if object.Id != "" || object.GetString("author") != local.Id || object.GetString("iri") != "https://www.local.example/api/v1/"+name+"/missing00000001" {
					t.Fatalf("local %s shell did not keep its stored author and origin: %v", name, object)
				}
			})
		}
	}
}

func TestLocalListHandleAliasesServeAndClearTheSyncFlag(t *testing.T) {
	app, local, list := localHandleObjectFixture(t)
	for _, handle := range []string{"@ALICE", "@alice@www.local.example", "@ALICE@WWW.LOCAL.EXAMPLE", "@Alice@LOCAL.EXAMPLE"} {
		t.Run(handle, func(t *testing.T) {
			list.Set("needs_full_sync", true)
			if err := app.Save(list); err != nil {
				t.Fatal(err)
			}
			event := cancelledLocalObjectEvent(app, handle, list.Id)
			if err := RemoteListGet(event); err != nil {
				t.Fatal(err)
			}
			response := event.Response.(*httptest.ResponseRecorder)
			var payload struct {
				ID            string `json:"id"`
				Author        string `json:"author"`
				NeedsFullSync bool   `json:"needs_full_sync"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || payload.ID != list.Id || payload.Author != local.Id || payload.NeedsFullSync {
				t.Fatalf("local list response status=%d body=%s", response.Code, response.Body.String())
			}
			stored, err := app.FindRecordById("lists", list.Id)
			if err != nil || stored.GetBool("needs_full_sync") {
				t.Fatalf("local list sync flag remains set: record=%v error=%v", stored, err)
			}

			missing := cancelledLocalObjectEvent(app, handle, "missing00000001")
			err = RemoteListGet(missing)
			apiError, ok := err.(*router.ApiError)
			if !ok || apiError.Status != http.StatusNotFound {
				t.Fatalf("missing local list error=%v, want 404", err)
			}
		})
	}
}
