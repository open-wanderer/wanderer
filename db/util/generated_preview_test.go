package util

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestGeneratedPreviewRecognizesStoredPocketBaseFilename(t *testing.T) {
	collection := core.NewBaseCollection("assets")
	collection.Fields.Add(&core.FileField{Name: "file", MaxSelect: 1}, &core.TextField{Name: "external_provider"}, &core.DateField{Name: "taken_at"}, &core.JSONField{Name: "metadata"})
	for _, extension := range []string{"png", "webp"} {
		file, err := filesystem.NewFileFromBytes([]byte("preview"), "wanderer-route-preview."+extension)
		if err != nil {
			t.Fatal(err)
		}
		asset := core.NewRecord(collection)
		asset.Set("file", file.Name)
		if !IsGeneratedRoutePreviewAsset(asset) {
			t.Fatalf("actual PocketBase filename %q was not recognized", file.Name)
		}
		asset.Set("external_provider", "immich")
		if IsGeneratedRoutePreviewAsset(asset) {
			t.Fatal("provider photo was misclassified")
		}
		asset.Set("external_provider", "")
		asset.Set("taken_at", time.Now())
		if IsGeneratedRoutePreviewAsset(asset) {
			t.Fatal("recorded photo was misclassified")
		}
	}
	for _, name := range []string{"wanderer-route-preview-holiday.png", "wanderer_route_preview_holiday.png", "route_abcdefghij.webp"} {
		asset := core.NewRecord(collection)
		asset.Set("file", name)
		if IsGeneratedRoutePreviewAsset(asset) {
			t.Fatalf("ordinary filename %q was misclassified", name)
		}
	}
}

func TestGeneratedPreviewRecognizesMigratedRoutePlaceholder(t *testing.T) {
	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(&core.TextField{Name: "file"}, &core.TextField{Name: "type"}, &core.TextField{Name: "storage_mode"}, &core.JSONField{Name: "metadata"})
	placeholder := core.NewRecord(assets)
	placeholder.Load(map[string]any{"file": "wanderer_route_placeholder_abcdefghij.svg", "type": "photo", "storage_mode": "copy", "metadata": map[string]any{"generated": map[string]any{"kind": "route-preview"}}})
	if !IsGeneratedRoutePreviewAsset(placeholder) {
		t.Fatal("migrated server placeholder lost its generated classification")
	}
	real := core.NewRecord(assets)
	real.Load(map[string]any{"file": "summit.jpg", "type": "photo", "storage_mode": "copy"})
	links := core.NewBaseCollection("trail_assets")
	links.Fields.Add(&core.BoolField{Name: "is_thumbnail"})
	marked := core.NewRecord(links)
	marked.Set("is_thumbnail", true)
	marked.SetExpand(map[string]any{"asset": placeholder})
	unmarked := core.NewRecord(links)
	unmarked.SetExpand(map[string]any{"asset": real})
	if got := CardPhotoThumbnail([]*core.Record{marked, unmarked}); got != 1 {
		t.Fatalf("generated placeholder outranked the real photo: %d", got)
	}
	if got := CardPhotoThumbnail([]*core.Record{marked}); got != 0 {
		t.Fatalf("placeholder-only trail lost its fallback: %d", got)
	}
}
