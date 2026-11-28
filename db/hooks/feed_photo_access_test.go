package hooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestFeedPhotoPreviewsUseCurrentRelationPermissions(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	app.Settings().Logs.MaxDays = 0
	add := func(name string, fields ...core.Field) *core.Collection {
		collection := core.NewBaseCollection(name)
		collection.ViewRule, collection.ListRule = types.Pointer(""), types.Pointer("")
		collection.Fields.Add(fields...)
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
		return collection
	}
	save := func(collection *core.Collection, values map[string]any) *core.Record {
		record := core.NewRecord(collection)
		record.Load(values)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record
	}
	users, _ := app.FindCollectionByNameOrId("users")
	user := save(users, map[string]any{"email": "feed-owner@a.example", "password": "Passw0rd!Passw0rd"})
	actors := add("activitypub_actors", &core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1})
	owner := save(actors, map[string]any{"user": user.Id})
	categories := add("categories", &core.TextField{Name: "name"})
	trails := add("trails", &core.TextField{Name: "name"}, &core.BoolField{Name: "public"}, &core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1}, &core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1})
	trails.ViewRule = types.Pointer("public=true || author.user=@request.auth.id")
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	assets := add("assets", &core.TextField{Name: "type"}, &core.TextField{Name: "file"}, &core.TextField{Name: "taken_at"}, &core.NumberField{Name: "lat"}, &core.BoolField{Name: "readable"})
	assets.ViewRule = types.Pointer("readable=true")
	if err := app.Save(assets); err != nil {
		t.Fatal(err)
	}
	links := add("trail_assets", &core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1}, &core.RelationField{Name: "asset", CollectionId: assets.Id, MaxSelect: 1})
	links.ViewRule = trails.ViewRule
	links.ViewRule = types.Pointer("trail.public=true || trail.author.user=@request.auth.id")
	if err := app.Save(links); err != nil {
		t.Fatal(err)
	}
	feed := add("feed", &core.TextField{Name: "type"}, &core.TextField{Name: "item"}, &core.TextField{Name: "actor"})
	public := save(trails, map[string]any{"name": "public", "public": true, "author": owner.Id})
	private := save(trails, map[string]any{"name": "private", "public": false, "author": owner.Id})
	allowed := save(assets, map[string]any{"type": "photo", "file": "public-photo.png", "taken_at": "public-time", "lat": 12, "readable": true})
	denied := save(assets, map[string]any{"type": "photo", "file": "nested-secret.png", "taken_at": "secret-time", "lat": 87, "readable": false})
	privateAsset := save(assets, map[string]any{"type": "photo", "file": "private-trail-photo.png", "lat": 54, "readable": true})
	save(links, map[string]any{"trail": public.Id, "asset": allowed.Id})
	save(links, map[string]any{"trail": public.Id, "asset": denied.Id})
	save(links, map[string]any{"trail": private.Id, "asset": privateAsset.Id})
	save(feed, map[string]any{"type": "trail", "item": public.Id, "actor": owner.Id})
	save(feed, map[string]any{"type": "trail", "item": private.Id, "actor": owner.Id})
	app.OnRecordsListRequest("feed").BindFunc(ListFeedHandler())
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	for _, authenticated := range []bool{false, true} {
		request := httptest.NewRequest(http.MethodGet, "/api/collections/feed/records", nil)
		if authenticated {
			request.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("feedstatus=%d %s", response.Code, response.Body.String())
		}
		var body struct {
			Items []struct {
				Item   string         `json:"item"`
				Expand map[string]any `json:"expand"`
			} `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		for _, entry := range body.Items {
			_, expanded := entry.Expand["item"]
			if entry.Item == private.Id && expanded != authenticated {
				t.Fatalf("private feeditem expansion=%v authenticated=%v", expanded, authenticated)
			}
			if entry.Item == public.Id && !expanded {
				t.Fatal("public feeditem lost")
			}
		}
		wire := response.Body.String()
		if strings.Contains(wire, "nested-secret") || strings.Contains(wire, "secret-time") || !strings.Contains(wire, "public-photo") {
			t.Fatalf("nested asset access was not applied: %s", wire)
		}
		if strings.Contains(wire, "private-trail-photo") != authenticated {
			t.Fatal("private trail photo exposure does not match current access")
		}
	}
}
