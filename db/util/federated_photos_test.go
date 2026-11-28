package util

import (
	"encoding/json"
	"fmt"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestFederatedPhotoAttachmentsOmitGeneratedPreviews(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()
	asset := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/preview")
	asset.Set("file", "map.webp")
	asset.Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
	if err := app.Save(asset); err != nil {
		t.Fatal(err)
	}
	image := FederatedPhotoImage(asset, "https://b.example")
	data, err := json.Marshal(image)
	if err != nil {
		t.Fatal(err)
	}
	var attachment pub.Object
	if err := json.Unmarshal(data, &attachment); err != nil {
		t.Fatal(err)
	}
	if photo, ok := FederatedPhotoFromAttachment(&attachment); ok {
		t.Fatalf("origin preview imported: %+v", photo)
	}

	legacy := &pub.Object{Type: pub.ImageType, URL: pub.IRI("https://a.example/photo.jpg")}
	photo, ok := FederatedPhotoFromAttachment(legacy)
	if !ok || photo.CanonicalID != photo.FileURL || photo.GeneratedKind != "" {
		t.Fatalf("legacy attachment changed: %+v", photo)
	}
}

func TestActivityPubTrailPrefersPhotographsWithinItsAttachmentLimit(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	for _, realPhotoCount := range []int{0, 3} {
		t.Run(fmt.Sprintf("real-photos=%d", realPhotoCount), func(t *testing.T) {
			app := setupFederatedAssetsTestApp(t)
			defer app.Cleanup()
			save := func(model core.Model) {
				t.Helper()
				if err := app.Save(model); err != nil {
					t.Fatal(err)
				}
			}
			categories := core.NewBaseCollection("categories")
			categories.Fields.Add(&core.TextField{Name: "name"})
			save(categories)
			subcategories := core.NewBaseCollection("subcategories")
			subcategories.Fields.Add(&core.TextField{Name: "name"})
			save(subcategories)
			tags := core.NewBaseCollection("tags")
			tags.Fields.Add(&core.TextField{Name: "name"})
			save(tags)
			actors, err := app.FindCollectionByNameOrId("activitypub_actors")
			if err != nil {
				t.Fatal(err)
			}
			trails := core.NewBaseCollection("trails")
			trails.Fields.Add(
				&core.TextField{Name: "name"}, &core.BoolField{Name: "public"},
				&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
				&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1},
				&core.RelationField{Name: "subcategory", CollectionId: subcategories.Id, MaxSelect: 1},
				&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 99},
			)
			save(trails)
			trail := core.NewRecord(trails)
			trail.Set("author", "actor0000000001")
			trail.Set("public", true)
			save(trail)
			preview := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/preview")
			preview.Set("file", "map.webp")
			preview.Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
			save(preview)
			link := createFederatedTestLink(t, app, "trail_assets", "trail", trail.Id, preview.Id)
			link.Set("is_thumbnail", true)
			save(link)
			for i := range realPhotoCount {
				asset := createFederatedTestAsset(t, app, "actor0000000001", fmt.Sprintf("https://a.example/api/v1/assets/real%d", i))
				asset.Set("file", fmt.Sprintf("camera%d.jpg", i))
				save(asset)
				createFederatedTestLink(t, app, "trail_assets", "trail", trail.Id, asset.Id)
			}
			object, err := ObjectFromTrail(app, trail, nil)
			if err != nil {
				t.Fatal(err)
			}
			attachments, err := ActivityPubAttachmentItems(object.Attachment)
			if err != nil {
				t.Fatal(err)
			}
			wantCount := realPhotoCount

			if attachments.Count() != uint(wantCount) {
				t.Fatalf("AP attachments = %d, want %d", attachments.Count(), wantCount)
			}
			for _, item := range attachments.Collection() {
				attachment, err := pub.ToObject(item)
				if err != nil {
					t.Fatal(err)
				}
				photo, ok := FederatedPhotoFromAttachment(attachment)
				if !ok || photo.GeneratedKind != "" {
					t.Fatalf("wrong AP preview priority/marker: %+v", photo)
				}
			}
		})
	}
}
