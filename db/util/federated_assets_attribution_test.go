package util

import (
	"errors"
	"github.com/pocketbase/pocketbase/core"
	"testing"
)

func TestCanonicalPhotoOwnershipRepairAndRemovalPreserveTransactionsAndSharedLinks(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	app := setupFederatedAssetsTestApp(t)
	defer app.Cleanup()
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actors.Fields.Add(&core.TextField{Name: "iri"}, &core.BoolField{Name: "is_local"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	old, err := app.FindRecordById(actors, "actor0000000001")
	if err != nil {
		t.Fatal(err)
	}
	old.Set("iri", "https://a.example/users/alice")
	if err := app.Save(old); err != nil {
		t.Fatal(err)
	}
	corrected := core.NewRecord(actors)
	corrected.Set("iri", "https://a.example:9443/users/carol")
	if err := app.Save(corrected); err != nil {
		t.Fatal(err)
	}
	asset := createFederatedTestAsset(t, app, old.Id, "https://a.example:9443/api/v1/assets/photo")
	shared := createFederatedTestAsset(t, app, old.Id, "https://a.example:9443/api/v1/assets/shared")
	unboundOrigin := createFederatedTestAsset(t, app, old.Id, "https://a.example/api/v1/assets/older-cache")
	local := core.NewRecord(actors)
	local.Set("iri", "https://b.example/users/alice")
	local.Set("is_local", true)
	if err := app.Save(local); err != nil {
		t.Fatal(err)
	}
	independentLocal := createFederatedTestAsset(t, app, local.Id, "https://a.example:9443/api/v1/assets/local-copy")
	createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", independentLocal.Id)
	for _, photo := range []*core.Record{asset, shared, unboundOrigin} {
		createFederatedTestLink(t, app, "summit_log_assets", "summit_log", "sl1", photo.Id)
	}
	createFederatedTestLink(t, app, "trail_assets", "trail", "trail1", shared.Id)
	photos := []FederatedPhoto{{CanonicalID: asset.GetString("external_id")}, {CanonicalID: shared.GetString("external_id")}}
	sentinel := errors.New("rollback")
	err = app.RunInTransaction(func(tx core.App) error {
		if err := ReconcileFederatedPhotoAssets(tx, t.Context(), "summit_log", "sl1", corrected.Id, photos); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	stored, err := app.FindRecordById("assets", asset.Id)
	if err != nil || stored.GetString("author") != old.Id {
		t.Fatalf("rollback changed ownership: %v %v", stored, err)
	}
	if err := app.RunInTransaction(func(tx core.App) error {
		return ReconcileFederatedPhotoAssets(tx, t.Context(), "summit_log", "sl1", corrected.Id, photos)
	}); err != nil {
		t.Fatal(err)
	}
	for _, photo := range []*core.Record{asset, shared} {
		stored, err := app.FindRecordById("assets", photo.Id)
		if err != nil || stored.GetString("author") != corrected.Id {
			t.Fatalf("canonical photo owner not repaired: %v %v", stored, err)
		}
	}
	if _, err := app.FindRecordById("assets", unboundOrigin.Id); err == nil {
		t.Fatal("removed old-owner federation photo leaked as an orphan")
	}
	if err := ReconcileFederatedPhotoAssets(app, t.Context(), "summit_log", "sl1", corrected.Id, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := app.FindRecordById("assets", asset.Id); err == nil {
		t.Fatal("removed repaired photo survived as orphan")
	}
	if _, err := app.FindRecordById("assets", shared.Id); err != nil {
		t.Fatal("photo with independent link deleted", err)
	}
	stored, err = app.FindRecordById("assets", independentLocal.Id)
	if err != nil || stored.GetString("author") != local.Id {
		t.Fatalf("independent local-owned federation photo changed or deleted: %v %v", stored, err)
	}
}
