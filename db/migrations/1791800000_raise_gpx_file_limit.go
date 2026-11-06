package migrations

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const gpxFileLimit1791800000 int64 = 25 << 20

// Uploads are sized by these fields, and PocketBase's 5 MiB default refuses a
// recorded track of a long day. A limit an admin chose stays as it is.
func init() {
	m.Register(func(app core.App) error {
		return setGpxFileLimit1791800000(app, core.DefaultFileFieldMaxSize, gpxFileLimit1791800000)
	}, func(app core.App) error {
		return setGpxFileLimit1791800000(app, gpxFileLimit1791800000, core.DefaultFileFieldMaxSize)
	})
}

func setGpxFileLimit1791800000(app core.App, from, to int64) error {
	for _, name := range []string{"trails", "summit_logs"} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}
		field, ok := collection.Fields.GetByName("gpx").(*core.FileField)
		if !ok {
			return errors.New(name + ".gpx is not a file field")
		}
		current := field.MaxSize
		if current <= 0 {
			current = core.DefaultFileFieldMaxSize
		}
		if current != from {
			continue
		}
		field.MaxSize = to
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}
