package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"pocketbase/util"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindAllRecords("users")
		if err != nil {
			return err
		}

		for _, u := range users {
			_, err = util.ActorFromUser(app, u)
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
