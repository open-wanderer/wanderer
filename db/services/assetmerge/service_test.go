package assetmerge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestSuggestGroupsUsesExternalReferenceAndMostLinkedTarget(t *testing.T) {
	app := setupAssetMergeTestApp(t)
	defer app.Cleanup()

	source := createTestAsset(t, app, "user1", "immich", "photo1")
	target := createTestAsset(t, app, "user1", "immich", "photo1")
	createTestAssetLink(t, app, "trail_assets", "trail", "trail1", source.Id)
	createTestAssetLink(t, app, "trail_assets", "trail", "trail1", target.Id)
	createTestAssetLink(t, app, "waypoint_assets", "waypoint", "waypoint1", target.Id)

	response, err := SuggestGroups(app, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(response.Groups))
	}
	group := response.Groups[0]
	if group.MatchReason != reasonExternalReference {
		t.Fatalf("match reason = %q, want %q", group.MatchReason, reasonExternalReference)
	}
	if group.TargetAssetID != target.Id {
		t.Fatalf("target asset = %q, want %q", group.TargetAssetID, target.Id)
	}
	if group.Reason != targetReasonMostLinks {
		t.Fatalf("target reason = %q, want %q", group.Reason, targetReasonMostLinks)
	}
}

func TestMergeReassignsLinksAndDeletesSourceAsset(t *testing.T) {
	app := setupAssetMergeTestApp(t)
	defer app.Cleanup()

	source := createTestAsset(t, app, "user1", "", "")
	target := createTestAsset(t, app, "user1", "", "")
	createTestAssetLink(t, app, "trail_assets", "trail", "trail1", source.Id)
	createTestAssetLink(t, app, "waypoint_assets", "waypoint", "waypoint1", source.Id)
	createTestAssetLink(t, app, "trail_assets", "trail", "trail1", target.Id)

	response, err := Merge(app, "user1", target.Id, []string{source.Id})
	if err != nil {
		t.Fatal(err)
	}
	if !response.Acknowledged {
		t.Fatal("merge was not acknowledged")
	}
	if response.ReassignedLinks != 2 {
		t.Fatalf("reassigned links = %d, want 2", response.ReassignedLinks)
	}
	if _, err := app.FindRecordById("assets", source.Id); err == nil {
		t.Fatal("source asset still exists")
	}
	assertAssetLinkExists(t, app, "trail_assets", "trail", "trail1", target.Id)
	assertAssetLinkExists(t, app, "waypoint_assets", "waypoint", "waypoint1", target.Id)
	assertNoAssetLinks(t, app, source.Id)
}

func TestMergeCopiesMissingSourceMetadataToTarget(t *testing.T) {
	app := setupAssetMergeTestApp(t)
	defer app.Cleanup()

	source := createTestAsset(t, app, "user1", "immich", "photo1")
	target := createTestAsset(t, app, "user1", "", "")
	takenAt := time.Date(2024, 7, 4, 12, 34, 56, 0, time.UTC)
	source.Set("taken_at", takenAt)
	source.Set("lat", 46.12345)
	source.Set("lon", 7.12345)
	source.Set("metadata", map[string]any{
		"source_file": "source.jpg",
		"remote": map[string]any{
			"filename": "remote.jpg",
		},
	})
	if err := app.Save(source); err != nil {
		t.Fatal(err)
	}

	if _, err := Merge(app, "user1", target.Id, []string{source.Id}); err != nil {
		t.Fatal(err)
	}

	updated, err := app.FindRecordById("assets", target.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.GetDateTime("taken_at").Time().Equal(takenAt) {
		t.Fatalf("taken_at = %s, want %s", updated.GetDateTime("taken_at").Time(), takenAt)
	}
	if updated.GetFloat("lat") != 46.12345 || updated.GetFloat("lon") != 7.12345 {
		t.Fatalf("coordinates = %f,%f, want 46.12345,7.12345", updated.GetFloat("lat"), updated.GetFloat("lon"))
	}
	if updated.GetString("external_provider") != "immich" || updated.GetString("external_id") != "photo1" {
		t.Fatalf("external reference = %q/%q, want immich/photo1", updated.GetString("external_provider"), updated.GetString("external_id"))
	}
	metadata, err := metadataMap(updated)
	if err != nil {
		t.Fatal(err)
	}
	if util.AssetMetadataString(metadata, "source_file") != "source.jpg" {
		t.Fatalf("source_file metadata = %q, want source.jpg", util.AssetMetadataString(metadata, "source_file"))
	}
	assertMergedAssetSnapshotContains(t, updated, source.Id, "source.jpg")
}

func TestMergePreservesConflictingSourceMetadataSnapshot(t *testing.T) {
	app := setupAssetMergeTestApp(t)
	defer app.Cleanup()

	source := createTestAsset(t, app, "user1", "immich", "source-photo")
	target := createTestAsset(t, app, "user1", "wanderer", "target-photo")
	sourceTakenAt := time.Date(2024, 7, 4, 12, 34, 56, 0, time.UTC)
	targetTakenAt := time.Date(2023, 6, 3, 11, 22, 33, 0, time.UTC)
	source.Set("taken_at", sourceTakenAt)
	source.Set("lat", 46.12345)
	source.Set("lon", 7.12345)
	source.Set("metadata", map[string]any{"source_file": "source.jpg"})
	target.Set("taken_at", targetTakenAt)
	target.Set("lat", 45.5)
	target.Set("lon", 8.5)
	target.Set("metadata", map[string]any{"source_file": "target.jpg"})
	if err := app.Save(source); err != nil {
		t.Fatal(err)
	}
	if err := app.Save(target); err != nil {
		t.Fatal(err)
	}

	if _, err := Merge(app, "user1", target.Id, []string{source.Id}); err != nil {
		t.Fatal(err)
	}

	updated, err := app.FindRecordById("assets", target.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.GetDateTime("taken_at").Time().Equal(targetTakenAt) {
		t.Fatalf("target taken_at was overwritten: %s", updated.GetDateTime("taken_at").Time())
	}
	if updated.GetFloat("lat") != 45.5 || updated.GetFloat("lon") != 8.5 {
		t.Fatalf("target coordinates were overwritten: %f,%f", updated.GetFloat("lat"), updated.GetFloat("lon"))
	}
	metadata, err := metadataMap(updated)
	if err != nil {
		t.Fatal(err)
	}
	if util.AssetMetadataString(metadata, "source_file") != "target.jpg" {
		t.Fatalf("target source_file metadata was overwritten: %q", util.AssetMetadataString(metadata, "source_file"))
	}
	assertMergedAssetSnapshotContains(t, updated, source.Id, "source.jpg")
}

func TestPersistAssetContentHashSkipsAssetUpdateHooks(t *testing.T) {
	app := setupAssetMergeTestApp(t)
	defer app.Cleanup()

	asset := createTestAsset(t, app, "user1", "", "")

	updateHooks := 0
	app.OnRecordAfterUpdateSuccess("assets").BindFunc(func(e *core.RecordEvent) error {
		updateHooks++
		return e.Next()
	})

	hash := strings.Repeat("a", 64)
	if err := persistAssetContentHash(app, asset, hash); err != nil {
		t.Fatal(err)
	}
	if updateHooks != 0 {
		t.Fatalf("asset update hooks fired %d times, want 0", updateHooks)
	}

	updated, err := app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := metadataMap(updated)
	if err != nil {
		t.Fatal(err)
	}
	if got := util.AssetMetadataString(metadata, "content_hash"); got != hash {
		t.Fatalf("content_hash = %q, want %q", got, hash)
	}
}

func TestPersistAssetContentHashDoesNotOverwriteConcurrentRecordChanges(t *testing.T) {
	app := setupAssetMergeTestApp(t)
	defer app.Cleanup()

	asset := createTestAsset(t, app, "user1", "", "")
	asset.Set("file", "photo.jpg")
	if err := app.Save(asset); err != nil {
		t.Fatal(err)
	}
	stale := asset.Clone()

	current, err := app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	current.Set("lat", 46.123)
	if err := app.Save(current); err != nil {
		t.Fatal(err)
	}

	hash := strings.Repeat("b", 64)
	if err := persistAssetContentHash(app, stale, hash); err != nil {
		t.Fatal(err)
	}

	updated, err := app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.GetFloat("lat"); got != 46.123 {
		t.Fatalf("lat = %f, want 46.123", got)
	}
	metadata, err := metadataMap(updated)
	if err != nil {
		t.Fatal(err)
	}
	if got := util.AssetMetadataString(metadata, "content_hash"); got != hash {
		t.Fatalf("content_hash = %q, want %q", got, hash)
	}
}

func TestPersistAssetContentHashSkipsStaleFileSnapshot(t *testing.T) {
	app := setupAssetMergeTestApp(t)
	defer app.Cleanup()

	asset := createTestAsset(t, app, "user1", "", "")
	asset.Set("file", "old.jpg")
	if err := app.Save(asset); err != nil {
		t.Fatal(err)
	}
	stale := asset.Clone()

	current, err := app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	current.Set("file", "new.jpg")
	if err := app.Save(current); err != nil {
		t.Fatal(err)
	}

	if err := persistAssetContentHash(app, stale, strings.Repeat("c", 64)); err != nil {
		t.Fatal(err)
	}

	updated, err := app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := metadataMap(updated)
	if err != nil {
		t.Fatal(err)
	}
	if got := util.AssetMetadataString(metadata, "content_hash"); got != "" {
		t.Fatalf("content_hash = %q, want empty", got)
	}
}

func setupAssetMergeTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(
		&core.TextField{Name: "type"},
		&core.TextField{Name: "storage_mode"},
		&core.TextField{Name: "remote_status"},
		&core.TextField{Name: "file"},
		&core.TextField{Name: "author"},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.DateField{Name: "taken_at"},
		&core.NumberField{Name: "lat"},
		&core.NumberField{Name: "lon"},
		&core.JSONField{Name: "metadata"},
	)
	if err := app.Save(assets); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	for _, config := range []struct {
		name  string
		field string
	}{
		{name: "trail_assets", field: "trail"},
		{name: "waypoint_assets", field: "waypoint"},
		{name: "summit_log_assets", field: "summit_log"},
	} {
		collection := core.NewBaseCollection(config.name)
		collection.Fields.Add(
			&core.TextField{Name: "asset"},
			&core.TextField{Name: config.field},
		)
		if err := app.Save(collection); err != nil {
			app.Cleanup()
			t.Fatal(err)
		}
	}

	return app
}

func assertMergedAssetSnapshotContains(t *testing.T, record *core.Record, sourceID string, expectedSourceFile string) {
	t.Helper()

	metadataJSON, err := json.Marshal(record.Get("metadata"))
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(metadataJSON)
	for _, expected := range []string{sourceID, expectedSourceFile} {
		if !strings.Contains(serialized, expected) {
			t.Fatalf("merged asset snapshot %s does not contain %q", serialized, expected)
		}
	}
}

func createTestAsset(t *testing.T, app core.App, author string, provider string, externalID string) *core.Record {
	t.Helper()

	record := core.NewRecord(mustFindAssetMergeTestCollection(t, app, "assets"))
	record.Set("type", "photo")
	record.Set("storage_mode", "copy")
	record.Set("remote_status", "available")
	record.Set("author", author)
	record.Set("external_provider", provider)
	record.Set("external_id", externalID)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func createTestAssetLink(t *testing.T, app core.App, collectionName string, field string, targetID string, assetID string) *core.Record {
	t.Helper()

	record := core.NewRecord(mustFindAssetMergeTestCollection(t, app, collectionName))
	record.Set("asset", assetID)
	record.Set(field, targetID)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func assertAssetLinkExists(t *testing.T, app core.App, collectionName string, field string, targetID string, assetID string) {
	t.Helper()

	records, err := app.FindRecordsByFilter(collectionName, "asset={:asset} && "+field+"={:target}", "", 1, 0, map[string]any{
		"asset":  assetID,
		"target": targetID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatalf("missing %s link for asset %s and %s %s", collectionName, assetID, field, targetID)
	}
}

func assertNoAssetLinks(t *testing.T, app core.App, assetID string) {
	t.Helper()

	for _, collectionName := range []string{"trail_assets", "waypoint_assets", "summit_log_assets"} {
		records, err := app.FindRecordsByFilter(collectionName, "asset={:asset}", "", 1, 0, map[string]any{
			"asset": assetID,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(records) > 0 {
			t.Fatalf("unexpected source links in %s", collectionName)
		}
	}
}

func mustFindAssetMergeTestCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}
	return collection
}
