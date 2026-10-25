package routes

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteTopLevelSyncAcceptsOnlyExactMetadataFields(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		for _, existing := range []bool{false, true} {
			state := "new"
			if existing {
				state = "existing"
			}
			for _, key := range []string{"author+", "+author", "author-", "trails+", "name+", "unknown+"} {
				t.Run(kind+"/"+state+"/"+key, func(t *testing.T) {
					f, record := topLevelAuthorFixture(t, kind, existing)
					localTrail := f.save(f.trails, map[string]any{"author": f.localActor.Id, "name": "Local trail"})
					payload := topLevelAuthorPayload(record, f.remoteActor.GetString("iri"))
					value := any(f.localActor.Id)
					if key == "author-" {
						value = f.remoteActor.Id
					} else if key == "trails+" {
						value = localTrail.Id
					} else if key == "name+" {
						value = " appended"
					}
					payload[key] = value
					serveTopLevelAuthorPayload(t, payload)
					if _, err := syncTopLevelAuthorFixture(t, f, kind, record); err != nil {
						t.Fatal(err)
					}
					stored, err := f.app.FindRecordById(record.Collection(), record.Id)
					if err != nil {
						t.Fatal(err)
					}
					if stored.GetString("author") != f.remoteActor.Id || stored.GetString("name") != "Canonical name" || slices.Contains(stored.GetStringSlice("trails"), localTrail.Id) {
						t.Fatalf("metadata changed bound fields: author=%s name=%q trails=%v", stored.GetString("author"), stored.GetString("name"), stored.GetStringSlice("trails"))
					}
				})
			}
		}
	}
}

func TestRemoteChildSyncAcceptsOnlyExactMetadataFields(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	trail := f.save(f.trails, map[string]any{"author": f.remoteActor.Id, "public": true})
	for _, kind := range []string{"waypoint", "summit-log"} {
		t.Run(kind, func(t *testing.T) {
			data := map[string]any{"id": "entry", "name": "Name", "text": "Text", "name+": " appended", "text+": " appended", "author+": f.localActor.Id, "trail+": "other", "id:autogenerate": true}
			collection := f.waypoints
			var err error
			if kind == "waypoint" {
				err = syncWaypoints(f.app, t.Context(), trail, "https://remote.example", []any{data})
			} else {
				collection = f.logs
				err = syncSummitLogs(f.app, t.Context(), trail, "https://remote.example", []any{data})
			}
			if err != nil {
				t.Fatal(err)
			}
			record, err := f.app.FindFirstRecordByData(collection, "iri", "https://remote.example/api/v1/"+kind+"/entry")
			if err != nil {
				t.Fatal(err)
			}
			if record.GetString("author") != f.remoteActor.Id || record.GetString("trail") != trail.Id || record.GetString("name") == "Name appended" || record.GetString("text") == "Text appended" {
				t.Fatalf("child metadata changed bound fields: %v", record.PublicExport())
			}
		})
	}
}

func TestRemoteRecordFieldsKeepFilesAndRelations(t *testing.T) {
	f := newRemoteSyncFixture(t)
	f.trails.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 10})
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(f.trails)
	record.Set("photos", []string{"existing.jpg"})
	record.Set("author", f.remoteActor.Id)
	loadRemoteRecordFields(record, map[string]any{"name": "Updated", "photos": []string{"remote.jpg"}, "photos+": []string{"extra.jpg"}, "author+": f.localActor.Id, "needs_full_sync": true, "unknown": "value"})
	if record.GetString("name") != "Updated" || record.GetString("author") != f.remoteActor.Id || !slices.Equal(record.GetStringSlice("photos"), []string{"existing.jpg"}) || record.GetBool("needs_full_sync") {
		t.Fatalf("remote fields replaced sync values: %v", record.PublicExport())
	}
}
