package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/services/trailmerge"
	"pocketbase/util"
)

func TestRepeatedTrailMergesLinkRealPhotosToEntriesAndKeepTargetGallery(t *testing.T) {
	for _, deleteSource := range []bool{false, true} {
		t.Run(fmt.Sprintf("delete=%t", deleteSource), func(t *testing.T) {
			f := newRecordHooksFixture(t)
			owner := f.localActor("owner")
			target := f.save("trails", map[string]any{"name": "Target", "author": owner.Id})
			shared := f.save("trails", map[string]any{"name": "Shared preview", "author": owner.Id})
			raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
			if err != nil {
				t.Fatal(err)
			}
			savePhoto := func(generated bool) *core.Record {
				name := "route.png"
				metadata := map[string]any{}
				if generated {
					name = "wanderer-route-preview.png"
					metadata["generated"] = map[string]any{"kind": "route-preview"}
				}
				file, err := filesystem.NewFileFromBytes(raw, name)
				if err != nil {
					t.Fatal(err)
				}
				return f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": owner.Id, "file": file, "metadata": metadata})
			}
			targetPhoto := savePhoto(false)
			f.save("trail_assets", map[string]any{"trail": target.Id, "asset": targetPhoto.Id, "is_thumbnail": true})

			f.bindHooks()
			search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
			}))
			defer search.Close()
			selected := targetPhoto.Id
			for i := range 4 {
				source := f.save("trails", map[string]any{"name": fmt.Sprintf("Source%d", i), "description": fmt.Sprintf("Source%d", i), "author": owner.Id, "date": "2026-01-01 00:00:00.000Z"})
				preview, real := savePhoto(true), savePhoto(false)
				f.save("trail_assets", map[string]any{"trail": source.Id, "asset": preview.Id})
				if i == 0 {
					f.save("trail_assets", map[string]any{"trail": shared.Id, "asset": preview.Id})
				}
				f.save("trail_assets", map[string]any{"trail": source.Id, "asset": real.Id})
				if err := trailmerge.Merge(f.app, meilisearch.New(search.URL), t.Context(), owner, source.Id, target.Id, trailmerge.MergeSettings{Photos: true, Delete: deleteSource}, func(*core.Record) {}); err != nil {
					t.Fatal(err)
				}
				links, err := f.app.FindRecordsByFilter("trail_assets", "trail={:trail} && is_thumbnail=true", "", -1, 0, map[string]any{"trail": target.Id})
				if err != nil || len(links) != 1 || links[0].GetString("asset") != selected {
					t.Fatalf("selected real thumbnail changed: %v %v", links, err)
				}
				photos, err := util.PhotoAssetsForTarget(f.app, "trail", target.Id, -1)
				if err != nil {
					t.Fatal(err)
				}
				if len(photos) != 1 || photos[0].Id != targetPhoto.Id {
					t.Fatalf("merge changed target gallery: %v", photos)
				}
				entries, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail} && text={:text}", "", -1, 0, map[string]any{"trail": target.Id, "text": fmt.Sprintf("Source%d", i)})
				if err != nil || len(entries) != 1 {
					t.Fatalf("merged entries=%d want=1: %v", len(entries), err)
				}
				entryPhotos, err := util.PhotoAssetsForTarget(f.app, "summit_log", entries[0].Id, -1)
				if err != nil || len(entryPhotos) != 1 || entryPhotos[0].Id != real.Id {
					t.Fatalf("new entry must link only the source real photo: %v %v", entryPhotos, err)
				}
				// Cascades retain shared files and remove only exclusive source previews.
				if f.exists(preview) != (i == 0) {
					t.Fatalf("source preview cleanup delete=%t shared=%t exists=%t", deleteSource, i == 0, f.exists(preview))
				}
				fs, err := f.app.NewFilesystem()
				if err != nil {
					t.Fatal(err)
				}
				expectedPhoto := real
				if i == 0 {
					expectedPhoto = preview
				}
				reader, err := fs.GetReader(expectedPhoto.BaseFilesPath() + "/" + expectedPhoto.GetString("file"))
				if err != nil {
					fs.Close()
					t.Fatal(err)
				}
				var content bytes.Buffer
				_, err = content.ReadFrom(reader)
				reader.Close()
				fs.Close()
				if err != nil || !bytes.Equal(content.Bytes(), raw) {
					t.Fatal("merge lost stored photo bytes")
				}
			}

		})
	}
}

func TestTrailMergeLinksOnlySourceRealPhotosToEntryAndKeepsCover(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("owner")
	source := f.save("trails", map[string]any{"name": "Source", "description": "Source outing", "author": owner.Id, "date": "2026-01-01 00:00:00.000Z"})
	target := f.save("trails", map[string]any{"name": "Target", "author": owner.Id})
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	photo := func(name string, generated bool) *core.Record {
		file, err := filesystem.NewFileFromBytes(raw, name)
		if err != nil {
			t.Fatal(err)
		}
		metadata := map[string]any{}
		if generated {
			metadata["generated"] = map[string]any{"kind": "route-preview"}
		}
		return f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": file, "metadata": metadata})
	}
	real, preview, cover := photo("real.png", false), photo("preview.png", true), photo("cover.png", false)
	f.save("trail_assets", map[string]any{"trail": source.Id, "asset": real.Id})
	f.save("trail_assets", map[string]any{"trail": source.Id, "asset": preview.Id})
	f.save("trail_assets", map[string]any{"trail": target.Id, "asset": cover.Id, "is_thumbnail": true})
	before, err := util.PhotoAssetsForTarget(f.app, "trail", source.Id, -1)
	if err != nil || len(before) != 2 {
		t.Fatalf("source fixture must contain real photo and preview: %v %v", before, err)
	}
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	defer search.Close()
	if err := trailmerge.Merge(f.app, meilisearch.New(search.URL), t.Context(), owner, source.Id, target.Id, trailmerge.MergeSettings{Photos: true}, func(*core.Record) {}); err != nil {
		t.Fatal(err)
	}
	entries, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail}", "", -1, 0, map[string]any{"trail": target.Id})
	if err != nil || len(entries) != 1 {
		t.Fatalf("created entries=%v error=%v", entries, err)
	}
	linked, err := util.PhotoAssetsForTarget(f.app, "summit_log", entries[0].Id, -1)
	if err != nil || len(linked) != 1 || linked[0].Id != real.Id {
		t.Fatalf("entry photos must contain only the source real photo: %v %v", linked, err)
	}
	gallery, err := f.app.FindRecordsByFilter("trail_assets", "trail={:trail}", "", -1, 0, map[string]any{"trail": target.Id})
	if err != nil || len(gallery) != 1 || gallery[0].GetString("asset") != cover.Id || !gallery[0].GetBool("is_thumbnail") {
		t.Fatalf("target gallery or cover changed: %v %v", gallery, err)
	}
}
