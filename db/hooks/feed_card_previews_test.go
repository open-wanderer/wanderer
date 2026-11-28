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

func TestFeedCardsKeepThreePhotosIncludingTheMarkedTitleWithoutLargeMetadata(t *testing.T) {
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
	actors := add("activitypub_actors", &core.TextField{Name: "name"})
	author := save(actors, map[string]any{"name": "Author"})
	categories := add("categories", &core.TextField{Name: "name"})
	trails := add("trails", &core.TextField{Name: "name"}, &core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1}, &core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1})
	trail := save(trails, map[string]any{"name": "Big collection", "author": author.Id})
	assets := add("assets", &core.TextField{Name: "type"}, &core.TextField{Name: "author"}, &core.TextField{Name: "storage_mode"}, &core.TextField{Name: "file"}, &core.JSONField{Name: "metadata"})
	links := add("trail_assets", &core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1}, &core.RelationField{Name: "asset", CollectionId: assets.Id, MaxSelect: 1}, &core.BoolField{Name: "is_thumbnail"}, &core.AutodateField{Name: "created", OnCreate: true}, &core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	feed := add("feed", &core.TextField{Name: "item"}, &core.TextField{Name: "type"}, &core.TextField{Name: "actor"})
	save(feed, map[string]any{"item": trail.Id, "type": "trail", "actor": author.Id})
	titleID := ""
	if err := app.RunInTransaction(func(tx core.App) error {
		for i := 0; i < 1500; i++ {
			asset := core.NewRecord(assets)
			asset.Load(map[string]any{"type": "photo", "author": author.Id, "storage_mode": "link_public", "metadata": map[string]any{"unrelated": strings.Repeat("private-large-field", 300)}})
			if err := tx.Save(asset); err != nil {
				return err
			}
			link := core.NewRecord(links)
			link.Load(map[string]any{"trail": trail.Id, "asset": asset.Id, "is_thumbnail": i == 1499})
			if err := tx.Save(link); err != nil {
				return err
			}
			if i == 1499 {
				titleID = asset.Id
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	app.OnRecordsListRequest("feed").BindFunc(ListFeedHandler())
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/collections/feed/records", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("feedstatus=%d %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []struct {
			Expand struct {
				Item struct {
					Expand map[string]json.RawMessage `json:"expand"`
				} `json:"item"`
			} `json:"expand"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("feeditemcount=%d", len(body.Items))
	}
	var photos []struct {
		Asset       string `json:"asset"`
		IsThumbnail bool   `json:"is_thumbnail"`
	}
	if err := json.Unmarshal(body.Items[0].Expand.Item.Expand["trail_assets_via_trail"], &photos); err != nil {
		t.Fatal(err)
	}
	if len(photos) != 3 || photos[0].Asset != titleID || !photos[0].IsThumbnail {
		t.Fatalf("marked late title lost among %d photos: %v", len(photos), photos)
	}
	if response.Body.Len() > 4500 || strings.Contains(response.Body.String(), "unrelated") {
		t.Fatalf("feed retained full photos/metadata: %d bytes", response.Body.Len())
	}
}
