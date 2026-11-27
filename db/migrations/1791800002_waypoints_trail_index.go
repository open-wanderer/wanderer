package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const waypointsTrailIndex = "idx_waypoints_trail"

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("waypoints")
		if err != nil {
			return err
		}
		collection.RemoveIndex(waypointsTrailIndex)
		collection.Indexes = append(collection.Indexes,
			"CREATE INDEX IF NOT EXISTS `"+waypointsTrailIndex+"` ON `waypoints` (`trail`)")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("waypoints")
		if err != nil {
			return err
		}
		collection.RemoveIndex(waypointsTrailIndex)
		return app.Save(collection)
	})
}
