package util

import (
	"context"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestFederatedPhotosForLocalAuthorsRequireLocalTargets(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, field := range []string{"trail", "waypoint", "summit_log"} {
		for _, iri := range []string{"https://a.example/api/v1/object/remote", "https://b.example/api/v1/object/local", ""} {
			t.Run(field+"/"+iri, func(t *testing.T) {
				app := setupFederatedAssetsTestApp(t)
				defer app.Cleanup()
				actor, err := app.FindRecordById("activitypub_actors", "actor0000000001")
				if err != nil {
					t.Fatal(err)
				}
				actor.Set("is_local", true)
				if err := app.Save(actor); err != nil {
					t.Fatal(err)
				}
				collectionName := map[string]string{"trail": "trails", "waypoint": "waypoints", "summit_log": "summit_logs"}[field]
				collection := core.NewBaseCollection(collectionName)
				collection.Fields.Add(&core.TextField{Name: "iri"})
				if err := app.Save(collection); err != nil {
					t.Fatal(err)
				}
				target := core.NewRecord(collection)
				target.Set("iri", iri)
				if err := app.Save(target); err != nil {
					t.Fatal(err)
				}
				// Seed an existing photo to exercise the reuse path as well as
				// reconciliation without performing any network fetch.
				asset := createFederatedTestAsset(t, app, actor.Id, "https://a.example/api/v1/assets/a1")
				input := PhotoAssetInput{
					Author: actor.Id, StorageMode: "link_private",
					ExternalProvider: FederationAssetProvider, ExternalID: asset.GetString("external_id"),
				}
				switch field {
				case "trail":
					input.Trail = target.Id
				case "waypoint":
					input.Waypoint = target.Id
				case "summit_log":
					input.SummitLog = target.Id
				}
				_, createErr := CreatePhotoAsset(app, input)
				photos := []FederatedPhoto{{CanonicalID: asset.GetString("external_id"), FileURL: "https://a.example/file"}}
				reconcileErr := ReconcileFederatedPhotoAssets(app, t.Context(), field, target.Id, actor.Id, photos)
				remote := strings.HasPrefix(iri, "https://a.example/")
				if remote {
					if createErr == nil || reconcileErr == nil {
						t.Fatalf("remote target accepted local author: create=%v reconcile=%v", createErr, reconcileErr)
					}
					assertFederatedLinkCount(t, app, field+"_assets", field, target.Id, 0)
				} else {
					if createErr != nil || reconcileErr != nil {
						t.Fatalf("local target rejected: create=%v reconcile=%v", createErr, reconcileErr)
					}
					assertFederatedLinkCount(t, app, field+"_assets", field, target.Id, 1)
				}
				assertFederatedAssetCount(t, app, 1)
			})
		}
	}
}

func TestReconcileFederatedPhotoAssetsIsIdempotent(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()

	asset := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a1")
	createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", asset.Id)

	photos := []FederatedPhoto{{CanonicalID: "https://a.example/api/v1/assets/a1", FileURL: "https://a.example/api/v1/assets/a1/file"}}
	for range 3 {
		if err := ReconcileFederatedPhotoAssets(app, context.Background(), "summit_log", "sl1", "actor0000000001", photos); err != nil {
			t.Fatal(err)
		}
	}

	assertFederatedAssetCount(t, app, 1)
	assertFederatedLinkCount(t, app, "summit_log_assets", "summit_log", "sl1", 1)
}

func TestReconcileFederatedPhotoAssetsLinksExistingAssetWithoutFetch(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()

	// same photo already materialized for this author on another target
	asset := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a1")
	createFederatedTestLink(t, app, "trail_assets", "trail", "trail1", asset.Id)

	// unreachable FileURL: reaching the fetch would drop the photo silently
	photos := []FederatedPhoto{{CanonicalID: "https://a.example/api/v1/assets/a1", FileURL: "https://a.example/file"}}
	if err := ReconcileFederatedPhotoAssets(app, context.Background(), "summit_log", "sl1", "actor0000000001", photos); err != nil {
		t.Fatal(err)
	}

	assertFederatedAssetCount(t, app, 1)
	assertFederatedLinkCount(t, app, "summit_log_assets", "summit_log", "sl1", 1)
	assertFederatedLinkCount(t, app, "trail_assets", "trail", "trail1", 1)
}

func TestReconcileFederatedPhotoAssetsRemovesUnlistedPhotos(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()

	kept := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a1")
	removed := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a2")
	shared := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a3")
	createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", kept.Id)
	createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", removed.Id)
	createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", shared.Id)
	createFederatedTestLink(t, app, "trail_assets", "trail", "trail1", shared.Id)

	photos := []FederatedPhoto{{CanonicalID: "https://a.example/api/v1/assets/a1", FileURL: "https://a.example/api/v1/assets/a1/file"}}
	if err := ReconcileFederatedPhotoAssets(app, context.Background(), "summit_log", "sl1", "actor0000000001", photos); err != nil {
		t.Fatal(err)
	}

	assertFederatedLinkCount(t, app, "summit_log_assets", "summit_log", "sl1", 1)
	if _, err := app.FindRecordById("assets", removed.Id); err == nil {
		t.Fatal("orphaned asset was not deleted")
	}
	// still linked elsewhere -> unlinked here but kept
	if _, err := app.FindRecordById("assets", shared.Id); err != nil {
		t.Fatal("asset linked to another target was deleted")
	}
	assertFederatedLinkCount(t, app, "trail_assets", "trail", "trail1", 1)
}

func TestReconcileFederatedPhotoAssetsKeepsLegacyPhotosUntouched(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()
	targets := core.NewBaseCollection("summit_logs")
	targets.Fields.Add(&core.TextField{Name: "iri"})
	if err := app.Save(targets); err != nil {
		t.Fatal(err)
	}
	target := core.NewRecord(targets)
	target.Id = "sl1"
	target.Set("iri", "https://a.example/api/v1/summit-log/sl1")
	if err := app.SaveNoValidate(target); err != nil {
		t.Fatal(err)
	}
	legacy := createFederatedTestAsset(t, app, "actor0000000001", "")
	legacy.Set("file", "camera_ab12cd34ef.jpg")
	if err := app.Save(legacy); err != nil {
		t.Fatal(err)
	}
	createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", legacy.Id)
	unmatched := createFederatedTestAsset(t, app, "actor0000000001", "")
	unmatched.Set("file", "unlisted.jpg")
	if err := app.Save(unmatched); err != nil {
		t.Fatal(err)
	}
	createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", unmatched.Id)
	added := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a2")
	photos := []FederatedPhoto{
		{CanonicalID: "https://a.example/api/v1/assets/a1", FileURL: "https://a.example/a1", SourceName: "camera.jpg"},
		{CanonicalID: added.GetString("external_id"), FileURL: "https://a.example/a2", SourceName: "next.jpg"},
	}
	if err := ReconcileFederatedPhotoAssets(app, WithPreparedFederatedFiles(t.Context(), nil), "summit_log", "sl1", "actor0000000001", photos); err != nil {
		t.Fatal(err)
	}
	assertFederatedAssetCount(t, app, 3)
	assertFederatedLinkCount(t, app, "summit_log_assets", "summit_log", "sl1", 3)
	stored, err := app.FindRecordById("assets", legacy.Id)
	if err != nil || stored.GetString("external_provider") != FederationAssetProvider || stored.GetString("external_id") != photos[0].CanonicalID {
		t.Fatalf("legacy asset not adopted: %v %v", stored, err)
	}
	stored, err = app.FindRecordById("assets", unmatched.Id)
	if err != nil || stored.GetString("external_provider") != "" {
		t.Fatal("unmatched legacy asset changed")
	}
}

func TestDeleteAssetIfOrphanedByAuthorKeepsForeignOrphan(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()

	asset := createFederatedTestAsset(t, app, "actor0000000002", "")
	deleted, err := DeleteAssetIfOrphanedByAuthor(app, asset.Id, "actor0000000001")
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("foreign orphaned asset was deleted")
	}
	if _, err := app.FindRecordById("assets", asset.Id); err != nil {
		t.Fatal("foreign orphaned asset was removed")
	}
}

func TestDeleteAssetIfOrphanedByAuthorDeletesOwnedOrphan(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()

	asset := createFederatedTestAsset(t, app, "actor0000000001", "")
	deleted, err := DeleteAssetIfOrphanedByAuthor(app, asset.Id, "actor0000000001")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("owned orphaned asset was not deleted")
	}
	if _, err := app.FindRecordById("assets", asset.Id); err == nil {
		t.Fatal("owned orphaned asset still exists")
	}
}

// A trail's thumbnail is its marked photo, else its first one. The route
// preview the editor generates stands in only for a trail without a real
// photo, as it did when it was stored as an ordinary trail photo.

func TestReconcileFederatedPhotoAssetsSetsTrailThumbnail(t *testing.T) {
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()

	first := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a1")
	marked := createFederatedTestAsset(t, app, "actor0000000001", "https://a.example/api/v1/assets/a2")

	photos := []FederatedPhoto{
		{CanonicalID: "https://a.example/api/v1/assets/a1", FileURL: "https://a.example/a1"},
		{CanonicalID: "https://a.example/api/v1/assets/a2", FileURL: "https://a.example/a2", IsThumbnail: true},
	}
	if err := ReconcileFederatedPhotoAssets(app, context.Background(), "trail", "trail1", "actor0000000001", photos); err != nil {
		t.Fatal(err)
	}

	assertFederatedLinkCount(t, app, "trail_assets", "trail", "trail1", 2)
	if first.Id == marked.Id {
		t.Fatal("test assets unexpectedly share an id")
	}
	for _, asset := range []*core.Record{first, marked} {
		link, err := app.FindFirstRecordByFilter("trail_assets", "trail={:trail} && asset={:asset}", dbx.Params{"trail": "trail1", "asset": asset.Id})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := link.GetBool("is_thumbnail"), asset.Id == marked.Id; got != want {
			t.Fatalf("asset %s is_thumbnail = %t, want %t", asset.Id, got, want)
		}
	}
}
