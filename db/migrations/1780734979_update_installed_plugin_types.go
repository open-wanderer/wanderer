package migrations

import (
	"fmt"
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return setInstalledPluginTypes(app, []string{"trails", "assets"})
	}, func(app core.App) error {
		records, err := app.FindRecordsByFilter("installed_plugins", "type = 'assets'", "", -1, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}
		return setInstalledPluginTypes(app, []string{"trails"})
	})
}

func setInstalledPluginTypes(app core.App, values []string) error {
	collection, err := app.FindCollectionByNameOrId("installed_plugins")
	if err != nil {
		return err
	}

	field, ok := collection.Fields.GetByName("type").(*core.SelectField)
	if !ok {
		return fmt.Errorf("installed_plugins.type is not a select field")
	}

	if slices.Equal(field.Values, values) {
		return nil
	}
	field.Values = values
	return app.Save(collection)
}
