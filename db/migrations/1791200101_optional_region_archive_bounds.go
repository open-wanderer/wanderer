package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// A region whose geometry cannot be resolved on its first build has no bbox,
// but still needs a region_archives row to report the failure on.
func init() {
	m.Register(func(app core.App) error {
		return setRegionArchiveBoundsRequired1791200101(app, false)
	}, func(app core.App) error {
		return setRegionArchiveBoundsRequired1791200101(app, true)
	})
}

func setRegionArchiveBoundsRequired1791200101(app core.App, required bool) error {
	collection, err := app.FindCollectionByNameOrId("region_archives")
	if err != nil {
		return err
	}
	for _, name := range []string{"min_lon", "min_lat", "max_lon", "max_lat"} {
		field, ok := collection.Fields.GetByName(name).(*core.NumberField)
		if !ok {
			return fmt.Errorf("region_archives.%s is not a number field", name)
		}
		field.Required = required
	}
	return app.Save(collection)
}
