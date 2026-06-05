package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const providerBackupColumn1780000006 = "provider_backup_1780000006"

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

		if err := app.Save(collection); err != nil {
			return err
		}

		if err := restoreProviderColumn1780000006(app); err != nil {
			return err
		}

		collection.Indexes = append(collection.Indexes, removedIndexes...)
		return app.Save(collection)
	}, func(app core.App) error {
		if err := backupProviderColumn1780000006(app); err != nil {
			return err
		}

		collection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}

		removedIndexes := stripProviderIndexes1780000006(collection)

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
