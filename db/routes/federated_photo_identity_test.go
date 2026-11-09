package routes

import (
	"bytes"
	"context"
	"errors"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
	"image"
	"image/png"
	"net/http"
	"pocketbase/util"
	"strings"
	"testing"
)

func TestFederatedPhotoAdoptionReusesPhotoIdentityAndMatchedLink(t *testing.T) {
	for _, kind := range []string{"migrated", "legacy_url"} {
		for _, alreadyLinked := range []bool{false, true} {
			for _, sharedLegacy := range []bool{false, true} {
				name := kind
				if alreadyLinked {
					name += "/already_linked"
				}
				if sharedLegacy {
					name += "/shared_legacy"
				}
				t.Run(name, func(t *testing.T) {
					f := newDirectedShareFixture(t)
					var definition string
					if err := f.app.DB().NewQuery("SELECT sql FROM sqlite_master WHERE name='idx_assets_external'").Row(&definition); err != nil || !strings.Contains(definition, "UNIQUE INDEX") {
						t.Fatal("fixture lacks federation identity uniqueness")
					}
					other := saveRemoteChildRecord(t, f.app, "trails", map[string]any{"name": "Other", "author": f.remote.Id, "iri": "https://b.example/api/v1/trail/othertrail00001"})
					var body bytes.Buffer
					if err := png.Encode(&body, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
						t.Fatal(err)
					}
					photo := func(name, identity string, trail *core.Record) *core.Record {
						t.Helper()
						file, err := filesystem.NewFileFromBytes(body.Bytes(), name)
						if err != nil {
							t.Fatal(err)
						}
						file.Name = name
						return testAddPhoto(t, f.app, trail, file, identity)
					}
					sourceName := "camera.png"
					oldIdentity := ""
					if kind == "legacy_url" {
						oldIdentity = "https://b.example/api/v1/files/trails/origintrail0001/" + sourceName
					}
					legacy := photo(util.FederatedSourceFileName(sourceName), oldIdentity, f.trail)
					identity := "https://b.example/api/v1/assets/canonicalphoto1"
					keeper := photo("keeper.png", identity, other)
					if sharedLegacy {
						saveRemoteChildRecord(t, f.app, "trail_assets", map[string]any{"trail": other.Id, "asset": legacy.Id})
					}
					matched, err := f.app.FindFirstRecordByFilter("trail_assets", "asset={:asset} && trail={:trail}", dbx.Params{"asset": legacy.Id, "trail": f.trail.Id})
					if err != nil {
						t.Fatal(err)
					}
					matched.Set("is_thumbnail", true)
					matched.SetRaw("created", types.NowDateTime())
					if err := f.app.Save(matched); err != nil {
						t.Fatal(err)
					}
					order := matched.GetString("created")
					if alreadyLinked {
						saveRemoteChildRecord(t, f.app, "trail_assets", map[string]any{"trail": f.trail.Id, "asset": keeper.Id})
					}
					data := map[string]any{"expand": map[string]any{"trail_assets_via_trail": []any{map[string]any{"is_thumbnail": false, "expand": map[string]any{"asset": map[string]any{"id": "canonicalphoto1", "type": "photo", "file": sourceName, "collectionId": "assets"}}}}}}
					downloads := 0
					previous := fetchRemoteSyncMedia
					t.Cleanup(func() { fetchRemoteSyncMedia = previous })
					fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
						downloads++
						return nil, func() {}, errors.New("unexpected photo download")
					}
					writes := 0
					f.app.OnRecordUpdate("assets", "trail_assets").BindFunc(func(e *core.RecordEvent) error { writes++; return e.Next() })
					f.app.OnRecordDelete("assets", "trail_assets").BindFunc(func(e *core.RecordEvent) error { writes++; return e.Next() })
					afterFirst := 0
					for pass := 0; pass < 2; pass++ {
						collector := remoteMediaCollector{seen: map[string]bool{}}
						collector.record(f.app, "trail", "origintrail0001", "https://b.example", f.trail.GetString("iri"), data)
						if len(collector.requests) != 0 {
							t.Fatal("prefetch ignored the existing photo identity")
						}
						ctx, cleanup, err := prepareRemoteMedia(t.Context(), collector.requests)
						if err != nil {
							t.Fatal(err)
						}
						err = syncRecordPhotos(f.app, ctx, "trail", f.trail.Id, "origintrail0001", f.remote.Id, "https://b.example", data, true)
						cleanup()
						if err != nil {
							t.Fatalf("photo identity reuse failed: %v", err)
						}
						links, err := f.app.FindRecordsByFilter("trail_assets", "trail={:trail}", "", -1, 0, dbx.Params{"trail": f.trail.Id})
						if err != nil || len(links) != 1 || links[0].GetString("asset") != keeper.Id || links[0].Id != matched.Id || !links[0].GetBool("is_thumbnail") || links[0].GetString("created") != order {
							t.Fatalf("canonical reuse lost the matched link, cover or order: links=%v error=%v expected_id=%s expected_created=%s", links, err, matched.Id, order)
						}
						_, legacyErr := f.app.FindRecordById("assets", legacy.Id)
						if sharedLegacy && legacyErr != nil {
							t.Fatal("canonical reuse deleted an independently linked legacy photo")
						}
						if !sharedLegacy && legacyErr == nil {
							t.Fatal("orphan legacy asset survived canonical reuse")
						}
						if downloads != 0 {
							t.Fatal("canonical reuse downloaded photo bytes")
						}
						if pass == 0 {
							afterFirst = writes
						} else if writes != afterFirst {
							t.Fatal("second canonical photo sync wrote records")
						}
					}
				})
			}
		}
	}
}

func TestFederatedPhotoAdoptionFailureKeepsMetadataAndUnlistedPhotos(t *testing.T) {
	f := newDirectedShareFixture(t)
	var body bytes.Buffer
	if err := png.Encode(&body, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(body.Bytes(), "camera.png")
	if err != nil {
		t.Fatal(err)
	}
	legacy := testAddPhoto(t, f.app, f.trail, file, "https://b.example/api/v1/files/trails/origintrail0001/camera.png")
	other, err := filesystem.NewFileFromBytes(body.Bytes(), "kept.png")
	if err != nil {
		t.Fatal(err)
	}
	kept := testAddPhoto(t, f.app, f.trail, other, "https://b.example/api/v1/assets/keptphoto000001")
	identity := "https://b.example/api/v1/assets/canonicalphoto1"
	f.app.OnRecordUpdate("assets").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id == legacy.Id && e.Record.GetString("external_id") == identity {
			return errors.New("photo adoption unavailable")
		}
		return e.Next()
	})
	f.client(t, func(*http.Request) (map[string]any, int) {
		data := f.payload()
		data["expand"].(map[string]any)["trail_assets_via_trail"] = []any{map[string]any{"expand": map[string]any{"asset": map[string]any{"id": "canonicalphoto1", "type": "photo", "file": "camera.png", "collectionId": "assets"}}}}
		return data, 200
	})
	result, err := performFullSync(f.app, t.Context(), nil, f.trail)
	if err != nil || result.GetString("name") != "Current private trail" {
		t.Fatalf("failed photo adoption stopped metadata sync: %v", err)
	}
	for _, asset := range []*core.Record{legacy, kept} {
		if _, err := f.app.FindRecordById("assets", asset.Id); err != nil {
			t.Fatal("incomplete adoption removed a retained photo")
		}
	}
	if got := len(testPhotoAssets(t, f.app, f.trail)); got != 2 {
		t.Fatalf("incomplete adoption changed photo set: %d", got)
	}
}

func TestRemotePhotoPrefetchUsesCanonicalAssetHeldOnAnotherTarget(t *testing.T) {
	f := newDirectedShareFixture(t)
	other := saveRemoteChildRecord(t, f.app, "trails", map[string]any{"name": "Other", "author": f.remote.Id, "iri": "https://b.example/api/v1/trail/othertrail00001"})
	var body bytes.Buffer
	if err := png.Encode(&body, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(body.Bytes(), "camera.png")
	if err != nil {
		t.Fatal(err)
	}
	keeper := testAddPhoto(t, f.app, other, file, "https://b.example/api/v1/assets/canonicalphoto1")
	data := map[string]any{"expand": map[string]any{"trail_assets_via_trail": []any{map[string]any{"expand": map[string]any{"asset": map[string]any{"id": "canonicalphoto1", "type": "photo", "file": "camera.png", "collectionId": "assets"}}}}}}
	collector := remoteMediaCollector{seen: map[string]bool{}}
	collector.record(f.app, "trail", "origintrail0001", "https://b.example", f.trail.GetString("iri"), data)
	if len(collector.requests) != 0 {
		t.Fatal("prefetch downloaded canonical asset already held on another target")
	}
	if err := syncRecordPhotos(f.app, util.WithPreparedFederatedFiles(t.Context(), nil), "trail", f.trail.Id, "origintrail0001", f.remote.Id, "https://b.example", data, true); err != nil {
		t.Fatal(err)
	}
	photos := testPhotoAssets(t, f.app, f.trail)
	if len(photos) != 1 || photos[0].Id != keeper.Id {
		t.Fatal("canonical asset from another target was not linked")
	}
}
