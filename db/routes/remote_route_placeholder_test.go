package routes

import (
	"context"
	"testing"

	"pocketbase/services/routeimage"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const (
	placeholderTestOrigin = "https://remote.example"
	placeholderTestIRI    = placeholderTestOrigin + "/api/v1/trail/remote"
	originPlaceholder     = "wanderer_route_placeholder_abc1234567.svg"
)

func newRoutePlaceholderSyncFixture(t *testing.T, maxPhotos int) *remoteSyncFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.trails.Fields.Add(&core.FileField{Name: "photos", MaxSelect: maxPhotos, MaxSize: 20 << 20})
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
		file, err := filesystem.NewFileFromBytes([]byte(name), name)
		return file, func() {}, err
	}
	return f
}

func TestRemoteMediaCollectorSkipsOriginPlaceholder(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t, 20)
	collector := remoteMediaCollector{seen: map[string]bool{}}
	collector.record(f.app, "trail", "remote", placeholderTestOrigin, placeholderTestIRI, map[string]any{
		"photos": []any{originPlaceholder, "summit.jpg"},
	})
	if len(collector.requests) != 1 || collector.requests[0].name != "summit.jpg" {
		t.Fatalf("requests=%v", collector.requests)
	}
}

func TestRemoteMediaCollectorPlaceholderLeavesPhotoLimit(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t, 1)
	collector := remoteMediaCollector{seen: map[string]bool{}}
	collector.record(f.app, "trail", "remote", placeholderTestOrigin, placeholderTestIRI, map[string]any{
		"photos": []any{originPlaceholder, "summit.jpg"},
	})
	if len(collector.requests) != 1 || collector.requests[0].name != "summit.jpg" {
		t.Fatalf("requests=%v", collector.requests)
	}
}

// The placeholder is prepared here on purpose: syncRecordFiles has to skip it
// even when the prefetch handed it a downloaded copy.
func TestSyncRecordFilesSkipsOriginPlaceholder(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		maxPhotos int
	}{{"room for both", 20}, {"room for one", 1}} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newRoutePlaceholderSyncFixture(t, scenario.maxPhotos)
			record := f.save(f.trails, map[string]any{"iri": placeholderTestIRI, "author": f.remoteActor.Id})
			requests := make([]remoteMediaRequest, 0, 2)
			for _, name := range []string{originPlaceholder, "summit.jpg"} {
				requests = append(requests, remoteMediaRequest{
					url:      placeholderTestOrigin + "/api/v1/files/trails/remote/" + name,
					origin:   placeholderTestOrigin,
					name:     name,
					maxBytes: 1 << 20,
				})
			}
			ctx, cleanup, err := prepareRemoteMedia(t.Context(), requests)
			defer cleanup()
			if err != nil {
				t.Fatal(err)
			}

			syncRecordFiles(ctx, record, "trails", "remote", placeholderTestOrigin, map[string]any{
				"photos": []any{originPlaceholder, "summit.jpg"},
			})
			if err := f.app.Save(record); err != nil {
				t.Fatal(err)
			}

			record, err = f.app.FindRecordById("trails", record.Id)
			if err != nil {
				t.Fatal(err)
			}
			saved := record.GetStringSlice("photos")
			if len(saved) != 1 || !remotePhotoAlreadyStored(saved, "summit.jpg") {
				t.Fatalf("photos=%v", saved)
			}
		})
	}
}

// A mirror renders its own placeholder, so a sync must leave it the only
// photo: anything added next to it makes routeimage drop it.
func TestSyncKeepsLocalPlaceholder(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t, 20)
	local, err := filesystem.NewFileFromBytes([]byte("<svg/>"), "wanderer_route_placeholder.svg")
	if err != nil {
		t.Fatal(err)
	}
	record := f.save(f.trails, map[string]any{"iri": placeholderTestIRI, "author": f.remoteActor.Id, "photos": []any{local}})
	before := record.GetStringSlice("photos")
	if len(before) != 1 || !routeimage.IsPlaceholder(before[0]) {
		t.Fatalf("fixture photos=%v", before)
	}

	// Each route edit on the origin gives its placeholder a new name.
	for _, name := range []string{originPlaceholder, "wanderer_route_placeholder_def7654321.svg"} {
		data := map[string]any{"photos": []any{name}}
		collector := remoteMediaCollector{seen: map[string]bool{}}
		collector.record(f.app, "trail", "remote", placeholderTestOrigin, placeholderTestIRI, data)
		ctx, cleanup, err := prepareRemoteMedia(t.Context(), collector.requests)
		if err != nil {
			cleanup()
			t.Fatal(err)
		}
		syncRecordFiles(ctx, record, "trails", "remote", placeholderTestOrigin, data)
		if err := f.app.Save(record); err != nil {
			cleanup()
			t.Fatal(err)
		}
		cleanup()
		record, err = f.app.FindRecordById("trails", record.Id)
		if err != nil {
			t.Fatal(err)
		}
	}

	after := record.GetStringSlice("photos")
	if len(after) != 1 || after[0] != before[0] {
		t.Fatalf("photos=%v, want %v", after, before)
	}
}
