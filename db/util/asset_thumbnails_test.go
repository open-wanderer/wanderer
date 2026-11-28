package util

import (
	"github.com/pocketbase/pocketbase/core"
	"testing"
)

func TestTrailThumbnailURLsPrefersRealPhotosOverGeneratedPreview(t *testing.T) {
	type photo struct {
		file     string
		metadata map[string]any
		marked   bool
	}
	preview := map[string]any{"generated": map[string]any{"kind": "route-preview"}}
	migratedPreview := map[string]any{"source_file": "route_abcdef1234.webp", "generated": map[string]any{"kind": "route-preview"}}
	for _, tc := range []struct {
		name   string
		photos []photo
		want   int
	}{
		{
			name:   "marked photo",
			photos: []photo{{file: "first.jpg"}, {file: "marked.jpg", marked: true}},
			want:   1,
		},
		{
			name:   "photo over marked preview",
			photos: []photo{{file: "route.webp", metadata: migratedPreview, marked: true}, {file: "photo.jpg"}},
			want:   1,
		},
		{
			name:   "preview without photo",
			photos: []photo{{file: "preview.webp", metadata: preview}},
			want:   0,
		},
		{
			name:   "marked preview",
			photos: []photo{{file: "old.webp", metadata: preview}, {file: "new.webp", metadata: preview, marked: true}},
			want:   1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := setupFederatedAssetsTestApp(t)
			defer app.Cleanup()

			assetIDs := make([]string, len(tc.photos))
			for i, p := range tc.photos {
				asset := createFederatedTestAsset(t, app, "actor0000000001", "")
				asset.Set("file", p.file)
				asset.Set("metadata", p.metadata)
				if err := app.Save(asset); err != nil {
					t.Fatal(err)
				}
				link := createFederatedTestLink(t, app, "trail_assets", "trail", "trail1", asset.Id)
				link.Set("is_thumbnail", p.marked)
				if err := app.Save(link); err != nil {
					t.Fatal(err)
				}
				assetIDs[i] = asset.Id
			}
			trail := core.NewRecord(core.NewBaseCollection("trails"))
			trail.Id = "trail1"

			urls, err := TrailThumbnailURLs(app, []*core.Record{trail}, "")
			if err != nil {
				t.Fatal(err)
			}
			collection, err := app.FindCollectionByNameOrId("assets")
			if err != nil {
				t.Fatal(err)
			}
			want := "/api/v1/files/" + collection.Id + "/" + assetIDs[tc.want] + "/" + tc.photos[tc.want].file
			if urls["trail1"] != want {
				t.Fatalf("thumbnail URL = %q, want %q", urls["trail1"], want)
			}
		})
	}
}
