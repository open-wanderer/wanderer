// Package migrations: this file creates the `regions` PocketBase collection
// (CATALOG-01: hierarchy fields comaps_id/parent/path/depth/sort_order/
// name/kind; CATALOG-02: leaf-only bbox; CATALOG-03: leaf-only enabled,
// default false) and the `region_polygons` collection (leaf polygon
// geometry, keyed by `path` — kept out of `regions` from the start so
// hierarchy/listing queries never load ~165KB/row of boundary geometry;
// see region_polygons' own field comments below), then, on a fresh
// instance, bulk-inserts every row from the committed
// db/migrations/initial_data/regions_seed.json.gz (gzip-compressed compact
// JSON, 28-04) into both collections inside one transaction (SEED-02).
// See .planning/phases/28-region-catalog-data-model-seeding/28-RESEARCH.md
// Pattern 1/2 and 28-PATTERNS.md for the exact construction this follows.
package migrations

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
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

		// region_polygons: leaf boundary geometry, kept out of `regions`
		// entirely. Keyed by `path` (plain TextField, not a relation) —
		// matches region_archives.region_id's existing path-keyed join
		// pattern (A2), avoiding relation-resolution overhead for a side
		// table that only services/regions/builder.go's buildRegion ever
		// reads (at archive-build time, only for currently-enabled
		// leaves). Every other reader of `regions` (GET /api/v1/regions,
		// the admin region picker) only needs hierarchy/bbox/enabled, so
		// this keeps every listing query's SELECT free of ~165KB/row of
		// full-precision boundary data.
		polyCollection := core.NewBaseCollection("region_polygons")
		polyCollection.Fields.Add(
			&core.TextField{Name: "path", Required: true},
			&core.JSONField{Name: "polygon", MaxSize: 8 << 20}, // Pitfall 1: default 1MB cap is too small for some high-vertex coastlines
		)
		polyCollection.AddIndex("idx_region_polygons_path", true, "path", "")

		if err := app.Save(polyCollection); err != nil {
			return err
		}

		// bulk insert — plan 28-03 Task 2 (28-04: seed is now gzip-
		// compressed compact JSON — the uncompressed 730 MB artifact
		// exceeded GitHub's 100 MB per-file push limit).
		seedFile, err := os.Open("migrations/initial_data/regions_seed.json.gz")
		if err != nil {
			return fmt.Errorf("open regions seed: %w", err)
		}
		defer seedFile.Close()

		gzReader, err := gzip.NewReader(seedFile)
		if err != nil {
			return fmt.Errorf("open gzip reader for regions seed: %w", err)
		}
		defer gzReader.Close()

		// T-28-09: the seed is a trusted in-repo artifact of known ~216 MB
		// decompressed size, read exactly once behind the CountRecords
		// idempotency guard above; bound the read against a pathological/
		// corrupt stream rather than trusting an unbounded decompression.
		//
		// Decoded token-by-token (not io.ReadAll + json.Unmarshal into one
		// []SeedRow) so peak heap holds one row at a time instead of the
		// full decompressed seed plus its parsed Go representation
		// (map[string]any polygons balloon several times over raw JSON
		// size) — the earlier all-at-once approach could exceed 512 MB on
		// memory-constrained hosts. Only path→id and the small
		// (comaps_id, path, parent_comaps_id) tuples needed for parent
		// linking survive past each row's processing.
		dec := json.NewDecoder(io.LimitReader(gzReader, 512<<20))
		if _, err := dec.Token(); err != nil { // consume opening '['
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
			for dec.More() {
				var row SeedRow
				if err := dec.Decode(&row); err != nil {
					return fmt.Errorf("parse regions seed: %w", err)
				}

				record := core.NewRecord(collection)
				record.Set("comaps_id", row.ComapsID)
				record.Set("path", row.Path)
				record.Set("depth", row.Depth)
				record.Set("sort_order", row.SortOrder)
				record.Set("name", row.Name)
				record.Set("kind", row.Kind)
				if row.Kind == "leaf" {
					record.Set("bbox", row.Bbox)
					record.Set("enabled", false) // CATALOG-03: leaf-only, never pre-enabled
				}
				if err := txApp.Save(record); err != nil {
					return fmt.Errorf("insert region %s (%s): %w", row.ComapsID, row.Path, err)
				}
				idByPath[row.Path] = record.Id

				if row.Kind == "leaf" && len(row.Polygon) > 0 {
					polyRecord := core.NewRecord(polyCollection)
					polyRecord.Set("path", row.Path)
					polyRecord.Set("polygon", row.Polygon)
					if err := txApp.Save(polyRecord); err != nil {
						return fmt.Errorf("insert region_polygons %s (%s): %w", row.ComapsID, row.Path, err)
					}
				}

				if row.ParentComapsID != "" {
					links = append(links, parentLink{ComapsID: row.ComapsID, Path: row.Path, ParentComapsID: row.ParentComapsID})
				}
			}
			if _, err := dec.Token(); err != nil { // consume closing ']'
				return fmt.Errorf("parse regions seed: %w", err)
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
		if collection, err := app.FindCollectionByNameOrId("region_polygons"); err == nil {
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
