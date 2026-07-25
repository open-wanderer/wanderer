# Phase 28: Region Catalog Data Model & Seeding - Pattern Map

**Mapped:** 2026-07-24
**Files analyzed:** 4 (new) + 1 (modified)
**Analogs found:** 4 / 4 (matched by role/data-flow proximity; none exact since self-relation + fetch-transform tooling is structurally new to this repo)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `db/commands/seed_regions.go` | utility (Cobra dev-tool, fetch-transform-write) | file-I/O + batch | `db/commands/dedup.go` | role-match (structural: Cobra `*cobra.Command` builder, flags, `Run` closure) — data-flow differs (dedup reads/writes live DB; seed_regions fetches HTTP + writes JSON file, no DB) |
| `db/migrations/<ts>_create_regions_collection.go` | migration (schema + bulk seed) | CRUD (bulk insert) | `db/migrations/1780000005_add_other_category.go` | role-match (idempotency-guard + committed-file read shape is identical); scale differs (single record vs ~1,300-row bulk transaction) |
| `db/migrations/<ts>_create_regions_collection.go` (collection construction) | model/schema | CRUD | `db/migrations/1781000000_categories_redesign.go` (`createSubcategoriesCollection` / `saveCollectionFromJSON`) | role-match for "define a brand-new collection in a migration"; RESEARCH.md's Pattern 1 (Go SDK `core.NewBaseCollection` + `.Fields.Add`) is the recommended construction style instead of this file's JSON-blob style — see rationale below |
| `db/migrations/initial_data/regions_seed.json` | config/fixture (committed seed data) | file-I/O | `db/migrations/initial_data/other.jpg` (referenced via `filesystem.NewFileFromPath` in `1780000005_add_other_category.go`) | role-match (committed data file consumed by a migration) — different format (JSON vs image binary) but same "committed fixture under `initial_data/`" convention |
| `db/main.go` (`setupCommands`) | config (command registration) | request-response (CLI) | `db/main.go:164-166` (existing `setupCommands`) | exact — this is the file being modified, one line added |

## Pattern Assignments

### `db/commands/seed_regions.go` (utility, file-I/O/batch)

**Analog:** `db/commands/dedup.go`

**Imports pattern** (lines 1-12):
```go
package commands

import (
	"crypto/sha1"
	"fmt"
	"log"
	"sort"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)
```
For `seed_regions.go`, per RESEARCH.md Pattern 3, drop the `pocketbase`/`core` imports and the `*pocketbase.PocketBase` parameter entirely — this tool never touches a live DB. Add `net/http`, `bufio`, `encoding/json`, `os` instead.

**Command construction pattern** (lines 14-21, 100-103):
```go
func Dedup(app *pocketbase.PocketBase) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "dedup",
		Short: "Deduplicate trails by all matching fields",
		Run: func(cmd *cobra.Command, args []string) {
			// ... work ...
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show duplicates without deleting them")

	return cmd
}
```
Copy this exact shape for `SeedRegions() *cobra.Command` (no `app` param per Pattern 3), adding `cmd.Flags().StringVar(&commit, "commit", defaultCommitHash, "CoMaps commit hash to fetch from")` and an `--out` flag for the seed JSON output path.

**Error handling pattern** (lines 22-24, 84-87, 90-96):
```go
records, err := app.FindAllRecords("trails")
if err != nil {
	log.Fatalf("failed to fetch trails: %v", err)
}
// ...
if dryRun {
	fmt.Printf("\n[Dry Run] Found %d duplicates (no deletions performed)\n", len(duplicates))
	return
}
for _, d := range duplicates {
	if err := app.Delete(d); err != nil {
		fmt.Printf("Failed to delete duplicate %s: %v\n", d.Id, err)
	} else {
		fmt.Printf("Deleted duplicate %s\n", d.Id)
	}
}
```
Follow the `log.Fatalf` on unrecoverable setup errors (network fetch failure), `fmt.Printf` progress reporting per file/region parsed. Per RESEARCH.md's Security Domain section, never `panic` on malformed upstream `.poly`/`hierarchy.txt` lines — return descriptive errors up to a single top-level `log.Fatalf` in `Run`, tagged with which file/line failed.

---

### `db/migrations/<ts>_create_regions_collection.go` (migration, CRUD/schema+bulk-insert)

**Analog:** `db/migrations/1780000005_add_other_category.go`

**Full idempotency-guard + committed-file pattern** (lines 1-46, verbatim):
```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func init() {
	m.Register(func(app core.App) error {
		categories, err := app.FindAllRecords("categories")
		if err != nil {
			return err
		}
		if len(categories) == 0 {
			return nil
		}

		existing, _ := app.FindFirstRecordByData("categories", "name", "Other")
		if existing != nil {
			return nil
		}

		collection, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		record := core.NewRecord(collection)
		record.Set("name", "Other")
		record.Set("settings", map[string]any{
			"wp_merge_enabled": true,
			"wp_merge_radius":  50,
		})
		if file, err := filesystem.NewFileFromPath("migrations/initial_data/other.jpg"); err == nil {
			record.Set("img", file)
		}
		return app.Save(record)
	}, func(app core.App) error {
		record, _ := app.FindFirstRecordByData("categories", "name", "Other")
		if record == nil {
			return nil
		}
		return app.Delete(record)
	})
}
```
For `regions`, replace the single-record `FindFirstRecordByData` guard with `app.CountRecords("regions") > 0` (per RESEARCH.md Pitfall 5), replace `filesystem.NewFileFromPath` with `os.ReadFile("migrations/initial_data/regions_seed.json")` + `json.Unmarshal`, and replace the single `record.Set`/`app.Save` with the two-pass bulk-insert-then-link-parent loop inside `app.RunInTransaction(...)` given verbatim in RESEARCH.md's Pattern 2 (lines 208-282 of 28-RESEARCH.md) — reproduced there with the exact `SeedRow` struct fields (`comaps_id`, `parent_comaps_id`, `path`, `depth`, `sort_order`, `name`, `kind`, `polygon`, `bbox`) and the `enabled=false`-only-for-leaf guard.

**Collection construction — use Go SDK, NOT the JSON-blob style:**

Two candidate styles exist in this repo:
1. `1781000000_categories_redesign.go`'s `saveCollectionFromJSON` (JSON blob unmarshaled into `&core.Collection{}`, e.g. `createSubcategoriesCollection` lines 430-560) — this is what PocketBase's Admin UI `migratecmd` Automigrate feature auto-generates, and is the dominant style for *existing* collections in this repo (e.g. `1784658610_created_region_archives.go`).
2. RESEARCH.md's Pattern 1 — `core.NewBaseCollection("regions")` + `.Fields.Add(&core.TextField{...}, &core.RelationField{...}, ...)`.

**Use style 2 (Go SDK).** RESEARCH.md explicitly recommends this because: (a) this agentic workflow has no interactive Admin UI session to hand-generate a correct JSON blob from, (b) the self-referencing `parent` relation needs `collection.Id` available before `Save()` — confirmed `core.NewBaseCollection` synchronously assigns `Id` via `initDefaultId()`, so no two-pass create-then-patch dance is needed — and (c) hand-authoring the JSON blob risks subtle mistakes (wrong field `id`, missing `system`/`hidden` defaults) that the SDK path avoids by construction. See RESEARCH.md Pattern 1 (lines 161-202) for the full field list including `MaxSize: 8 << 20` on the `polygon` `JSONField` (Pitfall 1: default cap is 1MB) and the three `AddIndex` calls (`comaps_id` unique, `parent`, `path`).

**Down-migration pattern** (lines 39-45, `1780000005_add_other_category.go`, and RESEARCH.md Pattern 2 lines 274-281):
```go
}, func(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("regions")
	if err != nil {
		return nil
	}
	return app.Delete(collection)
})
```

---

### `db/migrations/initial_data/regions_seed.json` (fixture, file-I/O)

**Analog:** `db/migrations/initial_data/other.jpg` (referenced in `1780000005_add_other_category.go` line 35) and the `SeedRow` shape given in RESEARCH.md Pattern 2 (lines 230, 241-258).

No code excerpt needed — this is a data file, not source. Follow the directory convention: `db/migrations/initial_data/regions_seed.json`, a flat JSON array of row objects matching the `SeedRow` struct (`comaps_id`, `parent_comaps_id`, `path`, `depth`, `sort_order`, `name`, `kind`, `polygon`, `bbox`), pre-sorted parent-before-child per RESEARCH.md's ordering note (line 283).

---

### `db/main.go` (`setupCommands`, config/command-registration)

**Analog:** itself — the exact insertion point.

**Current pattern** (`db/main.go:164-166`):
```go
func setupCommands(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(commands.Dedup(app))
}
```

**Required change** (per RESEARCH.md Pattern 3, line 288-294):
```go
func setupCommands(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(commands.Dedup(app))
	app.RootCmd.AddCommand(commands.SeedRegions()) // NEW — no app param needed (writes a file, not the DB)
}
```

## Shared Patterns

### Migration registration and idempotency
**Source:** `db/migrations/1780000005_add_other_category.go` (guard-then-write shape); `db/migrations/1781000000_categories_redesign.go` (larger multi-step migration with helper functions and both up/down bodies)
**Apply to:** `db/migrations/<ts>_create_regions_collection.go`
```go
func init() {
	m.Register(func(app core.App) error {
		// up: idempotency guard, then collection creation + bulk insert
	}, func(app core.App) error {
		// down: delete the collection (records cascade via PocketBase)
	})
}
```

### Cobra command registration
**Source:** `db/main.go:164-166`, `db/commands/dedup.go:14-21`
**Apply to:** `db/commands/seed_regions.go` + `db/main.go`'s `setupCommands`
```go
app.RootCmd.AddCommand(commands.SeedRegions())
```

### Committed fixture files read at migration time
**Source:** `db/migrations/1780000005_add_other_category.go` line 35 (`filesystem.NewFileFromPath("migrations/initial_data/other.jpg")`)
**Apply to:** `db/migrations/<ts>_create_regions_collection.go` reading `migrations/initial_data/regions_seed.json` via `os.ReadFile` + `json.Unmarshal` (plain JSON, not a PocketBase `filesystem.File` wrapper, since it's structured data consumed programmatically, not stored as a record file field)

### JSON field read/write idiom on `core.Record`
**Source:** RESEARCH.md, verified against `db/pluginsystem/installed.go:74`, `db/routes/waypoint_cluster.go:118`, `db/migrations/1780000005_add_other_category.go:31` (`record.Set(name, map[string]any{...})`)
```go
record.Set("polygon", geojsonPolygonOrMultiPolygon) // any JSON-marshalable Go value
record.Set("bbox", [4]float64{minLon, minLat, maxLon, maxLat})
```
**Apply to:** the bulk-insert loop in `db/migrations/<ts>_create_regions_collection.go`

### Existing bbox precedent (do not reuse type, only as a naming/semantics cross-check)
**Source:** `db/services/regions/config.go` lines 30-40 — `Region.Bbox [4]float64` with element order `[minLon, minLat, maxLon, maxLat]` (matches `pmtiles extract --bbox=` argument order)
**Apply to:** when deriving `bbox` for the new `regions.bbox` JSON field, keep the same `[minLon, minLat, maxLon, maxLat]` element order for consistency with this existing convention, even though the storage type differs (JSON field vs `[4]float64` Go struct field). Do not modify `db/services/regions/config.go` itself — Phase 28 leaves it untouched per CONTEXT.md.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `.poly` parser (internal to `seed_regions.go`, not a separate file per RESEARCH.md's recommended structure but could be split into `db/commands/poly_parser.go`) | utility (pure parsing) | transform | No existing Go code in this repo parses Osmosis `.poly` or does ring-winding correction; RESEARCH.md's Code Examples section (verified against real CoMaps `.poly` files) and "Don't Hand-Roll" table are the only source of truth — implement per the ~50-line spec in D-03/D-04, not from an in-repo analog |
| `hierarchy.txt` indentation-tree parser (internal to `seed_regions.go`) | utility (pure parsing) | transform | Same as above — no existing in-repo analog for parsing a flat, space-indented adjacency structure into a materialized-path tree; use RESEARCH.md's Code Examples (hierarchy.txt structure, Pitfall 2's `World`/`WorldCoasts` skip rule, Open Question 1's name-derivation rule) directly |

## Metadata

**Analog search scope:** `db/commands/`, `db/migrations/`, `db/services/regions/`, `db/main.go`
**Files scanned:** `dedup.go`, `1780000005_add_other_category.go`, `1781000000_categories_redesign.go`, `1784658610_created_region_archives.go` (listed, not fully read — JSON-blob-style precedent already established via `1781000000`), `config.go`, `main.go`
**Pattern extraction date:** 2026-07-24
