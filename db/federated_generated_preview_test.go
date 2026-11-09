package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"image"
	"image/png"
	"io"
	"pocketbase/util"
	"testing"
)

func TestFederatedRealPhotosReplaceGeneratedRouteAssets(t *testing.T) {
	for _, collection := range []string{"trails", "summit_logs"} {
		t.Run(collection, func(t *testing.T) {
			f, actor := placeholderFixture(t)
			target := savePlaceholderTarget(t, f, actor, collection, placeholderGPX)
			generated := onlyGeneratedPlaceholder(t, f.app, target)
			field, path := "trail", "trail"
			if collection == "summit_logs" {
				field, path = "summit_log", "summit-log"
			}
			target.Set("iri", "https://b.example/api/v1/"+path+"/"+target.Id)
			if err := f.app.Save(target); err != nil {
				t.Fatal(err)
			}
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			file, err := filesystem.NewFileFromBytes(encoded.Bytes(), "camera.png")
			if err != nil {
				t.Fatal(err)
			}
			photoURL := "https://b.example/api/v1/assets/photograph/file"
			ctx := util.WithPreparedFederatedFiles(t.Context(), []util.PreparedFederatedFile{{URL: photoURL, OriginIRI: target.GetString("iri"), File: file}})
			for range 2 {
				if err := util.ReconcileFederatedPhotoAssets(f.app, ctx, field, target.Id, actor.Id, []util.FederatedPhoto{{CanonicalID: "https://b.example/api/v1/assets/photograph", FileURL: photoURL}}); err != nil {
					t.Fatal(err)
				}
			}
			links := placeholderLinks(t, f.app, target)
			if len(links) != 1 {
				t.Fatalf("real photo links=%d, want one", len(links))
			}
			asset, err := f.app.FindRecordById("assets", links[0].GetString("asset"))
			if err != nil || asset.Id == generated.Id || util.IsGeneratedRoutePreviewAsset(asset) || asset.GetString("author") != actor.Id {
				t.Fatalf("real photo did not replace generated asset: %v %v", asset, err)
			}
			if _, err := f.app.FindRecordById("assets", generated.Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("orphaned generated asset survived: %v", err)
			}
			fs, err := f.app.NewFilesystem()
			if err != nil {
				t.Fatal(err)
			}
			defer fs.Close()
			reader, err := fs.GetReader(asset.BaseFilesPath() + "/" + asset.GetString("file"))
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			body, err := io.ReadAll(reader)
			if err != nil || !bytes.Equal(body, encoded.Bytes()) {
				t.Fatalf("real photo bytes changed: %v", err)
			}
		})
	}
}

func TestFederatedPhotoBatchFlushesBeforeThumbnailAndAfterRemovals(t *testing.T) {
	for _, collection := range []string{"trails", "summit_logs"} {
		t.Run(collection, func(t *testing.T) {
			f, actor := placeholderFixture(t)
			target := savePlaceholderTarget(t, f, actor, collection, placeholderGPX)
			generated := onlyGeneratedPlaceholder(t, f.app, target)
			field := "trail"
			if collection == "summit_logs" {
				field = "summit_log"
			}
			var photos []util.FederatedPhoto
			var assets []*core.Record
			for i := range 4 {
				asset := f.photo(actor)
				identity := fmt.Sprintf("https://b.example/api/v1/assets/photo%d", i)
				asset.Set("external_provider", util.FederationAssetProvider)
				asset.Set("external_id", identity)
				if err := f.app.Save(asset); err != nil {
					t.Fatal(err)
				}
				assets = append(assets, asset)
				photos = append(photos, util.FederatedPhoto{CanonicalID: identity, IsThumbnail: i == 2})
			}
			creates := 0
			f.app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
				if util.IsGeneratedRoutePreviewAsset(e.Record) {
					creates++
				}
				return e.Next()
			})
			err := f.app.RunInTransaction(func(tx core.App) error {
				// Exercise the thumbnail pass's early preserveUnlisted return.
				if err := util.ReconcileFederatedPhotoAssets(tx, t.Context(), field, target.Id, actor.Id, photos, true); err != nil {
					return err
				}
				links := placeholderLinks(t, tx, target)
				if len(links) != len(photos) {
					t.Fatalf("links=%d, want %d final real links inside tx", len(links), len(photos))
				}
				for _, link := range links {
					assetID := link.GetString("asset")
					if assetID == generated.Id {
						t.Fatal("generated link survived/reappeared during thumbnail update")
					}
					if field == "trail" && link.GetBool("is_thumbnail") != (assetID == assets[2].Id) {
						t.Fatal("thumbnail pass did not see the post-addition link set")
					}
				}
				if _, err := tx.FindRecordById("assets", generated.Id); !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("generated asset survived first flush: %v", err)
				}
				if util.DeferTargetSync(tx, "probe", nil) {
					t.Fatal("reconciliation returned with an active scope")
				}
				return nil
			})
			if err != nil || creates != 0 {
				t.Fatalf("add batch: error=%v extra generated creates=%d", err, creates)
			}
			err = f.app.RunInTransaction(func(tx core.App) error {
				if err := util.ReconcileFederatedPhotoAssets(tx, t.Context(), field, target.Id, actor.Id, nil); err != nil {
					return err
				}
				// The removal batch must flush before its helper returns,
				// not merely after the enclosing transaction commits.
				onlyGeneratedPlaceholder(t, tx, target)
				for _, asset := range assets {
					if _, err := tx.FindRecordById("assets", asset.Id); !errors.Is(err, sql.ErrNoRows) {
						t.Fatalf("unlisted orphan survived: %s: %v", asset.Id, err)
					}
				}
				return nil
			})
			if err != nil || creates != 1 {
				t.Fatalf("removal batch: error=%v generated creates=%d, want one", err, creates)
			}
			onlyGeneratedPlaceholder(t, f.app, target)
		})
	}
}
