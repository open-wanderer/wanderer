package routes

import (
	"context"
	"fmt"
	"net/http"
	"pocketbase/util"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestRemoteMediaDeadlinePreservesCompletedDownloads(t *testing.T) {
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	calls := 0
	fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
		calls++
		if calls == 2 {
			<-ctx.Done()
			return nil, func() {}, ctx.Err()
		}
		f, err := filesystem.NewFileFromBytes([]byte("first completed file"), name)
		return f, func() {}, err
	}
	requests := []remoteMediaRequest{{url: "https://remote.example/first.jpg", origin: "https://remote.example", name: "first.jpg", maxBytes: 1024}, {url: "https://remote.example/second.jpg", origin: "https://remote.example", name: "second.jpg", maxBytes: 1024}}
	ctx, cleanup, err := prepareRemoteMediaWithin(t.Context(), requests, 20*time.Millisecond)
	defer cleanup()
	if err != nil {
		t.Fatalf("media deadline discarded metadata: %v", err)
	}
	if _, err := util.PreparedRemoteSyncFile(ctx, requests[0].url, requests[0].origin, 1024); err != nil {
		t.Fatal(err)
	}
	if _, err := util.PreparedRemoteSyncFile(ctx, requests[1].url, requests[1].origin, 1024); err == nil {
		t.Fatal("unfinished file was available")
	}
}

func TestRemoteMediaContinuesPartialFieldsAfterSaving(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		for _, scenario := range []struct {
			name  string
			count int
			size  int64
		}{{"twenty", 20, 4 << 20}, {"twenty-five", 25, 4_000_000}, {"five", 5, 15 << 20}, {"many", 120, 1}} {
			t.Run(target+"/"+scenario.name, func(t *testing.T) {
				f := newRemoteSyncFixture(t)
				collection := map[string]*core.Collection{"trail": f.trails, "waypoint": f.waypoints, "summit_log": f.logs}[target]
				collection.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 200, MaxSize: 20 << 20})
				if err := f.app.Save(collection); err != nil {
					t.Fatal(err)
				}
				origin := "https://remote.example"
				iri := origin + "/api/v1/" + strings.ReplaceAll(target, "_", "-") + "/remote"
				record := f.save(collection, map[string]any{"iri": iri, "author": f.remoteActor.Id})
				photos := make([]any, scenario.count)
				for i := range photos {
					photos[i] = fmt.Sprintf("photo_%03d.jpg", i)
				}
				previous := fetchRemoteSyncMedia
				t.Cleanup(func() { fetchRemoteSyncMedia = previous })
				requests := map[string]int{}
				fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
					requests[name]++
					if scenario.size > limit {
						return nil, func() {}, util.ErrFederatedMediaBudget
					}
					file, err := filesystem.NewFileFromBytes([]byte(name), name)
					if file != nil {
						file.Size = scenario.size
					}
					return file, func() {}, err
				}
				var first []string
				for attempt := 0; attempt < 5; attempt++ {
					collector := remoteMediaCollector{seen: map[string]bool{}}
					collector.record(f.app, target, "remote", origin, iri, map[string]any{"photos": photos})
					ctx, cleanup, err := prepareRemoteMedia(t.Context(), collector.requests)
					if err != nil {
						t.Fatal(err)
					}
					syncRecordFiles(ctx, record, collection.Name, "remote", origin, map[string]any{"photos": photos})
					if err := f.app.Save(record); err != nil {
						t.Fatal(err)
					}
					cleanup()
					record, err = f.app.FindRecordById(collection, record.Id)
					if err != nil {
						t.Fatal(err)
					}
					if attempt == 0 {
						first = append([]string(nil), record.GetStringSlice("photos")...)
					}
				}
				stored := record.GetStringSlice("photos")
				if len(stored) != scenario.count {
					t.Fatalf("copied %d/%d photos", len(stored), scenario.count)
				}
				for i, name := range first {
					if stored[i] != name {
						t.Fatalf("saved file replaced: %q -> %q", name, stored[i])
					}
				}
				for _, value := range photos {
					if count := requests[value.(string)]; count > 2 {
						t.Fatalf("completed photo requested %d times", count)
					}
				}
			})
		}
	}
}

func TestRemoteMediaSyncStoresMetadataAfterUnavailableFiles(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.trails.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 20, MaxSize: 20 << 20})
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	record := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/remote", "author": f.remoteActor.Id, "needs_full_sync": true})
	previousClient, previousFetch := newRemoteSyncHTTPClient, fetchRemoteSyncMedia
	t.Cleanup(func() { newRemoteSyncHTTPClient, fetchRemoteSyncMedia = previousClient, previousFetch })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "remote", "name": "Visible metadata", "photos": []any{"one.jpg"}, "author": f.remoteActor.Id, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}}}
	}
	fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
		return nil, func() {}, context.DeadlineExceeded
	}
	result, err := performFullSync(f.app, t.Context(), nil, record)
	if err != nil || result.GetString("name") != "Visible metadata" || !result.GetBool("full_sync_completed") {
		t.Fatalf("metadata missing after media error: %v %v", result, err)
	}
}

func TestRemoteMediaNamesDistinguishLongAndSimilarSources(t *testing.T) {
	names := []string{strings.Repeat("a", 150) + "one.jpg", strings.Repeat("a", 150) + "two.jpg", "same-prefix_one.jpg", "same-prefix_two.jpg", "a snow photo.jpg", "a_snow_photo.jpg"}
	saved := []string{}
	for _, name := range names {
		if remotePhotoAlreadyStored(saved, name) {
			t.Fatalf("different source considered saved: %q", name)
		}
		saved = append(saved, remoteSyncFileName(name))
	}
	for _, name := range names {
		if !remotePhotoAlreadyStored(saved, name) {
			t.Fatalf("completed source missing: %q", name)
		}
	}
	if !remotePhotoAlreadyStored([]string{"photo_ab12cd34ef.jpg"}, "photo.jpg") {
		t.Fatal("earlier file suffix not recognized")
	}
	if remotePhotoAlreadyStored([]string{"photo_!!!!!!!!!!.jpg"}, "photo.jpg") {
		t.Fatal("invalid earlier suffix accepted")
	}
	for _, tc := range []struct {
		name, source, saved string
		want                bool
	}{
		{"plus", "photo.15+16.jpg", "photo_ab12cd34ef.15_16.jpg", true},
		{"equals", "photo.a=b.jpg", "photo_ab12cd34ef.a_b.jpg", true},
		{"asterisk", "photo.*.jpg", "photo_ab12cd34ef._.jpg", true},
		{"repeated punctuation", "photo.a+=b.jpg", "photo_ab12cd34ef.a_b.jpg", true},
		{"double dot", "photo..jpg", "photo_ab12cd34ef.jpg", true},
		{"trailing dot", "photo.jpg.", "photo_ab12cd34ef.jpg", true},
		{"trimmed and normalized", "photo.a=b.", "photo_ab12cd34ef.a_b", true},
		{"trimmed only", "photo.a=b.", "photo_ab12cd34ef.a=b", true},
		{"dots with detected extension", "photo..", "photo_ab12cd34ef.txt", true},
		{"dots with detected image extension", "photo..", "photo_ab12cd34ef.png", true},
		{"detected extension needs a dots-only source", "photo.jpg", "photo_ab12cd34ef.png", false},
		{"dots with long extension", "photo..", "photo_ab12cd34ef.abcdef", false},
		{"dots with extra component", "photo..", "photo_ab12cd34ef.jpg.png", false},
		{"dots with invalid suffix", "photo..", "photo_!!!!!!!!!!.png", false},
		{"dots with different base", "photo..", "other_ab12cd34ef.png", false},
		{"different extension", "photo.a=b.jpg", "photo_ab12cd34ef.a_c.jpg", false},
		{"invalid suffix", "photo.a=b.jpg", "photo_!!!!!!!!!!.a_b.jpg", false},
		{"different base", "photo.a=b.jpg", "other_ab12cd34ef.a_b.jpg", false},
		{"dots are not an empty extension", "photo..", "photo_ab12cd34ef", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := remotePhotoAlreadyStored([]string{tc.saved}, tc.source); got != tc.want {
				t.Fatalf("stored %q for %q: got %v, want %v", tc.saved, tc.source, got, tc.want)
			}
		})
	}
}

func TestRemoteMediaSavedNamesAvoidRepeatDownloads(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.trails.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 20, MaxSize: 20 << 20})
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	origin := "https://remote.example"
	iri := origin + "/api/v1/trail/remote"
	record := f.save(f.trails, map[string]any{"iri": iri, "author": f.remoteActor.Id})
	names := []any{strings.Repeat("a", 150) + "one.jpg", strings.Repeat("a", 150) + "two.jpg", "same-prefix_one.jpg", "same-prefix_two.jpg", "a snow photo.jpg", "a_snow_photo.jpg"}
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
		file, err := filesystem.NewFileFromBytes([]byte(name), name)
		return file, func() {}, err
	}
	collector := remoteMediaCollector{seen: map[string]bool{}}
	collector.record(f.app, "trail", "remote", origin, iri, map[string]any{"photos": names})
	ctx, cleanup, err := prepareRemoteMedia(t.Context(), collector.requests)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	syncRecordFiles(ctx, record, "trails", "remote", origin, map[string]any{"photos": names})
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	record, err = f.app.FindRecordById("trails", record.Id)
	if err != nil {
		t.Fatal(err)
	}
	saved := record.GetStringSlice("photos")
	if len(saved) != len(names) {
		t.Fatalf("saved %d/%d distinct photos", len(saved), len(names))
	}
	for _, name := range names {
		if !remotePhotoAlreadyStored(saved, name.(string)) {
			t.Fatalf("stored source not recognized: %q", name)
		}
	}
	next := remoteMediaCollector{seen: map[string]bool{}}
	next.record(f.app, "trail", "remote", origin, iri, map[string]any{"photos": names})
	if len(next.requests) != 0 {
		t.Fatalf("saved files requested again: %v", next.requests)
	}
}
