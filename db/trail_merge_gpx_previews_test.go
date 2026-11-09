package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/services/trailmerge"
	"pocketbase/util"
)

func TestTrailMergeGPXEntriesKeepOnePreviewAndRealPhotos(t *testing.T) {
	for _, mode := range []string{"manual_keep_source", "manual_delete_source", "auto_defaults"} {
		for _, contributor := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/contributor=%v", mode, contributor), func(t *testing.T) {
				f := newRecordHooksFixture(t)
				owner := f.localActor("owner")
				author := owner
				if contributor {
					author = f.localActor("contributor")
				}
				gpx := func() *filesystem.File {
					t.Helper()
					file, err := filesystem.NewFileFromBytes([]byte(placeholderGPX), "route.gpx")
					if err != nil {
						t.Fatal(err)
					}
					return file
				}
				source := f.save("trails", map[string]any{"name": "Ridge", "author": owner.Id, "date": "2026-01-01 00:00:00.000Z", "gpx": gpx()})
				sourceText, sourceDate := source.GetString("description"), source.GetDateTime("date").Time()
				if sourceText != "" || sourceDate.IsZero() {
					t.Fatalf("source fixture must have an empty description and a date: description=%q date=%v", sourceText, sourceDate)
				}
				target := f.save("trails", map[string]any{"name": "Ridge", "author": owner.Id, "gpx": gpx()})
				f.bindHooks()
				previewLog := f.save("summit_logs", map[string]any{"trail": source.Id, "author": author.Id, "text": "Route preview entry", "date": "2026-02-01 00:00:00.000Z", "gpx": gpx()})
				sourcePreview := onlyGeneratedPlaceholder(t, f.app, previewLog)
				realLog := f.save("summit_logs", map[string]any{"trail": source.Id, "author": author.Id, "text": "Photo entry", "date": "2026-03-01 00:00:00.000Z", "gpx": gpx()})
				raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
				if err != nil {
					t.Fatal(err)
				}
				realIDs := map[string]bool{}
				sourceIDs := map[string]bool{}
				for i := range 4 {
					file, err := filesystem.NewFileFromBytes(raw, fmt.Sprintf("summit%d.png", i))
					if err != nil {
						t.Fatal(err)
					}
					real := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": author.Id, "file": file, "lat": 46.6, "lon": 7.2})
					realIDs[real.Id] = true
					f.save("summit_log_assets", map[string]any{"summit_log": realLog.Id, "asset": real.Id})
					primary := f.photo(owner)
					sourceIDs[primary.Id] = true
					f.save("trail_assets", map[string]any{"trail": source.Id, "asset": primary.Id})
				}
				search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
				}))
				t.Cleanup(search.Close)
				deleteSource := mode != "manual_keep_source"
				if mode == "auto_defaults" {
					err = trailmerge.TryAutoMergeImportedTrail(f.app, meilisearch.New(search.URL), t.Context(), owner, source.Id,
						trailmerge.PluginAutoMergeSettings{Enabled: true}, func(*core.Record) {})
				} else {
					err = trailmerge.Merge(f.app, meilisearch.New(search.URL), t.Context(), owner, source.Id, target.Id,
						trailmerge.MergeSettings{Photos: true, SummitLog: true, Delete: deleteSource}, func(*core.Record) {})
				}
				if err != nil {
					t.Fatal(err)
				}
				mergedLog := func(text string) *core.Record {
					t.Helper()
					logs, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail} && text={:text}", "", -1, 0, dbx.Params{"trail": target.Id, "text": text})
					if err != nil || len(logs) != 1 {
						t.Fatalf("merged entry %q: count=%d error=%v", text, len(logs), err)
					}
					if logs[0].GetString("author") != author.Id || logs[0].GetString("gpx") == "" {
						t.Fatal("merged entry lost its author or GPX")
					}
					return logs[0]
				}
				mergedPreviewLog := mergedLog("Route preview entry")
				mergedPreview := onlyGeneratedPlaceholder(t, f.app, mergedPreviewLog)
				if mergedPreview.Id == sourcePreview.Id {
					t.Fatal("merged entry reused the source's generated preview")
				}
				// PocketBase v0.38.0 turns an empty string filter parameter into literal quotes.
				targetLogs, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail}", "", -1, 0, dbx.Params{"trail": target.Id})
				if err != nil {
					t.Fatalf("load target summit logs: %v", err)
				}
				var primaryLogs []*core.Record
				for _, log := range targetLogs {
					if log.GetString("text") == sourceText && log.GetDateTime("date").Time().Equal(sourceDate) {
						primaryLogs = append(primaryLogs, log)
					}
				}
				if len(primaryLogs) != 1 {
					t.Fatalf("primary merge entry for description %q on %v: got %d matches among %d target entries, want 1", sourceText, sourceDate, len(primaryLogs), len(targetLogs))
				}
				primary := primaryLogs[0]
				if primary.GetString("author") != owner.Id || primary.GetString("gpx") == "" {
					t.Fatal("primary merge entry lost its owner or GPX")
				}
				mergedPhotoLog := mergedLog("Photo entry")
				if primary.Id == mergedPreviewLog.Id || primary.Id == mergedPhotoLog.Id {
					t.Fatal("primary merge entry must be distinct from both copied entries")
				}
				primaryPhotos, err := util.PhotoAssetsForTarget(f.app, "summit_log", primary.Id, -1)
				if err != nil || len(primaryPhotos) != len(sourceIDs) {
					t.Fatalf("primary merge photos: count=%d error=%v", len(primaryPhotos), err)
				}
				for _, photo := range primaryPhotos {
					if util.IsGeneratedRoutePreviewAsset(photo) || !sourceIDs[photo.Id] {
						t.Fatal("primary merge batch retained a generated image or changed a real photo")
					}
				}
				photos, err := util.PhotoAssetsForTarget(f.app, "summit_log", mergedPhotoLog.Id, -1)
				if err != nil || len(photos) != len(realIDs) {
					t.Fatalf("merged real photos: count=%d error=%v", len(photos), err)
				}
				for _, photo := range photos {
					if util.IsGeneratedRoutePreviewAsset(photo) || photo.GetString("author") != author.Id || photo.GetFloat("lat") != 46.6 || photo.GetFloat("lon") != 7.2 {
						t.Fatal("merged real photo changed its owner/coordinates or retained a generated image")
					}
					if !contributor && !realIDs[photo.Id] {
						t.Fatal("owner's real photo was copied instead of linked")
					}
				}
				fsys, err := f.app.NewFilesystem()
				if err != nil {
					t.Fatal(err)
				}
				defer fsys.Close()
				reader, err := fsys.GetReader(photos[0].BaseFilesPath() + "/" + photos[0].GetString("file"))
				if err != nil {
					t.Fatal(err)
				}
				stored, readErr := io.ReadAll(reader)
				reader.Close()
				if readErr != nil || !bytes.Equal(stored, raw) {
					t.Fatal("merge changed or lost real photo bytes")
				}
				if deleteSource {
					if f.exists(source) || f.exists(previewLog) || f.exists(sourcePreview) {
						t.Fatal("deleted source left its exclusive route preview behind")
					}
				} else {
					if !f.exists(source) || !f.exists(previewLog) || !f.exists(sourcePreview) {
						t.Fatal("merge removed the retained source entry or its preview")
					}
					if kept := onlyGeneratedPlaceholder(t, f.app, previewLog); kept.Id != sourcePreview.Id {
						t.Fatal("retained source entry's preview changed")
					}
				}
			})
		}
	}
}
