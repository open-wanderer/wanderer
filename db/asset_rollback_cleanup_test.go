package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/routes"
)

func TestAssetRollbackCleanupKeepsLinkedPhotosAndChecksOrphansInTransaction(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, other := f.localActor("rollback-owner"), f.localActor("rollback-other")
	linked, orphan, foreign := f.photo(owner), f.photo(owner), f.photo(other)
	trail := f.save("trails", map[string]any{"author": owner.Id, "name": "Another saved target"})
	link := f.save("trail_assets", map[string]any{"trail": trail.Id, "asset": linked.Id})
	f.app.OnRecordDelete("assets").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id == orphan.Id && e.App.TxInfo() == nil {
			return fmt.Errorf("orphan cleanup must retain the link-check transaction through deletion")
		}
		return e.Next()
	})
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	api.DELETE("/assets/{id}", routes.AssetDelete)
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	user, err := f.app.FindRecordById("users", owner.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	request := func(asset *core.Record, query string) (int, bool) {
		req := httptest.NewRequest(http.MethodDelete, "/assets/"+asset.Id+query, nil)
		req.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		var body struct {
			AssetDeleted bool `json:"assetDeleted"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return response.Code, body.AssetDeleted
	}
	if status, deleted := request(linked, "?orphan=1"); status != 200 || deleted || !f.exists(linked) || !f.exists(link) {
		t.Fatalf("rollback cleanup removed another target's saved photo: status=%d deleted=%t", status, deleted)
	}
	if status, deleted := request(orphan, "?orphan=1"); status != 200 || !deleted || f.exists(orphan) {
		t.Fatalf("unlinked owned photo was not cleaned in its transaction: status=%d deleted=%t", status, deleted)
	}
	if status, _ := request(foreign, "?orphan=1"); status != 403 || !f.exists(foreign) {
		t.Fatalf("rollback cleanup changed another author's photo: status=%d", status)
	}
	if status, _ := request(linked, ""); status != 200 || f.exists(linked) || f.exists(link) {
		t.Fatalf("explicit owner deletion no longer removes the photo and links: status=%d", status)
	}
}
