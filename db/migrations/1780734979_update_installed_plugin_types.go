package migrations

import (
	"fmt"
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return setInstalledAssetPluginTypePresent(app, true)
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
		return setInstalledAssetPluginTypePresent(app, false)
	})
}

// Asset-plugin migrations are deployed independently. Change only the asset
// type value and preserve types registered by other migration bundles.
func setInstalledAssetPluginTypePresent(app core.App, present bool) error {
	collection, err := app.FindCollectionByNameOrId("installed_plugins")
	if err != nil {
		return err
	}

	field, ok := collection.Fields.GetByName("type").(*core.SelectField)
	if !ok {
		return fmt.Errorf("installed_plugins.type is not a select field")
	}

	index := slices.Index(field.Values, "assets")
	if (present && index >= 0) || (!present && index < 0) {
		return nil
	}
	if present {
		field.Values = append(field.Values, "assets")
	} else {
		field.Values = slices.Delete(field.Values, index, index+1)
	}
	return app.Save(collection)
}
