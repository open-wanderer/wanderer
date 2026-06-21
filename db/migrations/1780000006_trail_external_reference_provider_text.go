package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const providerBackupColumn1780000006 = "provider_backup_1780000006"
const userPluginIndex1780000006 = "CREATE INDEX `idx_trail_external_reference_user_plugin_id` ON `trail_external_reference` (`user`, `plugin_id`)"

func init() {
	m.Register(func(app core.App) error {
		if err := backupProviderColumn1780000006(app); err != nil {
			return err
		}

		collection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}

		// Drop+re-add (with a new id) blanks the provider column, so the
		// provider-scoped unique indexes must not be rebuilt until the values
		// have been restored, otherwise a cross-provider external_id clash would
		// fail index creation and abort the migration.
		removedIndexes := stripProviderIndexes1780000006(collection)

		collection.Fields.RemoveByName("provider")
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text420001002",
			"max": 128,
			"min": 1,
			"name": "provider",
			"pattern": "^[a-z0-9][a-z0-9_-]*$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}
		if collection.Fields.GetByName("plugin_id") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "textpluginref",
				"max": 64,
				"min": 0,
				"name": "plugin_id",
				"pattern": "^[a-z0-9][a-z0-9_-]*$",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			}`)); err != nil {
				return err
			}
		}
		if collection.Fields.GetByName("provider_category") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "txtrmtecat01",
				"max": 255,
				"min": 0,
				"name": "provider_category",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			}`)); err != nil {
				return err
			}
		}
		if collection.Fields.GetByName("provider_category_checked_at") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
				"hidden": false,
				"id": "datermtecat1",
				"max": "",
				"min": "",
				"name": "provider_category_checked_at",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			}`)); err != nil {
				return err
			}
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		if err := restoreProviderColumn1780000006(app); err != nil {
			return err
		}

		collection.Indexes = append(collection.Indexes, removedIndexes...)
		if !hasIndex1780000006(collection, userPluginIndex1780000006) {
			collection.Indexes = append(collection.Indexes, userPluginIndex1780000006)
		}
		if err := app.Save(collection); err != nil {
			return err
		}

		refs, err := app.FindAllRecords("trail_external_reference")
		if err != nil {
			return err
		}
		for _, ref := range refs {
			if ref.GetString("plugin_id") != "" {
				continue
			}
			ref.Set("plugin_id", ref.GetString("provider"))
			if err := app.Save(ref); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		if err := backupProviderColumn1780000006(app); err != nil {
			return err
		}

		collection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}

		removedIndexes := stripProviderIndexes1780000006(collection)
		removeIndex1780000006(collection, userPluginIndex1780000006)

		collection.Fields.RemoveByName("provider_category_checked_at")
		collection.Fields.RemoveByName("provider_category")
		collection.Fields.RemoveByName("plugin_id")
		collection.Fields.RemoveByName("provider")
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"hidden": false,
			"id": "select420001002",
			"maxSelect": 1,
			"name": "provider",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"strava",
				"komoot",
				"hammerhead"
			]
		}`)); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		if err := restoreProviderColumn1780000006(app); err != nil {
			return err
		}

		collection.Indexes = append(collection.Indexes, removedIndexes...)
		return app.Save(collection)
	})
}

// stripProviderIndexes1780000006 removes the provider-scoped indexes from the
// collection metadata and returns them so they can be re-added once the
// provider values have been restored. PocketBase rebuilds indexes from the
// collection metadata on every save; leaving the provider indexes in place
// while the column is transiently empty risks a unique-constraint failure.
func stripProviderIndexes1780000006(collection *core.Collection) []string {
	kept := make([]string, 0, len(collection.Indexes))
	removed := make([]string, 0)
	for _, idx := range collection.Indexes {
		if strings.Contains(idx, "`provider`") {
			removed = append(removed, idx)
			continue
		}
		kept = append(kept, idx)
	}
	collection.Indexes = kept
	return removed
}

func hasIndex1780000006(collection *core.Collection, index string) bool {
	for _, existing := range collection.Indexes {
		if existing == index {
			return true
		}
	}
	return false
}

func removeIndex1780000006(collection *core.Collection, index string) {
	indexes := collection.Indexes[:0]
	for _, existing := range collection.Indexes {
		if existing == index {
			continue
		}
		indexes = append(indexes, existing)
	}
	collection.Indexes = indexes
}

func backupProviderColumn1780000006(app core.App) error {
	exists, err := columnExists1780000006(app, providerBackupColumn1780000006)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	if _, err := app.DB().
		NewQuery("ALTER TABLE trail_external_reference ADD COLUMN " + providerBackupColumn1780000006 + " TEXT DEFAULT '' NOT NULL").
		Execute(); err != nil {
		return err
	}

	_, err = app.DB().
		NewQuery("UPDATE trail_external_reference SET " + providerBackupColumn1780000006 + " = provider").
		Execute()
	return err
}

func restoreProviderColumn1780000006(app core.App) error {
	if _, err := app.DB().
		NewQuery("UPDATE trail_external_reference SET provider = " + providerBackupColumn1780000006).
		Execute(); err != nil {
		return err
	}

	_, err := app.DB().DropColumn("trail_external_reference", providerBackupColumn1780000006).Execute()
	return err
}

func columnExists1780000006(app core.App, column string) (bool, error) {
	columns, err := app.TableColumns("trail_external_reference")
	if err != nil {
		return false, err
	}

	for _, existing := range columns {
		if existing == column {
			return true, nil
		}
	}

	return false, nil
}
