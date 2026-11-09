package federation

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/util"
	"testing"
)

func ensureTestPhotoCollections(t *testing.T, app core.App) {
	t.Helper()
	if _, err := app.FindCollectionByNameOrId("assets"); err == nil {
		return
	}
	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(&core.TextField{Name: "type"}, &core.TextField{Name: "author"}, &core.TextField{Name: "storage_mode"}, &core.TextField{Name: "remote_status"}, &core.TextField{Name: "external_provider"}, &core.TextField{Name: "external_id"}, &core.TextField{Name: "content_hash"}, &core.FileField{Name: "file", MaxSelect: 1, MaxSize: util.RemotePhotoMaxBytes}, &core.JSONField{Name: "metadata"}, &core.NumberField{Name: "lat"}, &core.NumberField{Name: "lon"}, &core.AutodateField{Name: "created", OnCreate: true}, &core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	if err := app.Save(assets); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		links := core.NewBaseCollection(target + "_assets")
		links.Fields.Add(&core.TextField{Name: target}, &core.RelationField{Name: "asset", CollectionId: assets.Id, MaxSelect: 1}, &core.AutodateField{Name: "created", OnCreate: true}, &core.BoolField{Name: "is_thumbnail"})
		if err := app.Save(links); err != nil {
			t.Fatal(err)
		}
	}
}

func testPhotoTarget(record *core.Record) string {
	return map[string]string{"trails": "trail", "waypoints": "waypoint", "summit_logs": "summit_log"}[record.Collection().Name]
}

func testPhotoAssets(t *testing.T, app core.App, record *core.Record) []*core.Record {
	t.Helper()
	assets, err := util.PhotoAssetsForTarget(app, testPhotoTarget(record), record.Id, -1)
	if err != nil {
		t.Fatal(err)
	}
	return assets
}

func testPhotoFiles(t *testing.T, app core.App, record *core.Record) []string {
	t.Helper()
	files := []string{}
	for _, asset := range testPhotoAssets(t, app, record) {
		files = append(files, asset.GetString("file"))
	}
	return files
}

func testAddPhoto(t *testing.T, app core.App, record *core.Record, file *filesystem.File, externalID string) *core.Record {
	t.Helper()
	input := util.PhotoAssetInput{Author: record.GetString("author"), File: file}
	switch testPhotoTarget(record) {
	case "trail":
		input.Trail = record.Id
	case "waypoint":
		input.Waypoint = record.Id
	case "summit_log":
		input.SummitLog = record.Id
	}
	if externalID != "" {
		input.ExternalProvider = util.FederationAssetProvider
		input.ExternalID = externalID
	}
	asset, err := util.CreatePhotoAsset(app, input)
	if err != nil {
		t.Fatal(err)
	}
	return asset
}
