package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"pocketbase/util"
)

func init() {
	m.Register(func(app core.App) error {
		const pageSize int64 = 100
		var page int64

		for {
			trails := []*core.Record{}
			if err := app.RecordQuery("trails").
				Limit(pageSize).
				Offset(page * pageSize).
				All(&trails); err != nil {
				return fmt.Errorf("failed to query trails: %w", err)
			}
			if len(trails) == 0 {
				break
			}

			for _, trail := range trails {
				if _, err := util.UpdateTrailBoundingBox(app, trail); err != nil {
					return fmt.Errorf("failed to update bbox for trail %s: %w", trail.Id, err)
				}
			}

			page++
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}
