package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestFeedExpansionUsesCurrentItemAndRelationPermissions(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, reader := f.localActor("owner"), f.localActor("reader")
	category := f.save("categories", map[string]any{"name": "Restricted category"})
	public := f.save("trails", map[string]any{"name": "Public trail", "author": owner.Id, "public": true, "category": category.Id})
	private := f.save("trails", map[string]any{"name": "Private trail", "author": owner.Id})
	shared := f.save("trails", map[string]any{"name": "Shared trail", "author": owner.Id})
	list := f.save("lists", map[string]any{"name": "Shared list", "author": owner.Id})
	entry := f.save("summit_logs", map[string]any{"trail": private.Id, "author": reader.Id, "text": "Reader entry"})
	trailShare := f.save("trail_share", map[string]any{"trail": shared.Id, "actor": reader.Id, "permission": "view"})
	listShare := f.save("list_share", map[string]any{"list": list.Id, "actor": reader.Id, "permission": "view"})
	items := map[string]*core.Record{public.Id: public, private.Id: private, shared.Id: shared, list.Id: list, entry.Id: entry}
	for _, item := range items {
		typ := map[string]string{"trails": "trail", "lists": "list", "summit_logs": "summit_log"}[item.Collection().Name]
		f.save("feed", map[string]any{"actor": reader.Id, "author": item.GetString("author"), "type": typ, "item": item.Id})
	}
	for _, change := range []struct{ name, rule string }{{"categories", "name != 'Restricted category'"}, {"activitypub_actors", "user = @request.auth.id"}} {
		collection, err := f.app.FindCollectionByNameOrId(change.name)
		if err != nil {
			t.Fatal(err)
		}
		collection.ViewRule = types.Pointer(change.rule)
		if err := f.app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	f.bindHooks()
	f.app.OnRecordsListRequest("feed").Bind(&hook.Handler[*core.RecordsListRequestEvent]{Priority: -10, Func: func(e *core.RecordsListRequestEvent) error {
		for _, row := range e.Records {
			if row.GetString("item") == private.Id {
				row.MergeExpand(map[string]any{"item": private})
			}
		}
		return e.Next()
	}})
	user, err := f.app.FindRecordById("users", reader.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, revoked := range []bool{false, true} {
		if revoked {
			for _, share := range []*core.Record{trailShare, listShare} {
				if err := f.app.Delete(share); err != nil {
					t.Fatal(err)
				}
			}
		}
		request := httptest.NewRequest(http.MethodGet, "/api/collections/feed/records?perPage=100", nil)
		request.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("feed status=%d: %s", response.Code, response.Body.String())
		}
		var body struct {
			Items []struct {
				Item   string `json:"item"`
				Expand struct {
					Item *struct {
						ID     string         `json:"id"`
						Expand map[string]any `json:"expand"`
					} `json:"item"`
				} `json:"expand"`
			} `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Items) != len(items) {
			t.Fatalf("feed returned %d rows, want %d", len(body.Items), len(items))
		}
		for _, row := range body.Items {
			want := row.Item == public.Id || row.Item == entry.Id || (!revoked && (row.Item == shared.Id || row.Item == list.Id))
			if (row.Expand.Item != nil) != want {
				t.Fatalf("item %s expansion=%v want=%v after share revocation=%v", row.Item, row.Expand.Item, want, revoked)
			}
			if row.Expand.Item == nil {
				continue
			}
			if row.Expand.Item.ID != row.Item {
				t.Fatalf("expanded a different item: %+v", row)
			}
			for _, relation := range []string{"category", "trail"} {
				if row.Expand.Item.Expand[relation] != nil {
					t.Fatalf("unreadable %s relation expanded for item %s", relation, row.Item)
				}
			}
			if row.Item == public.Id && row.Expand.Item.Expand["author"] != nil {
				t.Fatal("unreadable author expanded for the public item")
			}
			if row.Item == entry.Id && row.Expand.Item.Expand["author"] == nil {
				t.Fatal("readable entry author was not expanded")
			}
		}
	}
}
