package migrations

import (
	"github.com/pocketbase/dbx"
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
			trail, err := app.FindFirstRecordByFilter("trails", "waypoints ?~ {:id}", dbx.Params{"id": wp.Id})
			if err != nil {
				continue
			}
			wp.Set("trail", trail.Id)
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
