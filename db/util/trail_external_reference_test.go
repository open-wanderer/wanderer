package util

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestReassignTrailExternalReferencesMarksMovedReferences(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "name"})
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	refs := core.NewBaseCollection("trail_external_reference")
	refs.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1, Required: true},
		&core.TextField{Name: "user"},
		&core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"},
		&core.TextField{Name: "track_source"},
	)
	if err := app.Save(refs); err != nil {
		t.Fatal(err)
	}

	source := core.NewRecord(trails)
	source.Set("name", "source")
	target := core.NewRecord(trails)
	target.Set("name", "target")
	for _, trail := range []*core.Record{source, target} {
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
	}
	ref := core.NewRecord(refs)
	ref.Load(map[string]any{"trail": source.Id, "user": "user-1", "provider": "hammerhead", "external_id": "act-1"})
	if err := app.Save(ref); err != nil {
		t.Fatal(err)
	}

	if err := ReassignTrailExternalReferences(app, source.Id, target.Id); err != nil {
		t.Fatal(err)
	}
	moved, err := app.FindRecordById("trail_external_reference", ref.Id)
	if err != nil {
		t.Fatal(err)
	}
	if moved.GetString("trail") != target.Id || moved.GetString("track_source") != TrackSourceMoved {
		t.Fatalf("expected the reference on the target marked as moved, got trail=%q source=%q", moved.GetString("trail"), moved.GetString("track_source"))
	}
}

func TestClassifyExternalReferencesBeforeTracking(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "name"}, &core.AutodateField{Name: "created", OnCreate: true})
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	refs := core.NewBaseCollection("trail_external_reference")
	refs.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1, Required: true},
		&core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"},
		&core.TextField{Name: "track_source"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	if err := app.Save(refs); err != nil {
		t.Fatal(err)
	}
	backdate := func(table string, id string, created time.Time) {
		stamp, _ := types.ParseDateTime(created)
		if _, err := app.DB().NewQuery("UPDATE " + table + " SET created = {:created} WHERE id = {:id}").
			Bind(dbx.Params{"created": stamp.String(), "id": id}).Execute(); err != nil {
			t.Fatal(err)
		}
	}
	create := func(name string, trailCreated time.Time, refCreated time.Time, source string) *core.Record {
		trail := core.NewRecord(trails)
		trail.Set("name", name)
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
		ref := core.NewRecord(refs)
		ref.Load(map[string]any{"trail": trail.Id, "provider": "hammerhead", "external_id": name, "track_source": source})
		if err := app.Save(ref); err != nil {
			t.Fatal(err)
		}
		backdate("trails", trail.Id, trailCreated)
		backdate("trail_external_reference", ref.Id, refCreated)
		return ref
	}
	sourceOf := func(ref *core.Record) string {
		record, err := app.FindRecordById("trail_external_reference", ref.Id)
		if err != nil {
			t.Fatal(err)
		}
		return record.GetString("track_source")
	}

	now := time.Now()
	backfill := now.Add(-30 * 24 * time.Hour)
	imported := create("imported", now, now.Add(2*time.Second), "")
	mergedIn := create("merged-in", now.Add(-48*time.Hour), now, "")
	mergedInOlder := create("merged-in-older", now, now.Add(-48*time.Hour), "")
	backfilled := create("backfilled", now.Add(-300*24*time.Hour), backfill.Add(-time.Minute), "")
	alreadyMarked := create("marked", now.Add(-48*time.Hour), now, TrackSourceMoved)

	moved, legacy, err := ClassifyExternalReferencesBeforeTracking(app, backfill)
	if err != nil {
		t.Fatal(err)
	}
	if moved != 2 || legacy != 2 {
		t.Fatalf("expected two moved and two legacy references, got %d and %d", moved, legacy)
	}
	for name, ref := range map[string]*core.Record{"imported": imported, "backfilled": backfilled} {
		if sourceOf(ref) != TrackSourceLegacy {
			t.Fatalf("%s reference must stay usable but be marked legacy, got %q", name, sourceOf(ref))
		}
	}
	for name, ref := range map[string]*core.Record{"merged-in": mergedIn, "merged-in-older": mergedInOlder, "marked": alreadyMarked} {
		if sourceOf(ref) != TrackSourceMoved {
			t.Fatalf("%s reference must be marked as moved, got %q", name, sourceOf(ref))
		}
	}

	// A second run finds nothing left to classify.
	if moved, legacy, err = ClassifyExternalReferencesBeforeTracking(app, backfill); err != nil || moved != 0 || legacy != 0 {
		t.Fatalf("expected an idempotent classification, got %d/%d (%v)", moved, legacy, err)
	}
}
