package util

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// Prepared for future authorized integration execution. Under the no-start
// boundary this test is COMPILED ONLY, never selected or executed.
func TestReconcileCorrectionIntoLinkedKeeperKeepsBothPhotosAndCover(t *testing.T) {
	for _, sourceFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "source-before-keeper", false: "keeper-before-source"}[sourceFirst], func(t *testing.T) {
			t.Setenv("ORIGIN", "https://local.example")
			app := setupFederatedAssetsTestApp(t)
			defer app.Cleanup()
			actors, err := app.FindCollectionByNameOrId("activitypub_actors")
			if err != nil {
				t.Fatal(err)
			}
			actors.Fields.Add(&core.TextField{Name: "iri"})
			if err := app.Save(actors); err != nil {
				t.Fatal(err)
			}
			for id, iri := range map[string]string{
				"actor0000000001": "https://wrong.example/users/old",
				"actor0000000002": "https://remote.example/users/author",
			} {
				actor, err := app.FindRecordById(actors, id)
				if err != nil {
					t.Fatal(err)
				}
				actor.Set("iri", iri)
				if err := app.Save(actor); err != nil {
					t.Fatal(err)
				}
			}
			const author = "actor0000000002"
			const legacy = "https://remote.example/api/files/trails/id/photo.jpg"
			source := createFederatedTestAsset(t, app, "actor0000000001", legacy)
			keeper := createFederatedTestAsset(t, app, author, legacy)
			p1 := FederatedPhoto{CanonicalID: "https://remote.example/api/v1/assets/p1", SourceName: "photo.jpg", IsThumbnail: true}
			p2 := FederatedPhoto{CanonicalID: "https://remote.example/api/v1/assets/p2", SourceName: p1.SourceName}
			holder := createFederatedTestAsset(t, app, author, p2.CanonicalID)
			// In this deliberately minimal schema file is text. Nonempty keeper/
			// holder values avoid filesystem materialization in the reuse helper.
			for _, asset := range []*core.Record{keeper, holder} {
				asset.Set("file", "photo.jpg")
				if err := app.Save(asset); err != nil {
					t.Fatal(err)
				}
			}
			order := []*core.Record{source, keeper}
			if !sourceFirst {
				order = []*core.Record{keeper, source}
			}
			for i, asset := range order {
				link := createFederatedTestLink(t, app, "trail_assets", "trail", "trail1", asset.Id)
				link.Set("created", []string{"2026-01-01 00:00:00.000Z", "2026-01-02 00:00:00.000Z"}[i])
				if err := app.SaveNoValidate(link); err != nil {
					t.Fatal(err)
				}
			}
			if err := ReconcileFederatedPhotoAssets(app, WithPreparedFederatedFiles(t.Context(), nil), "trail", "trail1", author, []FederatedPhoto{p1, p2}); err != nil {
				t.Fatal(err)
			}
			first, err := app.FindRecordById("assets", keeper.Id)
			if err != nil || first.GetString("external_id") != p1.CanonicalID {
				t.Fatalf("P1 keeper disappeared/was overwritten: %v %v", first, err)
			}
			second, err := app.FindRecordById("assets", holder.Id)
			if err != nil || second.GetString("external_id") != p2.CanonicalID {
				t.Fatalf("known P2 holder changed: %v %v", second, err)
			}
			links, err := app.FindRecordsByFilter("trail_assets", "trail='trail1'", "", -1, 0)
			if err != nil || len(links) != 2 {
				t.Fatalf("missing photo links: %d %v", len(links), err)
			}
			for _, link := range links {
				if link.GetBool("is_thumbnail") != (link.GetString("asset") == keeper.Id) {
					t.Fatal("P1 cover lost or moved to P2")
				}
			}
		})
	}
}
