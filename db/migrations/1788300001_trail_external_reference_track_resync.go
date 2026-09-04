package migrations

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"pocketbase/util"
)

// Fields supporting the on-demand track resync of a single imported trail:
//   - kind: whether the reference came from a planned route or a completed
//     activity, so the resync can call the matching detail capability;
//   - track_source: "moved" for references a merge moved onto another
//     trail; those never describe that trail's track and cannot be resynced.
//     Merges before this marker existed are recognised from the records'
//     timestamps, the remaining older references become "legacy" (see
//     util.ClassifyExternalReferencesBeforeTracking);
//   - track_resynced_at: when the track was last fetched again.
//
// Idempotent, and it removes track_resync_requested_at where a development
// database still carries it from an earlier draft.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("track_resync_requested_at")
		if collection.Fields.GetByName("kind") == nil {
			collection.Fields.Add(&core.TextField{Name: "kind", Max: 16})
		}
		addedTrackSource := false
		if collection.Fields.GetByName("track_source") == nil {
			collection.Fields.Add(&core.TextField{Name: "track_source", Max: 16})
			addedTrackSource = true
		}
		if collection.Fields.GetByName("track_resynced_at") == nil {
			collection.Fields.Add(&core.DateField{Name: "track_resynced_at"})
		}
		if err := app.Save(collection); err != nil {
			return err
		}
		if addedTrackSource {
			moved, legacy, err := util.ClassifyExternalReferencesBeforeTracking(app, referenceBackfillAppliedAt(app))
			if err != nil {
				return err
			}
			app.Logger().Info("classified external references from before track_source existed", "moved", moved, "legacy", legacy)
		}
		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("kind")
		collection.Fields.RemoveByName("track_source")
		collection.Fields.RemoveByName("track_resynced_at")
		return app.Save(collection)
	})
}

// referenceBackfillAppliedAt is when migration 1772400001 created references
// for trails imported before the reference table existed; zero if unknown.
func referenceBackfillAppliedAt(app core.App) time.Time {
	var applied int64
	err := app.DB().NewQuery("SELECT applied FROM _migrations WHERE file = {:file}").
		Bind(dbx.Params{"file": "1772400001_created_trail_external_reference.go"}).
		Row(&applied)
	if err != nil || applied <= 0 {
		return time.Time{}
	}
	// PocketBase 0.38 records microseconds; older versions recorded seconds
	// or milliseconds, so read the scale off the magnitude.
	switch {
	case applied > 1e14:
		return time.UnixMicro(applied)
	case applied > 1e11:
		return time.UnixMilli(applied)
	default:
		return time.Unix(applied, 0)
	}
}
