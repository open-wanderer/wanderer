package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"pocketbase/services/trailmerge"
	"pocketbase/util"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestTrailMergeKeepsGeneratedSummitLogPhotoAndItsStoredBytes(t *testing.T) {
	for _, deleteSource := range []bool{false, true} {
		for _, contributor := range []bool{false, true} {
			t.Run(fmt.Sprintf("delete=%v/contributor=%v", deleteSource, contributor), func(t *testing.T) {
				f := newRecordHooksFixture(t)
				owner := f.localActor("owner")
				author := owner
				if contributor {
					author = f.localActor("contributor")
				}
				source := f.save("trails", map[string]any{"name": "Source", "author": owner.Id, "date": "2026-01-01 00:00:00.000Z"})
				target := f.save("trails", map[string]any{"name": "Target", "author": owner.Id})
				log := f.save("summit_logs", map[string]any{"trail": source.Id, "author": author.Id, "text": "Independent summit", "date": "2026-02-01 00:00:00.000Z"})
				png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
				if err != nil {
					t.Fatal(err)
				}
				file, err := filesystem.NewFileFromBytes(png, "wanderer-route-preview.png")
				if err != nil {
					t.Fatal(err)
				}
				preview := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": author.Id, "file": file, "metadata": map[string]any{"generated": map[string]any{"kind": "route-preview"}}})
				f.save("summit_log_assets", map[string]any{"summit_log": log.Id, "asset": preview.Id})
				f.bindHooks()
				search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
				}))
				defer search.Close()
				if err := trailmerge.Merge(f.app, meilisearch.New(search.URL), t.Context(), owner, source.Id, target.Id, trailmerge.MergeSettings{Photos: true, SummitLog: true, Delete: deleteSource}, func(*core.Record) {}); err != nil {
					t.Fatal(err)
				}
				copied, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail} && text={:text}", "", -1, 0, dbx.Params{"trail": target.Id, "text": "Independent summit"})
				if err != nil || len(copied) != 1 {
					t.Fatalf("summit log copy missing: count=%d err=%v", len(copied), err)
				}
				photos, err := util.PhotoAssetsForTarget(f.app, "summit_log", copied[0].Id, -1)
				if err != nil || len(photos) != 1 || !util.IsGeneratedRoutePreviewAsset(photos[0]) {
					t.Fatalf("copied summit log lost generated photo: count=%d err=%v", len(photos), err)
				}
				if photos[0].GetString("author") != author.Id {
					t.Fatal("generated photo changed its author")
				}
				fs, err := f.app.NewFilesystem()
				if err != nil {
					t.Fatal(err)
				}
				defer fs.Close()
				reader, err := fs.GetReader(photos[0].BaseFilesPath() + "/" + photos[0].GetString("file"))
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				stored, err := io.ReadAll(reader)
				if err != nil || !bytes.Equal(stored, png) {
					t.Fatal("generated summit log bytes changed or disappeared")
				}
			})
		}
	}
}
