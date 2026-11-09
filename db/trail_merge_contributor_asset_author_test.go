package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"pocketbase/services/trailmerge"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestMergedContributorLogPhotosRetainContributorOwnershipAndAccountCascade(t *testing.T) {
	for _, edited := range []bool{false, true} {
		t.Run(fmt.Sprintf("editor-photo=%t", edited), func(t *testing.T) { testMergedLogPhotoOwner(t, edited) })
	}
}

func testMergedLogPhotoOwner(t *testing.T, edited bool) {
	f := newRecordHooksFixture(t)
	merger, contributor := f.localActor("merger"), f.localActor("contributor")
	photoAuthor := contributor
	if edited {
		photoAuthor = f.localActor("editor")
	}

	source := f.save("trails", map[string]any{"name": "Source", "author": merger.Id, "date": "2026-01-01 00:00:00.000Z"})
	target := f.save("trails", map[string]any{"name": "Target", "author": merger.Id, "public": true})
	entry := f.save("summit_logs", map[string]any{"trail": source.Id, "author": contributor.Id, "text": "Contributor entry", "date": "2026-02-01 00:00:00.000Z"})
	var photoBytes bytes.Buffer
	if err := png.Encode(&photoBytes, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(photoBytes.Bytes(), "contributor.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := f.save("assets", map[string]any{"author": photoAuthor.Id, "type": "photo", "storage_mode": "copy", "file": file})
	f.save("trail_assets", map[string]any{"trail": source.Id, "asset": asset.Id})
	f.save("summit_log_assets", map[string]any{"summit_log": entry.Id, "asset": asset.Id})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	defer server.Close()
	if err := trailmerge.Merge(f.app, meilisearch.New(server.URL), context.Background(), merger, source.Id, target.Id, trailmerge.MergeSettings{SummitLog: true, Photos: true}, func(*core.Record) {}); err != nil {
		t.Fatal(err)
	}
	logs, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail} && author={:author}", "", -1, 0, map[string]any{"trail": target.Id, "author": contributor.Id})
	if err != nil || len(logs) != 1 {
		t.Fatalf("contributor log missing: %v %v", logs, err)
	}
	links, err := f.app.FindRecordsByFilter("summit_log_assets", "summit_log={:log}", "", -1, 0, map[string]any{"log": logs[0].Id})
	if err != nil || len(links) != 1 {
		t.Fatalf("copied log photo missing: %v %v", links, err)
	}
	copy, err := f.app.FindRecordById("assets", links[0].GetString("asset"))
	if err != nil || copy.GetString("author") != photoAuthor.Id || copy.Id == asset.Id {
		t.Fatalf("contributor lost ownership of independent log photo copy: %v %v", copy, err)
	}
	primary, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail} && author={:author}", "", -1, 0, map[string]any{"trail": target.Id, "author": merger.Id})
	if err != nil || len(primary) != 1 {
		t.Fatalf("source trail entry missing: %v %v", primary, err)
	}
	primaryLinks, err := f.app.FindRecordsByFilter("summit_log_assets", "summit_log={:log}", "", -1, 0, map[string]any{"log": primary[0].Id})
	if err != nil || len(primaryLinks) != 1 || primaryLinks[0].GetString("asset") != copy.Id {
		t.Fatalf("source entry and contributor entry lost their shared owned photo copy: %v %v", primaryLinks, err)
	}
	trailLinks, err := f.app.FindRecordsByFilter("trail_assets", "trail={:trail}", "", -1, 0, map[string]any{"trail": target.Id})
	if err != nil || len(trailLinks) != 0 {
		t.Fatalf("merge changed target gallery: %v %v", trailLinks, err)
	}
	user, err := f.app.FindRecordById("users", photoAuthor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	f.bindHooks()
	if err := f.app.Delete(user); err != nil {
		t.Fatal(err)
	}
	for _, photo := range []*core.Record{asset, copy} {
		stored, err := f.app.FindRecordById("assets", photo.Id)
		if err != nil || stored.GetString("author") == photoAuthor.Id {
			t.Fatalf("photo linked to a surviving trail was not handed over: %v %v", stored, err)
		}
	}
}
