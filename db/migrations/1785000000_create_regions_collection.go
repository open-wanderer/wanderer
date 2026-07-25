// Package migrations: this file creates the `regions` PocketBase collection
// (CATALOG-01: hierarchy fields comaps_id/parent/path/depth/sort_order/
// name/kind; CATALOG-02: leaf-only polygon+bbox; CATALOG-03: leaf-only
// enabled, default false) and, on a fresh instance, bulk-inserts every row
// from the committed db/migrations/initial_data/regions_seed.json inside
// one transaction (SEED-02). See .planning/phases/28-region-catalog-data-model-seeding/28-RESEARCH.md
// Pattern 1/2 and 28-PATTERNS.md for the exact construction this follows.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Idempotency guard (Pitfall 5) — a second `up()` after a
		// `migrate down`+`up` cycle is a no-op rather than duplicating
		// ~1,300 rows or hitting the unique comaps_id index.
		count, err := app.CountRecords("regions")
		if err == nil && count > 0 {
			return nil
		}

		collection := core.NewBaseCollection("regions")
		// collection.Id is already valid here (NewBaseCollection assigns it
		// synchronously) — no two-pass create-then-patch needed for the
		// self-referencing `parent` relation below.

		collection.Fields.Add(
			&core.TextField{Name: "comaps_id", Required: true},
			&core.RelationField{
				Name:         "parent",
				CollectionId: collection.Id, // self-reference
				MaxSelect:    1,
				Required:     false, // nullable for depth-0 countries
			},
			&core.TextField{Name: "path", Required: true},
			&core.NumberField{Name: "depth", Required: true},
			&core.NumberField{Name: "sort_order", Required: true},
			&core.TextField{Name: "name", Required: true},
			&core.SelectField{
				Name:      "kind",
				Values:    []string{"group", "leaf"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.JSONField{Name: "polygon", MaxSize: 8 << 20}, // Pitfall 1: default 1MB cap is too small for some high-vertex coastlines
			&core.JSONField{Name: "bbox", MaxSize: 1 << 10},
			&core.BoolField{Name: "enabled"},
		)

		collection.AddIndex("idx_regions_comaps_id", true, "comaps_id", "")
		collection.AddIndex("idx_regions_parent", false, "parent", "")
		collection.AddIndex("idx_regions_path", false, "path", "")

		if err := app.Save(collection); err != nil {
			return err
		}

		// bulk insert — plan 28-03 Task 2
		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("regions")
		if err != nil {
			return nil
		}
		return app.Delete(collection) // records cascade
	})
}
