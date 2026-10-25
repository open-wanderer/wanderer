package routes

import (
	"context"
	"pocketbase/util"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestRemoteMediaHonorsRaisedAndLoweredFieldLimits(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, target := range []string{"trail", "waypoint", "summit_log", "list"} {
		for _, size := range []int64{1024, 25 << 20} {
			t.Run(target+"/"+map[bool]string{true: "raised", false: "lowered"}[size > util.RemotePhotoMaxBytes], func(t *testing.T) {
				f := newRemoteSyncFixture(t)
				col := map[string]*core.Collection{"trail": f.trails, "waypoint": f.waypoints, "summit_log": f.logs, "list": f.lists}[target]
				field := "gpx"
				if target == "list" {
					field = "avatar"
				}
				col.Fields.Add(&core.FileField{Name: field, MaxSelect: 1, MaxSize: size}, &core.FileField{Name: "photos", MaxSelect: 10, MaxSize: size})
				if err := f.app.Save(col); err != nil {
					t.Fatal(err)
				}
				origin := "https://remote.example"
				iri := origin + "/api/v1/" + target + "/remote"
				record := f.save(col, map[string]any{"iri": iri, "author": f.remoteActor.Id})
				data := map[string]any{field: "route.gpx"}
				if target != "list" {
					data["photos"] = []any{"photo.jpg"}
				}
				collector := remoteMediaCollector{seen: map[string]bool{}}
				collector.record(f.app, target, "remote", origin, iri, data)
				want := 2
				if target == "list" {
					want = 1
				}
				if len(collector.requests) != want {
					t.Fatalf("requests=%d want%d", len(collector.requests), want)
				}
				previous := fetchRemoteSyncMedia
				t.Cleanup(func() { fetchRemoteSyncMedia = previous })
				fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
					if limit != size {
						t.Fatalf("configured maxSize=%d, download cap=%d", size, limit)
					}
					file, err := filesystem.NewFileFromBytes([]byte(name), name)
					if file != nil {
						file.Size = size
					}
					return file, func() {}, err
				}
				ctx, cleanup, err := prepareRemoteMedia(t.Context(), collector.requests)
				defer cleanup()
				if err != nil {
					t.Fatal(err)
				}
				if target == "list" {
					syncListRecordFiles(ctx, record, col.Name, "remote", origin, data)
				} else {
					syncRecordFiles(ctx, record, col.Name, "remote", origin, data)
				}
				if err := f.app.Save(record); err != nil {
					t.Fatal(err)
				}
				record, err = f.app.FindRecordById(col, record.Id)
				if err != nil || record.GetString(field) == "" {
					t.Fatalf("configured file was not saved: %v", err)
				}
				if target != "list" && len(record.GetStringSlice("photos")) != 1 {
					t.Fatal("configured photo was not saved")
				}
			})
		}
	}
}
