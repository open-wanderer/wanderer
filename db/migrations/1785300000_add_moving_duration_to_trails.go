// Package migrations: this file adds the moving_duration field to the trails
// collection.
//
// moving_duration is a new, separate, optional field. It exists
// specifically so that `duration` keeps exactly one meaning everywhere in
// the codebase — GPX-derived elapsed time (last trkpt time minus first) —
// and so nothing that recomputes `duration` from a route (e.g. the web
// trail-edit page's updateTotals()) is able to silently overwrite a
// recording's moving time. moving_duration is only ever written by capture
// paths that have a live session to take it from; no writer of it exists on
// the web at all.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		trailsCollection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}

		if trailsCollection.Fields.GetByName("moving_duration") == nil {
			movingDurationMin := float64(0)

			trailsCollection.Fields.Add(&core.NumberField{
				Name:        "moving_duration",
				Min:         &movingDurationMin,
				Max:         nil,
				OnlyInt:     false,
				Required:    false,
				Hidden:      false,
				Presentable: false,
				System:      false,
			})
		}

		return app.Save(trailsCollection)
	}, func(app core.App) error {
		trailsCollection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}

		trailsCollection.Fields.RemoveByName("moving_duration")

		return app.Save(trailsCollection)
	})
}
