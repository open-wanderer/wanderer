// Package migrations: this file creates the `regions` PocketBase collection
// (CATALOG-01: hierarchy fields comaps_id/parent/path/depth/sort_order/
// name/kind; CATALOG-02: leaf-only polygon+bbox; CATALOG-03: leaf-only
// enabled, default false) and, on a fresh instance, bulk-inserts every row
// from the committed db/migrations/initial_data/regions_seed.json inside
// one transaction (SEED-02). See .planning/phases/28-region-catalog-data-model-seeding/28-RESEARCH.md
// Pattern 1/2 and 28-PATTERNS.md for the exact construction this follows.
package migrations

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// SeedRow is the reader-side counterpart of db/commands/seed_regions.go's
// writer SeedRow (plan 28-02) — JSON tags match byte-for-byte. Defined
// locally (not imported from package commands) to avoid a
// migrations->commands package dependency.
type SeedRow struct {
	ComapsID       string         `json:"comaps_id"`
	ParentComapsID string         `json:"parent_comaps_id"`
	Path           string         `json:"path"`
	Depth          int            `json:"depth"`
	SortOrder      int            `json:"sort_order"`
	Name           string         `json:"name"`
	Kind           string         `json:"kind"`
	Polygon        map[string]any `json:"polygon,omitempty"`
	Bbox           []float64      `json:"bbox,omitempty"`
}

func init() {
	m.Register(func(app core.App) error {
		// Idempotency guard (Pitfall 5) — a second `up()` after a
		// `migrate down`+`up` cycle is a no-op rather than duplicating
		// ~1,300 rows or hitting the unique path index.
		count, err := app.CountRecords("regions")
		if err == nil && count > 0 {
			return nil
		}

		collection := core.NewBaseCollection("regions")
		// collection.Id is assigned synchronously by NewBaseCollection, so
		// collection.Id is a valid, stable value here. However, PocketBase's
		// RelationField validation (core.checkCollectionId) resolves
		// CollectionId via app.FindCachedCollectionByNameOrId — which only
		// finds collections that already exist in the app, not one still
		// being created in the same Save() call. A true self-reference
		// therefore needs two Save() passes: (1) save every field except
		// the self-relation, so the collection exists; (2) add `parent`
		// (now resolvable) and save again.
		collection.Fields.Add(
			&core.TextField{Name: "comaps_id", Required: true},
			&core.TextField{Name: "path", Required: true},
			&core.NumberField{Name: "depth"},      // Required omitted: PocketBase's NumberField.Required means "non-zero", but depth=0 (every top-level region) is a legitimate value
			&core.NumberField{Name: "sort_order"}, // same reasoning — sort_order=0 (first sibling in a group) is legitimate
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

		// comaps_id is NOT globally unique in real CoMaps data: 5 disputed/
		// shared-territory leaves (Abkhazia, South Ossetia, Jerusalem,
		// Crimea, Campo de Hielo Sur) legitimately appear twice under
		// different parents (e.g. Jerusalem under both "Israel Region" and
		// "Palestine Region"). `path` (parent-slug-prefixed) IS globally
		// unique across the full seed — verified against the committed
		// regions_seed.json (1306/1306 unique paths) — so `path` carries
		// the unique index and comaps_id's index stays a plain lookup index.
		collection.AddIndex("idx_regions_comaps_id", false, "comaps_id", "")
		collection.AddIndex("idx_regions_path", true, "path", "")

		if err := app.Save(collection); err != nil {
			return err
		}

		collection.Fields.Add(&core.RelationField{
			Name:         "parent",
			CollectionId: collection.Id, // self-reference, now resolvable
			MaxSelect:    1,
			Required:     false, // nullable for depth-0 countries
		})
		collection.AddIndex("idx_regions_parent", false, "parent", "")

		if err := app.Save(collection); err != nil {
			return err
		}

		// bulk insert — plan 28-03 Task 2
		data, err := os.ReadFile("migrations/initial_data/regions_seed.json")
		if err != nil {
			return fmt.Errorf("read regions seed: %w", err)
		}

		var seed []SeedRow
		if err := json.Unmarshal(data, &seed); err != nil {
			return fmt.Errorf("parse regions seed: %w", err)
		}

		return app.RunInTransaction(func(txApp core.App) error {
			// Keyed by `path`, not `comaps_id` — comaps_id collides for 5
			// real disputed-territory leaves (see the index comment above),
			// but `path` (parent-slug-prefixed) is provably unique across
			// the whole seed, so it's the safe join key for parent
			// resolution below.
			idByPath := make(map[string]string, len(seed))

			// Pass 1: create every record, capturing generated ids.
			for _, row := range seed {
				record := core.NewRecord(collection)
				record.Set("comaps_id", row.ComapsID)
				record.Set("path", row.Path)
				record.Set("depth", row.Depth)
				record.Set("sort_order", row.SortOrder)
				record.Set("name", row.Name)
				record.Set("kind", row.Kind)
				if row.Kind == "leaf" {
					record.Set("polygon", row.Polygon)
					record.Set("bbox", row.Bbox)
					record.Set("enabled", false) // CATALOG-03: leaf-only, never pre-enabled
				}
				if err := txApp.Save(record); err != nil {
					return fmt.Errorf("insert region %s (%s): %w", row.ComapsID, row.Path, err)
				}
				idByPath[row.Path] = record.Id
			}

			// Pass 2: resolve parent links now that every id is known
			// (defensive against any future re-ordering of the seed file;
			// the committed seed is already parent-before-child). A row's
			// parent is looked up by its own path's parent segment
			// (path minus its last "."-delimited component), not by
			// parent_comaps_id — see the idByPath comment above.
			for _, row := range seed {
				if row.ParentComapsID == "" {
					continue
				}
				sep := strings.LastIndex(row.Path, ".")
				if sep < 0 {
					return fmt.Errorf("region %s: has parent_comaps_id %q but path %q has no parent segment", row.ComapsID, row.ParentComapsID, row.Path)
				}
				parentID, ok := idByPath[row.Path[:sep]]
				if !ok {
					return fmt.Errorf("region %s: parent path %q not found", row.ComapsID, row.Path[:sep])
				}
				record, err := txApp.FindRecordById("regions", idByPath[row.Path])
				if err != nil {
					return err
				}
				record.Set("parent", parentID)
				if err := txApp.Save(record); err != nil {
					return fmt.Errorf("link parent for %s: %w", row.ComapsID, err)
				}
			}

			return nil
		})
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("regions")
		if err != nil {
			return nil
		}
		return app.Delete(collection) // records cascade
	})
}
