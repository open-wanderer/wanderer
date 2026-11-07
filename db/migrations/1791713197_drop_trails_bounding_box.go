package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Nothing reads the trails_bounding_box view: clients get their box from
// /api/v1/trail/bounding-box, which queries Meilisearch. Its index also misled
// the planner on other queries that filter on public, because PRAGMA optimize
// samples far too few rows per value of that column.
func init() {
	m.Register(func(app core.App) error {
		boundingBox, err := app.FindCollectionByNameOrId("trails_bounding_box")
		if err != nil {
			return err
		}
		if err := app.Delete(boundingBox); err != nil {
			return err
		}

		trails, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}
		trails.RemoveIndex("idx_trails_public_bounds")

		return app.Save(trails)
	}, func(app core.App) error {
		trails, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}
		trails.AddIndex("idx_trails_public_bounds", false, "`public`, `max_lat`, `max_lon`, `min_lat`, `min_lon`", "")
		if err := app.Save(trails); err != nil {
			return err
		}

		rule := `@request.auth.id != "" && @request.auth.id = user`
		boundingBox := core.NewViewCollection("trails_bounding_box", "urytyc428mwlbqq")
		boundingBox.ViewQuery = correlatedTrailBoundsViewQuery
		boundingBox.ViewRule = &rule

		return app.Save(boundingBox)
	})
}
