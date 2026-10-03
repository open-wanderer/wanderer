package routes

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// List pull-sync may only write trails on the list origin's host that are
// authored there.

const listID = "list1"

func (f *pullFixture) addListsCollection(t *testing.T) {
	t.Helper()
	actors, err := f.app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	trails, err := f.app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 50},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "needs_full_sync"},
		&core.BoolField{Name: "full_sync_completed"},
		&core.DateField{Name: "updated"},
	)
	lists.ViewRule = types.Pointer("")
	lists.ListRule = lists.ViewRule
	if err := f.app.Save(lists); err != nil {
		t.Fatal(err)
	}
}

// pullList drives RemoteListGet for alice's list with the origin answering
// the list document with the given trails, and returns the synced list.
func (f *pullFixture) pullList(t *testing.T, items []map[string]any) *core.Record {
	t.Helper()
	f.addListsCollection(t)

	trails := make([]any, len(items))
	for i, item := range items {
		trails[i] = item
	}
	f.origin.paths["/api/v1/list/"+listID] = f.mustJSON(map[string]any{
		"id":     listID,
		"name":   "remote list",
		"expand": map[string]any{"trails": trails},
	})

	request := httptest.NewRequest(http.MethodGet, "/remote/list/"+listID+"?handle=alice@remote.example", nil)
	request.SetPathValue("id", listID)
	response := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.App, e.Request, e.Response = f.app, request, response
	if err := RemoteListGet(e); err != nil {
		t.Fatalf("RemoteListGet: %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", response.Code, response.Body.String())
	}

	list, err := f.app.FindFirstRecordByData("lists", "iri", pullOrigin+"/api/v1/list/"+listID)
	if err != nil {
		t.Fatalf("list was not stored: %v", err)
	}
	return list
}

func (f *pullFixture) trailAt(t *testing.T, host string, author *core.Record, public bool, name string) *core.Record {
	t.Helper()
	trails, err := f.app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(trails)
	r.Set("author", author.Id)
	r.Set("public", public)
	r.Set("name", name)
	r.Set("full_sync_completed", true)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	r.Set("iri", host+"/api/v1/trail/"+r.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *pullFixture) assertTrailUntouched(t *testing.T, label string, want *core.Record) {
	t.Helper()
	got, err := f.app.FindRecordById("trails", want.Id)
	if err != nil {
		t.Errorf("%s: row vanished: %v", label, err)
		return
	}
	if got.GetString("name") != want.GetString("name") {
		t.Errorf("%s: name changed from %q to %q", label, want.GetString("name"), got.GetString("name"))
	}
	if got.GetBool("public") != want.GetBool("public") {
		t.Errorf("%s: public changed from %v to %v", label, want.GetBool("public"), got.GetBool("public"))
	}
	if got.GetString("author") != want.GetString("author") {
		t.Errorf("%s: author changed from %s to %s", label, want.GetString("author"), got.GetString("author"))
	}
	if got.GetString("iri") != want.GetString("iri") {
		t.Errorf("%s: iri changed from %s to %s", label, want.GetString("iri"), got.GetString("iri"))
	}
}

func TestListSyncRefusesLocalAndForeignTrails(t *testing.T) {
	f := setupPullFixture(t)

	local := f.trailAt(t, "https://local.example", f.lou, false, "local original")
	third := f.trailAt(t, pullThird, f.carol, false, "third original")
	other := f.trailAt(t, pullOrigin, f.alice, false, "other original")

	list := f.pullList(t, []map[string]any{
		{"id": local.Id, "iri": local.GetString("iri"), "name": "local pwned", "public": true, "expand": authorExpand(f.mallory)},
		{"id": third.Id, "iri": third.GetString("iri"), "name": "third pwned", "public": true, "expand": authorExpand(f.mallory)},
		{"id": other.Id, "iri": other.GetString("iri"), "name": "other pwned", "public": true, "expand": authorExpand(f.mallory)},
	})

	f.assertTrailUntouched(t, "local trail", local)
	f.assertTrailUntouched(t, "third-host trail", third)
	f.assertTrailUntouched(t, "trail of another author", other)

	linked := list.GetStringSlice("trails")
	for label, r := range map[string]*core.Record{"local trail": local, "third-host trail": third} {
		if !slices.Contains(linked, r.Id) {
			t.Errorf("%s was not linked into the synced list (trails = %v)", label, linked)
		}
	}
}

func TestListSyncStoresSameHostTrails(t *testing.T) {
	f := setupPullFixture(t)

	existing := f.trailAt(t, pullOrigin, f.alice, false, "old")

	list := f.pullList(t, []map[string]any{
		{"id": "n1", "name": "new", "public": true},
		{"id": existing.Id, "iri": existing.GetString("iri"), "name": "updated", "public": true, "expand": authorExpand(f.alice)},
	})

	created, err := f.app.FindFirstRecordByData("trails", "iri", pullOrigin+"/api/v1/trail/n1")
	if err != nil {
		t.Fatalf("new origin trail was not stored: %v", err)
	}
	if created.GetString("name") != "new" || created.GetString("author") != f.alice.Id {
		t.Errorf("new trail = name %q author %s", created.GetString("name"), created.GetString("author"))
	}

	updated, err := f.app.FindRecordById("trails", existing.Id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GetString("name") != "updated" || !updated.GetBool("public") {
		t.Errorf("existing origin trail = name %q public %v", updated.GetString("name"), updated.GetBool("public"))
	}

	linked := list.GetStringSlice("trails")
	if !slices.Contains(linked, created.Id) || !slices.Contains(linked, existing.Id) {
		t.Errorf("synced list trails = %v, want %s and %s", linked, created.Id, existing.Id)
	}
}

func TestListSyncSkipsUnknownForeignTrail(t *testing.T) {
	f := setupPullFixture(t)

	iri := pullThird + "/api/v1/trail/unknown"
	list := f.pullList(t, []map[string]any{
		{"id": "unknown", "iri": iri, "name": "foreign", "public": true, "expand": authorExpand(f.carol)},
		{"id": "n1", "name": "new", "public": true},
	})

	if got, err := f.app.FindFirstRecordByData("trails", "iri", iri); err == nil {
		t.Errorf("third-host trail was created (author %s)", got.GetString("author"))
	}

	created, err := f.app.FindFirstRecordByData("trails", "iri", pullOrigin+"/api/v1/trail/n1")
	if err != nil {
		t.Fatalf("same-host trail after the refused entry was not stored: %v", err)
	}
	linked := list.GetStringSlice("trails")
	if len(linked) != 1 || linked[0] != created.Id {
		t.Errorf("synced list trails = %v, want only %s", linked, created.Id)
	}
}
