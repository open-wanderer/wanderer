package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestFederatedAuthorCorrectionReusesCanonicalAssetDespiteUniqueIndex(t *testing.T) {
	for _, scenario := range []string{"complete", "empty-keeper", "existing-link", "independent-link", "trail-thumbnail", "empty-linked-keeper"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRecordHooksFixture(t)
			previous := f.save("activitypub_actors", map[string]any{"iri": "https://a.example/users/previous", "preferred_username": "previous", "domain": "a.example"})
			corrected := f.save("activitypub_actors", map[string]any{"iri": "https://b.example/users/correct", "preferred_username": "correct", "domain": "b.example"})
			trail := f.save("trails", map[string]any{"name": "Remote", "iri": "https://a.example/api/v1/trail/t", "author": previous.Id})
			log := f.save("summit_logs", map[string]any{"trail": trail.Id, "author": corrected.Id, "iri": "https://b.example/api/v1/summit-log/l"})
			targetField, targetID, linkCollection := "summit_log", log.Id, "summit_log_assets"
			if scenario == "trail-thumbnail" {
				targetField, targetID, linkCollection = "trail", trail.Id, "trail_assets"
			}
			canonical := "https://b.example/api/v1/assets/photo"
			var sourceBytes bytes.Buffer
			if err := png.Encode(&sourceBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			photo := func(author *core.Record, name string, empty bool, metadata map[string]any) *core.Record {
				t.Helper()
				fields := map[string]any{"type": "photo", "storage_mode": "copy", "author": author.Id, "external_provider": "federation", "external_id": canonical, "metadata": metadata}
				if !empty {
					file, err := filesystem.NewFileFromBytes(sourceBytes.Bytes(), name)
					if err != nil {
						t.Fatal(err)
					}
					fields["file"] = file
				}
				return f.save("assets", fields)
			}
			old := photo(previous, "old.png", false, map[string]any{"preserved": "old-only"})
			keeper := photo(corrected, "keeper.png", scenario == "empty-keeper" || scenario == "empty-linked-keeper", map[string]any{"preferred": "current"})
			oldLink := f.save(linkCollection, map[string]any{targetField: targetID, "asset": old.Id, "is_thumbnail": true})
			if scenario == "existing-link" || scenario == "empty-linked-keeper" {
				f.save("summit_log_assets", map[string]any{"summit_log": log.Id, "asset": keeper.Id})
			}
			if scenario == "independent-link" {
				f.save("trail_assets", map[string]any{"trail": trail.Id, "asset": old.Id})
			}
			for range 2 {
				if err := util.ReconcileFederatedPhotoAssets(f.app, t.Context(), targetField, targetID, corrected.Id, []util.FederatedPhoto{{CanonicalID: canonical, FileURL: "https://a.example/unreachable"}}); err != nil {
					t.Fatal(err)
				}
			}
			links, err := f.app.FindRecordsByFilter(linkCollection, targetField+"={:target}", "", -1, 0, dbx.Params{"target": targetID})
			if err != nil || len(links) != 1 || links[0].GetString("asset") != keeper.Id {
				t.Fatalf("corrected canonical photo not linked once: %v %v", links, err)
			}
			if scenario != "existing-link" && scenario != "empty-linked-keeper" && links[0].Id != oldLink.Id {
				t.Fatal("reassignment changed the original link's identity/order")
			}
			if scenario == "trail-thumbnail" && !links[0].GetBool("is_thumbnail") {
				t.Fatal("author correction discarded the trail thumbnail")
			}
			current, err := f.app.FindRecordById("assets", keeper.Id)
			if err != nil || current.GetString("author") != corrected.Id || current.GetString("file") == "" {
				t.Fatalf("keeper materialization lost: %v %v", current, err)
			}
			metadata := util.AssetMetadataWithProvenance(current)
			if metadata["preferred"] != "current" || metadata["preserved"] != "old-only" {
				t.Fatalf("reuse discarded metadata: %#v", metadata)
			}
			body, err := os.ReadFile(filepath.Join(f.app.DataDir(), "storage", current.BaseFilesPath(), current.GetString("file")))
			if err != nil || !bytes.Equal(body, sourceBytes.Bytes()) {
				t.Fatalf("keeper file bytes lost: %v", err)
			}
			oldExists := f.exists(old)
			if oldExists != (scenario == "independent-link") {
				t.Fatalf("old asset orphan handling wrong: exists=%t", oldExists)
			}
			if scenario == "independent-link" {
				if _, err := os.Stat(filepath.Join(f.app.DataDir(), "storage", old.BaseFilesPath(), old.GetString("file"))); err != nil {
					t.Fatal("independent source link lost its file", err)
				}
			} else {
				// PocketBase removes an orphan's storage prefix asynchronously.
				path := filepath.Join(f.app.DataDir(), "storage", old.BaseFilesPath(), old.GetString("file"))
				deadline := time.Now().Add(time.Second)
				for {
					_, err := os.Stat(path)
					if os.IsNotExist(err) {
						break
					}
					if err != nil || time.Now().After(deadline) {
						t.Fatalf("orphaned older file survived reuse: %v", err)
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
		})
	}
}
