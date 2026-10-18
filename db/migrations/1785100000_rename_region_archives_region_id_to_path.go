package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Renames region_archives.region_id -> path. The field never held a record id
// or relation: it stores a regions.path materialized value (e.g.
// "canada.alberta.south"), the same string on-disk archive dirs and download
// URLs are named after. Renaming (reusing field id "text3477051923") preserves
// existing column data; the label changes and its index is recreated as UNIQUE
// (one archive per region). This aligns the name with regions.path /
// region_polygons.path — the three collections join purely by string-matching
// path (there is no DB relation between them; integrity is enforced by the
// ValidateRegionPathReferenceHandler hook instead).
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("region_archives")
		if err != nil {
			return err
		}

		// rename field (same id preserves data)
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"help": "",
			"hidden": false,
			"id": "text3477051923",
			"max": 0,
			"min": 0,
			"name": "path",
			"pattern": "",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// UNIQUE: there is exactly one archive row per region (findOrCreate in
		// services/regions/builder.go guarantees it), so the path->regions.path
		// join is provably 1:1 and a duplicate archive row is a data error the
		// DB should reject.
		collection.Indexes = []string{
			"CREATE UNIQUE INDEX `idx_region_archives_path` ON `region_archives` (`path`)",
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("region_archives")
		if err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"help": "",
			"hidden": false,
			"id": "text3477051923",
			"max": 0,
			"min": 0,
			"name": "region_id",
			"pattern": "",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		collection.Indexes = []string{
			"CREATE INDEX `idx_region_archives_region_id` ON `region_archives` (`region_id`)",
		}

		return app.Save(collection)
	})
}
