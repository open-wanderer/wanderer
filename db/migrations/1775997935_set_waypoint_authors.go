package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		wps, err := app.FindAllRecords("waypoints")
		if err != nil {
			return err
		}

		for _, wp := range wps {
			actor, err := app.FindFirstRecordByData("activitypub_actors", "user", wp.GetString("user"))
			if err != nil {
				return err
			}
			wp.Set("author", actor.Id)
			err = app.UnsafeWithoutHooks().Save(wp)
			if err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// add down queries...

		return nil
	})
}
