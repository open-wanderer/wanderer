package util

import (
	"bytes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"image"
	"image/png"
	"testing"
)

func TestFederatedPhotosContinuePastLocalGeneratedPreview(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()
	schema, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	schema.Fields.RemoveByName("file")
	if err := app.Save(schema); err != nil {
		t.Fatal(err)
	}
	schema.Fields.Add(&core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1 << 20})
	if err := app.Save(schema); err != nil {
		t.Fatal(err)
	}
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "author"})
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("iri", "https://a.example/api/v1/trail/route")
	trail.Set("author", "actor0000000001")
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	preview := createFederatedTestAsset(t, app, "actor0000000001", "")
	preview.Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
	if err := app.Save(preview); err != nil {
		t.Fatal(err)
	}
	createFederatedTestLink(t, app, "trail_assets", "trail", trail.Id, preview.Id)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(encoded.Bytes(), "camera.png")
	if err != nil {
		t.Fatal(err)
	}
	url := "https://a.example/api/v1/assets/photo/file"
	ctx := WithPreparedFederatedFiles(t.Context(), []PreparedFederatedFile{{URL: url, OriginIRI: trail.GetString("iri"), File: file}})
	for range 2 {
		if err := ReconcileFederatedPhotoAssets(app, ctx, "trail", trail.Id, "actor0000000001", []FederatedPhoto{{CanonicalID: "https://a.example/api/v1/assets/photo", FileURL: url}}); err != nil {
			t.Fatal(err)
		}
	}
	assets, err := PhotoAssetsForTarget(app, "trail", trail.Id, -1)
	if err != nil {
		t.Fatal(err)
	}
	real := 0
	for _, asset := range assets {
		if asset.GetString("external_id") == "https://a.example/api/v1/assets/photo" && asset.GetString("file") != "" {
			real++
		}
	}
	if real != 1 {
		t.Fatalf("generated preview prevented real photo materialization: assets=%d real=%d", len(assets), real)
	}
}
