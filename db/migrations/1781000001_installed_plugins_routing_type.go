package migrations

import (
	"fmt"
	"slices"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addInstalledRoutingPluginType, removeInstalledRoutingPluginType)
}

func addInstalledRoutingPluginType(app core.App) error {
	return setInstalledRoutingPluginTypePresent(app, true)
}

func removeInstalledRoutingPluginType(app core.App) error {
	records, err := app.FindRecordsByFilter(
		"installed_plugins",
		"type={:type}",
		"",
		-1,
		0,
		dbx.Params{"type": "routing"},
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		instances, err := app.FindRecordsByFilter(
			"plugin_instances",
			"plugin_id={:plugin_id}",
			"",
			-1,
			0,
			dbx.Params{"plugin_id": record.GetString("plugin_id")},
		)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			if err := app.Delete(instance); err != nil {
				return err
			}
		}
		if err := app.Delete(record); err != nil {
			return err
		}
	}
	return setInstalledRoutingPluginTypePresent(app, false)
}

// Keep this helper local to the routing migration. Asset-plugin migrations are
// shipped independently, so this migration changes only the routing value and
// preserves every plugin type installed by another migration bundle.
func setInstalledRoutingPluginTypePresent(app core.App, present bool) error {
	collection, err := app.FindCollectionByNameOrId("installed_plugins")
	if err != nil {
		return err
	}
	field, ok := collection.Fields.GetByName("type").(*core.SelectField)
	if !ok {
		return fmt.Errorf("installed_plugins.type is not a select field")
	}
	index := slices.Index(field.Values, "routing")
	if (present && index >= 0) || (!present && index < 0) {
		return nil
	}
	if present {
		field.Values = append(field.Values, "routing")
	} else {
		field.Values = slices.Delete(field.Values, index, index+1)
	}
	return app.Save(collection)
}
