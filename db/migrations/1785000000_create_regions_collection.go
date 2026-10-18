// Package migrations: this file creates the `regions` PocketBase collection
// (hierarchy fields comaps_id/parent/path/depth/sort_order/name/kind;
// leaf-only `enabled`, default false; every row also carries
// `catalog_commit`, the pinned CoMaps commit SHA the seed artifact was
// generated from) and the `region_geometry` collection (leaf
// boundary geometry — both `bbox` and `polygon` — keyed by `path`, kept out
// of `regions` from the start so hierarchy/listing queries never load
// geometry; see region_geometry's own field comments below). `region_geometry`
// ships empty: it is populated on demand by the runtime geometry fetch
// path, not by this migration. On a fresh
// instance, this migration bulk-inserts every hierarchy row from the
// committed, plain-JSON db/migrations/initial_data/regions_seed.json
// inside one transaction.
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
// writer SeedRow (plan 32-01) — JSON tags match byte-for-byte. Defined
// locally (not imported from package commands) to avoid a
// migrations->commands package dependency. Carries pure hierarchy only — no
// geometry: a leaf's polygon/bbox are fetched on demand at
// runtime instead of being carried in this catalog.
type SeedRow struct {
	ComapsID       string `json:"comaps_id"`
	ParentComapsID string `json:"parent_comaps_id"`
	Path           string `json:"path"`
	Depth          int    `json:"depth"`
	SortOrder      int    `json:"sort_order"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
}

// SeedCatalog is the reader-side counterpart of db/commands/seed_regions.go's
// writer SeedCatalog (plan 32-01): the pinned CoMaps commit this catalog was
// generated from, recorded once, plus every flattened hierarchy row.
type SeedCatalog struct {
	Commit string    `json:"commit"`
	Rows   []SeedRow `json:"rows"`
}

func init() {
	m.Register(func(app core.App) error {
		// Idempotency guard — a second `up()` after a
		// `migrate down`+`up` cycle is a no-op rather than duplicating
		// ~1,300 rows or hitting the unique path index.
		//
		// Execution note: this guard means an already-seeded dev box
		// will NOT re-run this migration after it is edited in place, and
		// will therefore retain a stale, orphaned pre-rename geometry
		// collection from before this edit. Resetting local pb_data (or
		// manually dropping the collection) is a required step on
		// already-seeded dev instances — accepted knowingly over shipping
		// a defensive drop migration (no fleet of production instances
		// carries the old collection; this migration has never shipped).
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
			// catalog_commit: the pinned CoMaps commit SHA the
			// seed artifact was generated from, carried on every
			// row rather than a shared Go const so it can never silently
			// desync from what was actually fetched. Both runtime
			// consumers of geometry (buildRegion, plan 32-04; the
			// geometry route, plan 32-05) already hold the leaf's
			// `regions` record, so carrying the commit here costs zero
			// extra lookups.
			&core.TextField{Name: "catalog_commit", Required: true},
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

		// region_geometry: leaf
		// boundary geometry, kept out of `regions` entirely. Keyed by `path`
		// (plain TextField, not a relation) — matches region_archives's
		// existing path-keyed join pattern (A2), avoiding relation-
		// resolution overhead for a side table that only the runtime
		// geometry fetch path (buildRegion and the geometry route) and the
		// admin SPA ever read. It holds both `bbox` (moved here from
		// `regions`) and `polygon`, and ships
		// empty: this migration performs no insert into it. It is
		// populated on demand — the first time a currently-enabled region's
		// geometry is fetched from CoMaps — rather than at seed time.
		geometryCollection := core.NewBaseCollection("region_geometry")
		geometryCollection.Fields.Add(
			&core.TextField{Name: "path", Required: true},
			&core.JSONField{Name: "polygon", MaxSize: 8 << 20}, // the default 1MB cap is too small for some high-vertex coastlines
			&core.JSONField{Name: "bbox", MaxSize: 1 << 10},
		)
		geometryCollection.AddIndex("idx_region_geometry_path", true, "path", "")

		if err := app.Save(geometryCollection); err != nil {
			return err
		}

		// Read the committed, plain-JSON hierarchy-only catalog:
		// no gzip layer, no streaming decoder, no decompression-bomb bound.
		// Those retired with the polygons — the artifact is now ~292 KB, not
		// ~216 MB, so a plain ReadFile + Unmarshal is entirely adequate; the
		// bound retired because the problem it guarded against (an
		// unbounded decompression of a many-hundred-MB stream) no longer
		// exists, not because bounding was wrong in principle.
		data, err := os.ReadFile("migrations/initial_data/regions_seed.json")
		if err != nil {
			return fmt.Errorf("read regions seed: %w", err)
		}

		var catalog SeedCatalog
		if err := json.Unmarshal(data, &catalog); err != nil {
			return fmt.Errorf("parse regions seed: %w", err)
		}

		type parentLink struct {
			ComapsID       string
			Path           string
			ParentComapsID string
		}

		return app.RunInTransaction(func(txApp core.App) error {
			// Keyed by `path`, not `comaps_id` — comaps_id collides for 5
			// real disputed-territory leaves (see the index comment above),
			// but `path` (parent-slug-prefixed) is provably unique across
			// the whole seed, so it's the safe join key for parent
			// resolution below.
			idByPath := make(map[string]string, 1536)
			links := make([]parentLink, 0, 1536)

			// Pass 1: create every record, capturing generated ids.
			for _, row := range catalog.Rows {
				record := core.NewRecord(collection)
				record.Set("comaps_id", row.ComapsID)
				record.Set("path", row.Path)
				record.Set("depth", row.Depth)
				record.Set("sort_order", row.SortOrder)
				record.Set("name", row.Name)
				record.Set("kind", row.Kind)
				record.Set("catalog_commit", catalog.Commit)
				if row.Kind == "leaf" {
					record.Set("enabled", false) // leaf-only, never pre-enabled
				}
				if err := txApp.Save(record); err != nil {
					return fmt.Errorf("insert region %s (%s): %w", row.ComapsID, row.Path, err)
				}
				idByPath[row.Path] = record.Id

				if row.ParentComapsID != "" {
					links = append(links, parentLink{ComapsID: row.ComapsID, Path: row.Path, ParentComapsID: row.ParentComapsID})
				}
			}

			// Pass 2: resolve parent links now that every id is known
			// (defensive against any future re-ordering of the seed file;
			// the committed seed is already parent-before-child). A row's
			// parent is looked up by its own path's parent segment
			// (path minus its last "."-delimited component), not by
			// parent_comaps_id — see the idByPath comment above.
			for _, link := range links {
				sep := strings.LastIndex(link.Path, ".")
				if sep < 0 {
					return fmt.Errorf("region %s: has parent_comaps_id %q but path %q has no parent segment", link.ComapsID, link.ParentComapsID, link.Path)
				}
				parentID, ok := idByPath[link.Path[:sep]]
				if !ok {
					return fmt.Errorf("region %s: parent path %q not found", link.ComapsID, link.Path[:sep])
				}
				record, err := txApp.FindRecordById("regions", idByPath[link.Path])
				if err != nil {
					return err
				}
				record.Set("parent", parentID)
				if err := txApp.Save(record); err != nil {
					return fmt.Errorf("link parent for %s: %w", link.ComapsID, err)
				}
			}

			return nil
		})
	}, func(app core.App) error {
		if collection, err := app.FindCollectionByNameOrId("region_geometry"); err == nil {
			if err := app.Delete(collection); err != nil { // records cascade
				return err
			}
		}
		collection, err := app.FindCollectionByNameOrId("regions")
		if err != nil {
			return nil
		}
		return app.Delete(collection) // records cascade
	})
}
