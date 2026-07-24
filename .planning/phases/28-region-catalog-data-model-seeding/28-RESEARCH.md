# Phase 28: Region Catalog Data Model & Seeding - Research

**Researched:** 2026-07-24
**Domain:** Go/PocketBase schema migrations + a hand-rolled CoMaps hierarchy/`.poly` parser
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### CoMaps Snapshot Sourcing
- **D-01:** `seed_regions.go` fetches `hierarchy.txt` + `.poly` files fresh from Codeberg (`comaps/comaps`) at tool-run time — nothing raw is vendored/committed to this repo. Only the tool's flattened JSON output is committed.
- **D-02:** The commit hash to fetch is a CLI flag (e.g. `--commit=<hash>`) with a sensible default baked in, so a maintainer can override ad hoc without editing source. Refresh flow: bump the default (or pass `--commit`), re-run, review the JSON diff, commit, ship as a normal migration.

#### .poly → GeoJSON Parsing
- **D-03:** Hand-roll a minimal Go parser for CoMaps' Osmosis-format `.poly` files (~50 lines: name header, ring blocks terminated by `END`, `!`-prefixed inner rings) rather than pulling in a new dependency for this one-off maintainer tool.
- **D-04:** The parser must support multi-ring geometry from day one — both holes (`!`-prefixed inner rings, e.g. lakes) and multi-part regions (e.g. a country with islands/exclaves). Output GeoJSON `Polygon` for single-ring regions, `MultiPolygon` when a region has multiple outer rings. Skipping this would silently produce wrong/simplified boundaries for coastal or multi-part regions with no easy way to notice.

### Claude's Discretion
- Exact seed JSON file name/location under `db/migrations/initial_data/` (or similar) — follow existing migration conventions (see `db/migrations/1780000005_add_other_category.go` for the `initial_data`-relative-file pattern).
- Internal structure of the fetch step (HTTP client choice, caching of fetched files during a single tool run, error handling for network failures) — this is a maintainer-run dev-time tool, not production code, so keep it simple.
- `sort_order` derivation — preserve CoMaps' own sibling ordering as encountered in `hierarchy.txt`.

### Deferred Ideas (OUT OF SCOPE)
None raised during this discussion — it stayed within phase scope. (Broader deferred items — CATALOG-F01 automated refresh, CATALOG-F02 cascading enable, CATALOG-F03 group-level map preview — are already tracked in `.planning/REQUIREMENTS.md` and don't need re-capturing here.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| CATALOG-01 | Backend `regions` PocketBase collection stores each CoMaps hierarchy node with `comaps_id`, self-referencing `parent`, materialized `path`, `depth`, `sort_order`, `name`, and `kind` (`group`\|`leaf`) | Pattern 1 (self-referencing relation field, verified `collection.Id` timing) gives the exact field construction; Code Examples section gives verified `hierarchy.txt` structure to derive `path`/`depth`/`sort_order`/`name`/`kind` from |
| CATALOG-02 | Leaf rows additionally store a canonical `polygon` (GeoJSON, converted from CoMaps `.poly`) and a derived `bbox` | Code Examples' `.poly` structure (single-ring + multi-ring Fiji example) + Pitfall 1 (JSON field size cap) + Pitfall 4 (winding order) give the concrete conversion requirements |
| CATALOG-03 | Leaf rows carry an `enabled` boolean (default false); group rows carry no `enabled`/`polygon`/`bbox` semantics | Pattern 2's migration example explicitly sets `enabled=false` only when `kind == "leaf"`, leaving group rows without those fields set |
| SEED-01 | A maintainer-run `db/commands/seed_regions.go` command parses vendored CoMaps `hierarchy.txt` + `.poly` files and writes a flattened JSON seed file matching the `regions` schema | Pattern 3 (Cobra registration without an `app` param) + the full hierarchy.txt/.poly format research (Architecture Patterns, Code Examples) + Open Question 3 (commit-hash default) |
| SEED-02 | A standard PocketBase migration creates the `regions` collection and bulk-inserts from the committed JSON seed automatically on every instance startup | Pattern 2 (idempotent bulk-insert migration via `app.RunInTransaction`) + confirmation that `core.AppMigrations`/`RunAppMigrations()` runs automatically on every `OnServe` per the vendored source |
</phase_requirements>

## Summary

This phase is two small, well-precedented Go pieces bolted onto conventions the codebase already has: a `db/commands/seed_regions.go` Cobra tool that fetches CoMaps' `data/hierarchy.txt` + `data/borders/*.poly` files fresh from Codeberg at tool-run time, flattens them into a JSON seed file, and a standard `db/migrations/*.go` migration that creates the `regions` collection and bulk-inserts from that committed JSON on every startup. Neither piece needs a new Go dependency — Cobra is already vendored (used by `dedup.go`), and `.poly`'s Osmosis format is small enough to hand-parse (confirmed against real CoMaps files below).

The two structurally new things this phase introduces to the codebase are (1) a **self-referencing PocketBase relation field** (`parent` pointing at the `regions` collection itself) and (2) a **GeoJSON polygon/bbox JSON field pair with no existing Go type to reuse** — `db/services/regions/config.go`'s `Region.Bbox [4]float64` is the only precedent, and it's a plain array, not GeoJSON. Both are straightforward with the PocketBase v0.38.0 Go SDK (verified directly against the vendored source in `$GOMODCACHE`, not just docs): `core.NewBaseCollection(name)` synchronously generates `collection.Id` before `Save()`, so a self-relation's `CollectionId` can point at the collection's own id immediately — no two-pass "create collection, then patch in the self-reference" dance is needed.

The actual CoMaps hierarchy was fetched and inspected directly (not assumed from training data): `hierarchy.txt` is flat, semicolon-delimited, indented with **exactly one literal space character per depth level** (max depth observed: 2 — country → state/province → district), with two non-region header lines (`World`, `WorldCoasts`) that must be explicitly skipped. `.poly` files use the standard Osmosis format (ring header, tab-separated `lon lat` coordinate pairs in scientific notation, `END` terminators); a real multi-outer-ring example (`Fiji.poly`, split across the antimeridian) was found and inspected, confirming D-04's MultiPolygon requirement is not theoretical. No live example of a `!`-prefixed hole ring was found in a small sample, but the format is unambiguously documented (OSM wiki) and D-04 already locks in support for it regardless.

**Primary recommendation:** Build the `regions` collection with the Go SDK (`core.NewBaseCollection` + `.Fields.Add(...)`), not a hand-copied JSON collection blob — the JSON-blob migrations in this repo (`1784658610_created_region_archives.go` etc.) are auto-generated snapshots from the PocketBase Admin UI's `migratecmd` Automigrate feature, which this agentic workflow has no interactive access to. Write `seed_regions.go` as a pure fetch-transform-write tool that does **not** take a `*pocketbase.PocketBase` parameter (unlike `dedup.go`) since it never touches a live database — it only writes a JSON file to disk.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CoMaps hierarchy/`.poly` fetch + flatten (dev-time tool) | API/Backend (Go, dev-time Cobra command) | — | Maintainer-run, not part of the running service; lives in `db/commands/` alongside `dedup.go` |
| `regions` collection schema definition | API/Backend (Go migration) | Database/Storage (PocketBase/SQLite) | Migrations are the only schema-authority mechanism in this codebase; PocketBase persists the schema into its own SQLite `_collections` table |
| Seed data bulk-insert | API/Backend (Go migration, `app.RunInTransaction`) | Database/Storage | Runs automatically on every startup per `migratecmd`'s `RunAppMigrations()`; satisfies SEED-02's "zero admin action" requirement |
| Polygon/bbox geometry storage | Database/Storage (JSON columns) | — | No API surface exposes this yet (Phase 29's concern); this phase only persists it |

## Package Legitimacy Audit

No new external packages are installed this phase. `github.com/spf13/cobra v1.10.2` is already a direct dependency (`db/go.mod`), used by the existing `db/commands/dedup.go`. D-03 explicitly rejects adding a `.poly`/geometry-parsing dependency in favor of a hand-rolled ~50-line parser. No `pip`/`npm`/`cargo` installs apply — this is a Go-only backend phase.

**Packages removed due to slopcheck `[SLOP]` verdict:** none (nothing installed)
**Packages flagged as suspicious `[SUS]`:** none (nothing installed)

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/pocketbase/pocketbase` | v0.38.0 `[VERIFIED: db/go.mod]` | Collection/field/migration Go SDK | Already the project's backend framework; verified directly against the vendored source at `$GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0` |
| `github.com/spf13/cobra` | v1.10.2 `[VERIFIED: db/go.mod]` | CLI command definition for `seed_regions.go` | Already used by `db/commands/dedup.go`; already a direct dependency, no new install |
| `encoding/json` (stdlib) | Go 1.25 `[VERIFIED: db/go.mod toolchain]` | Flattened seed file read/write | No third-party JSON library used anywhere else in `db/` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `net/http` (stdlib) | Go 1.25 | Fetch `hierarchy.txt`/`.poly` files from Codeberg's raw content URLs at tool-run time | D-01's fetch-at-runtime sourcing; no auth needed, plain `GET` on `https://codeberg.org/comaps/comaps/raw/branch/<commit>/data/...` |
| `bufio`/`strings` (stdlib) | Go 1.25 | Line-oriented parsing of `hierarchy.txt` and `.poly` | Both formats are line-oriented plaintext; no need for a parser-combinator library |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled `.poly` parser (D-03, locked) | `github.com/paulmach/orb` + a community `.poly` reader | Rejected in CONTEXT.md — adds a dependency for a ~50-line, one-off maintainer tool |
| Go SDK collection construction | Hand-copied JSON collection blob (matches `1784658610_created_region_archives.go`'s style) | The JSON-blob style is this repo's *existing* convention (auto-generated by PocketBase Admin UI's `migratecmd` Automigrate), but hand-authoring that JSON by hand is error-prone and this agent has no interactive Admin UI session to generate it from. The Go SDK (`core.NewBaseCollection`, `.Fields.Add`) produces an equivalent collection and is easier to hand-verify. Recommend SDK-based; both are valid PocketBase migration styles. |

**Installation:**
No new packages to install — both `pocketbase` and `cobra` are already present in `db/go.mod`.

**Version verification:** Confirmed directly against `db/go.mod`/`db/go.sum` (`pocketbase v0.38.0`, `cobra v1.10.2`) — no registry lookup needed since these are pinned, in-repo, already-vendored versions (present in `$GOMODCACHE`).

## Architecture Patterns

### System Architecture Diagram

```text
Codeberg (comaps/comaps)                    Wanderer repo (committed)
┌─────────────────────────┐                 ┌──────────────────────────┐
│ data/hierarchy.txt       │  HTTP GET       │ db/commands/              │
│ data/borders/*.poly      │ ───(fetch)───►  │   seed_regions.go         │
│ (pinned --commit=<hash>) │                 │   (Cobra command,        │
└─────────────────────────┘                 │    dev-time only)         │
                                              │                          │
                                              │  1. fetch hierarchy.txt  │
                                              │  2. parse indent tree    │
                                              │     -> group/leaf nodes  │
                                              │  3. for each leaf:       │
                                              │     fetch its .poly,     │
                                              │     parse rings,         │
                                              │     emit GeoJSON +bbox   │
                                              │  4. compute path/depth/  │
                                              │     sort_order           │
                                              │  5. write flattened JSON │
                                              └───────────┬──────────────┘
                                                           │ writes
                                                           ▼
                                              db/migrations/initial_data/
                                                  regions_seed.json  (committed)
                                                           │
                                                           │ read at every
                                                           │ instance startup
                                                           ▼
                                              db/migrations/<ts>_create_regions.go
                                              ┌──────────────────────────┐
                                              │ up(app):                 │
                                              │  1. skip if already      │
                                              │     seeded (idempotency) │
                                              │  2. build `regions`      │
                                              │     collection (fields:  │
                                              │     comaps_id, parent    │
                                              │     [self-relation],     │
                                              │     path, depth,         │
                                              │     sort_order, name,    │
                                              │     kind, polygon, bbox, │
                                              │     enabled)             │
                                              │  3. app.RunInTransaction:│
                                              │     bulk-insert every    │
                                              │     seed row             │
                                              └───────────┬──────────────┘
                                                           │ runs via
                                                           │ core.AppMigrations
                                                           │ on every OnServe
                                                           ▼
                                              PocketBase SQLite: `regions` table
                                              (fresh instance, zero admin action)
```

### Recommended Project Structure
```
db/
├── commands/
│   └── seed_regions.go          # NEW - Cobra command, dev-time only
├── migrations/
│   ├── initial_data/
│   │   └── regions_seed.json    # NEW - committed flattened output
│   └── <timestamp>_create_regions_collection.go   # NEW - schema + bulk insert
└── main.go                      # setupCommands() gains one line
```

### Pattern 1: Self-referencing relation field created in one pass
**What:** `core.NewBaseCollection(name)` synchronously assigns `collection.Id` (via its internal `initDefaultId()`) before the collection is ever saved — so a self-referencing `parent` relation field can be built in the same migration, no two-phase create-then-patch needed.
**When to use:** Any new collection that needs a self-relation (adjacency-list parent pointer), which is exactly `regions.parent`.
**Example:**
```go
// Source: verified against $GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0/core/collection_model.go
// (NewBaseCollection calls m.initDefaultId() before returning) and
// core/field_relation.go (RelationField.CollectionId)
collection := core.NewBaseCollection("regions")
// collection.Id is already a valid, stable id here — no Save() needed first.

collection.Fields.Add(
    &core.TextField{Name: "comaps_id", Required: true},
    &core.RelationField{
        Name:         "parent",
        CollectionId: collection.Id, // self-reference
        MaxSelect:    1,
        Required:     false, // nullable for root/top-level countries
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
    &core.JSONField{Name: "polygon", MaxSize: 8 << 20}, // see Pitfall 1 re: default 1MB limit
    &core.JSONField{Name: "bbox", MaxSize: 1 << 10},
    &core.BoolField{Name: "enabled"},
)

collection.AddIndex("idx_regions_comaps_id", true, "comaps_id", "")
collection.AddIndex("idx_regions_parent", false, "parent", "")
collection.AddIndex("idx_regions_path", false, "path", "")

if err := app.Save(collection); err != nil {
    return err
}
```

### Pattern 2: Idempotent bulk-insert migration reading committed JSON
**What:** Follows `1780000005_add_other_category.go`'s established shape (guard-then-insert), scaled to bulk insert, wrapped in a single transaction.
**When to use:** Exactly this phase's SEED-02 requirement.
**Example:**
```go
// Source: pattern adapted from db/migrations/1780000005_add_other_category.go
// (FindFirstRecordByData guard, filesystem-relative committed-file read) +
// core.App.RunInTransaction (verified in
// $GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0/core/app.go:360-363)
func init() {
	m.Register(func(app core.App) error {
		count, err := app.CountRecords("regions")
		if err == nil && count > 0 {
			return nil // already seeded — idempotency guard
		}

		collection, err := app.FindCollectionByNameOrId("regions")
		if err != nil {
			return err
		}

		data, err := os.ReadFile("migrations/initial_data/regions_seed.json")
		if err != nil {
			return fmt.Errorf("read regions seed: %w", err)
		}

		var seed []SeedRow // comaps_id, parent_comaps_id, path, depth, sort_order, name, kind, polygon, bbox
		if err := json.Unmarshal(data, &seed); err != nil {
			return fmt.Errorf("parse regions seed: %w", err)
		}

		return app.RunInTransaction(func(txApp core.App) error {
			idByComapsID := map[string]string{}
			// Two passes: (1) create every record (capturing generated ids),
			// (2) Set `parent` now that every id is known — parent_comaps_id
			// in the seed file can't be resolved to a real PB record id
			// until the referenced row itself has been created.
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
					record.Set("enabled", false) // SEED requirement: never pre-enabled
				}
				if err := txApp.Save(record); err != nil {
					return fmt.Errorf("insert region %s: %w", row.ComapsID, err)
				}
				idByComapsID[row.ComapsID] = record.Id
			}
			for _, row := range seed {
				if row.ParentComapsID == "" {
					continue
				}
				record, err := txApp.FindRecordById("regions", idByComapsID[row.ComapsID])
				if err != nil {
					return err
				}
				record.Set("parent", idByComapsID[row.ParentComapsID])
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
		return app.Delete(collection)
	})
}
```
**Note on ordering:** Seeding depth-first (parent before children) means `idByComapsID` is already populated when a child's parent is resolved, making the second pass (or even a single pass, if the seed JSON is pre-sorted parent-first) potentially unnecessary — but the two-pass approach above is defensive against any future re-ordering of the seed file and costs almost nothing extra at ~1,300 rows.

### Pattern 3: Cobra command registration
**What:** Every custom command is wired into `db/main.go`'s `setupCommands`.
**Example:**
```go
// Source: db/main.go:164-166 (existing, verbatim)
func setupCommands(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(commands.Dedup(app))
	app.RootCmd.AddCommand(commands.SeedRegions()) // NEW — no app param needed (writes a file, not the DB)
}
```
`seed_regions.go`'s constructor should NOT take `*pocketbase.PocketBase` (unlike `Dedup`) — it never touches a live database, only fetches, transforms, and writes a JSON file. Its `cobra.Command.Run` closure needs no `app` reference at all.

### Anti-Patterns to Avoid
- **Vendoring raw CoMaps files into the repo:** D-01 explicitly rejects this — only the tool's JSON *output* is committed, never `hierarchy.txt`/`.poly` themselves.
- **Hand-writing the collection as a giant JSON blob to match the repo's existing migration style:** That style is an *artifact* of PocketBase Admin UI's Automigrate feature, not a hand-authoring convention — reproducing it by hand invites subtle JSON-schema mistakes (wrong field `id`, missing `system`/`hidden` defaults). Use the Go SDK instead (Pattern 1).
- **Skipping the CountRecords idempotency guard:** Without it, a `migrate down` + `migrate up` cycle (common in local dev/testing) would either duplicate ~1,300 rows or hit the unique `comaps_id` index and fail loudly — either way, worse than a clean early return.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Materialized-path tree queries | A custom recursive-CTE-emulating query layer | The `path` text field + PocketBase's own `like`/prefix filter expressions (`path ~ 'germany.%'`) | PocketBase (SQLite backend) has no recursive CTE support (already established in the design note) — prefix-match on a precomputed `path` string is the standard workaround, not a novel invention |
| GeoJSON validity (ring winding) | A full topology-validation library | A ~10-line signed-area check per ring (shoelace formula) before emitting each `Polygon`/`MultiPolygon` | RFC 7946 requires right-hand-rule winding (exterior rings counterexpr-clockwise when viewed lon=x/lat=y, holes clockwise); `.poly` files carry no winding guarantee. A tiny helper, not a library, is the right size for this |
| Bulk-insert transaction handling | Manual `BEGIN`/`COMMIT` SQL | `app.RunInTransaction(func(txApp core.App) error {...})` | Already the SDK's own primitive (verified in `core/app.go`), used elsewhere in this codebase's migrations for related multi-step writes |

**Key insight:** Everything genuinely novel in this phase (the `.poly` parser) is intentionally small per D-03; everything else is either an existing PocketBase SDK primitive or an existing in-repo migration pattern. Resist the urge to introduce new abstractions beyond what's listed here.

## Common Pitfalls

### Pitfall 1: JSON field's default 1MB size cap silently rejects large polygons
**What goes wrong:** `core.JSONField`'s `DefaultJSONFieldMaxSize` is 1MB (`const DefaultJSONFieldMaxSize int64 = 1 << 20`, verified in `core/field_json.go`). A handful of leaf regions have very long, high-vertex-count coastlines (e.g. Norway-like fjord coasts, Indonesian/Philippine archipelago leaves, Fiji's antimeridian split) whose full-precision GeoJSON could plausibly exceed 1MB.
**Why it happens:** The field's `MaxSize` defaults to 0, which the SDK treats as "use the 1MB default" — easy to leave unset when hand-authoring the migration.
**How to avoid:** Explicitly set a larger `MaxSize` on the `polygon` field (e.g. 8–16MB) when building the collection. Coordinate precision reduction/simplification is explicitly out of scope for this phase (not mentioned in CONTEXT.md) — just raise the cap rather than attempting simplification.
**Warning signs:** A migration that runs cleanly for most rows but fails validation partway through the bulk-insert loop on one specific large coastal region.

### Pitfall 2: `hierarchy.txt` has two non-region header lines
**What goes wrong:** The file's first two lines are literally `World` and `WorldCoasts` — no semicolon, no depth indentation, not real geographic regions (they're CoMaps' global low-zoom overview `.mwm` files). A parser that doesn't special-case them will either crash (expecting at least one `;`-delimited field) or silently create two bogus root "regions."
**Why it happens:** Not obvious from a purely truncated read of the file — only visible once you look at the very first lines and count semicolons per line (confirmed directly: `awk -F';' '{print NF}'` on the real file shows exactly 2 lines with `NF==1`, both `World`/`WorldCoasts`).
**How to avoid:** Skip any line containing zero semicolons before attempting to parse it as a region entry.
**Warning signs:** Seed output contains a "World" or "WorldCoasts" top-level node with no `.poly` file to match.

### Pitfall 3: `.poly` filenames contain literal spaces, not underscores
**What goes wrong:** Compound region ids like `Germany_Free State of Bavaria` contain literal space characters in both the `hierarchy.txt`/`countries.txt` id string *and* the corresponding `.poly` filename (`data/borders/Germany_Free State of Bavaria.poly`) — confirmed directly by fetching that exact file. A parser assuming space-to-underscore normalization for filenames will 404.
**Why it happens:** Natural assumption from seeing underscores used as the hierarchy-level separator (`Germany_Bavaria`) — but sub-names themselves (`Free State of Bavaria`, `Regierungsbezirk Freiburg`) keep their internal spaces verbatim.
**How to avoid:** Use the `id` string exactly as it appears (URL-encode only for the HTTP fetch — `%20` for spaces — but keep the raw string, spaces included, as `comaps_id`).
**Warning signs:** Sporadic 404s when fetching `.poly` files for regions with multi-word names.

### Pitfall 4: Ring winding order is unenforced by the source format
**What goes wrong:** GeoJSON (RFC 7946) requires exterior rings wound counterclockwise and interior (hole) rings clockwise (when x=lon, y=lat) for correct rendering/processing by strict consumers. The Osmosis `.poly` format imposes no such rule — a ring's point order reflects whatever the original polygon-extraction tooling emitted.
**Why it happens:** Easy to assume "just emit the points in file order" is sufficient; it produces syntactically valid GeoJSON that some consumers (including possibly `pmtiles extract --region`, per the still-pending Phase 29 spike in `.planning/todos/pending/2026-07-24-comaps-poly-region-extraction-spike.md`) may render or clip incorrectly.
**How to avoid:** Compute the signed area (shoelace formula) for every parsed ring and reverse point order as needed to guarantee outer-CCW/hole-CW before writing GeoJSON — a small, mechanical, well-tested step (~10 lines).
**Warning signs:** Not directly observable at Phase 28's schema/seed level (no rendering happens yet) — but Phase 29's extraction spike is the natural place this would surface as "extracted archive looks wrong at a boundary." Doing it right in Phase 28 avoids re-deriving all seed data later.

### Pitfall 5: Migration idempotency vs. a genuinely fresh instance
**What goes wrong:** Conflating "the migration ran once" (PocketBase's own `_migrations` bookkeeping, which already prevents re-running `up()` on a normal startup) with "the data is still there" (an admin could, in theory, bulk-delete `regions` rows without touching the migrations table). A `CountRecords > 0` guard makes the migration a no-op in that scenario too, which is almost certainly the desired behavior (this migration seeds initial data, it's not meant to "heal" a deliberately emptied catalog) — but be explicit about that call, since it's a deliberate interpretation choice, not enforced by PocketBase.
**Why it happens:** It's tempting to skip the guard entirely, relying solely on PocketBase's migration-tracking table — that's actually sufficient for the "zero admin action on a fresh instance" requirement (SEED-02) on its own. The guard mainly matters for local dev's `migrate down && migrate up` cycles.
**How to avoid:** Keep the `CountRecords` guard for dev/test ergonomics, but don't over-engineer it into a full diff/reconcile mechanism — that's explicitly CATALOG-F01 (deferred to v2).
**Warning signs:** A `migrate down` + `migrate up` cycle during local testing that fails on a duplicate unique `comaps_id` instead of cleanly no-op'ing.

## Code Examples

### `.poly` file structure (verified against real CoMaps data, fetched 2026-07-24)
```
Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg
1
	8.875421E+00	4.765625E+01
	8.875422E+00	4.765629E+01
	...
END
END
```
Source: `https://codeberg.org/comaps/comaps/raw/branch/main/data/borders/Germany_Baden-Wurttemberg_Regierungsbezirk%20Freiburg.poly` `[VERIFIED: fetched and inspected directly]`
- Line 1: a free-text name (not necessarily identical to the filename/id — treat as a comment, not data).
- Line 2: ring header (`1`, `2`, ... or `!1`/`!name` for a hole) — CoMaps uses plain integers as ring names.
- Coordinate lines: **tab-separated**, `lon<TAB>lat`, scientific notation (`8.875421E+00`).
- Each ring closes with a lone `END`; the whole file closes with a final `END`.

### Multi-outer-ring (MultiPolygon) real example — Fiji, split at the antimeridian
```
Fiji
1
	1.764398E+02	-1.216289E+01
	1.772658E+02	-1.193753E+01
	1.800000E+02	-1.550887E+01
	1.800000E+02	-1.980052E+01
	1.759944E+02	-1.970520E+01
	1.764398E+02	-1.216289E+01
END
2
	-1.800000E+02	-1.980052E+01
	-1.800000E+02	-1.550887E+01
	-1.788395E+02	-1.590862E+01
	-1.780239E+02	-1.791490E+01
	-1.778952E+02	-1.970240E+01
	-1.785027E+02	-2.117538E+01
	-1.790168E+02	-2.122739E+01
	-1.800000E+02	-1.980052E+01
END
END
```
Source: `https://codeberg.org/comaps/comaps/raw/branch/main/data/borders/Fiji.poly` `[VERIFIED: fetched and inspected directly]`
Two ring headers (`1`, `2`), neither `!`-prefixed — this is D-04's "multi-part regions" case, output as a GeoJSON `MultiPolygon` with two single-ring polygons, not a `Polygon` with two rings.

### `hierarchy.txt` structure (verified against real CoMaps data, fetched 2026-07-24)
```
World
WorldCoasts
Abkhazia;Q23334;;ab,ru
Afghanistan;Q889;af;ps
...
Germany;Q183;de;de
 Germany_Baden-Wurttemberg;Q985
  Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg;Q2833
  Germany_Baden-Wurttemberg_Regierungsbezirk Karlsruhe;Q8165
 Germany_Berlin;Q64
 Germany_Free State of Bavaria;Q980
  Germany_Free State of Bavaria_Upper Bavaria_Munchen;Q10562-Q1726
```
Source: `https://codeberg.org/comaps/comaps/raw/branch/main/data/hierarchy.txt` `[VERIFIED: fetched and inspected directly, 1308 lines total]`
- Format: `Name;WikidataID[;ISOcode[;languages]]` — field count varies 1–4 (`awk -F';'` on the real file: 2 lines with 1 field [`World`/`WorldCoasts`, no delimiter at all], 1006 with 2, 17 with 3, 283 with 4). Only the first field (`Name`) matters for this phase — it's the same compound id string used in `countries.txt`'s `id` and the `.poly` filename.
- **Depth = exact count of leading space characters, one space per level** (confirmed via `awk` byte-counting the raw file — not tabs, not 2-space indents). Observed depth range: 0, 1, 2 (max 3 levels: e.g. `Germany` → `Germany_Baden-Wurttemberg` → `Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg`).
- **No continent-level grouping exists** — every depth-0 entry is a country (or a special non-country header, see Pitfall 2). The design note's illustrative path example (`europe.germany.thuringia`) is *not* literally accurate — there is no "europe" node; a real path looks like `germany.germany_free_state_of_bavaria.germany_free_state_of_bavaria_upper_bavaria_munchen` (using slugified `comaps_id` segments).

### `countries.txt` group/leaf structure (verified against real CoMaps data, fetched 2026-07-24)
```json
{
  "id": "Germany",
  "g": [
    {
      "id": "Germany_Baden-Wurttemberg",
      "g": [
        {
          "id": "Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg",
          "s": 153618777,
          "sha1_base64": "W9EmSWdLljDLueARsQkgq/IT9oI="
        }
      ]
    },
    { "id": "Germany_Berlin", "s": 227193600, "sha1_base64": "..." }
  ]
}
```
Source: `https://codeberg.org/comaps/comaps/raw/branch/main/data/countries.txt` `[VERIFIED: fetched and parsed directly, top-level `{"id":"Countries","v":260714,"map_series":"2026.06.28","g":[...226 entries...]}`]`
- Presence of a `"g"` array (with no `"s"`/`"sha1_base64"` at that level) = `kind: group`.
- Presence of `"s"`/`"sha1_base64"` (no `"g"`) = `kind: leaf`.
- **This phase does not need `countries.txt` at all** — `hierarchy.txt`'s indentation alone gives the exact same tree shape and `id` strings, and leaf-vs-group can be derived just as reliably from "does this hierarchy.txt line have any indented children below it?" (no children = leaf). Fetching `countries.txt` too would be redundant belt-and-suspenders; note this as a simplification opportunity, not a requirement.

### Reading/writing a JSON field on a `core.Record` (established in-repo idiom)
```go
// Source: db/pluginsystem/installed.go:74, db/routes/waypoint_cluster.go:118
// (record.UnmarshalJSONField), db/migrations/1780000005_add_other_category.go:31
// (record.Set(name, map[string]any{...}) for arbitrary JSON-marshalable values)
record.Set("polygon", geojsonPolygonOrMultiPolygon) // any JSON-marshalable Go value
record.Set("bbox", [4]float64{minLon, minLat, maxLon, maxLat})

var polygon map[string]any
_ = record.UnmarshalJSONField("polygon", &polygon)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Admin hand-authors `region_config.json` with bbox arithmetic | Curated, seeded `regions` catalog toggled via admin UI | This milestone (v1.7), superseding v1.6's `region-catalog-backend-decision-trail.md` design | Removes research/bbox-arithmetic burden per `streamlined-region-definition.md`; this phase is the data-model half of that shift |
| bbox-based `pmtiles extract --bbox=` | polygon-based `pmtiles extract --region=<geojson>` | Phase 29 (not this phase) | This phase only needs to produce *correct* GeoJSON polygons — actual extraction-flag behavior is Phase 29's concern, already flagged for its own spike |

**Deprecated/outdated:** None — `region_config.json` and `db/services/regions/` (`config.go`/`builder.go`/`staleness.go`) remain untouched and functioning through this phase; they are superseded starting Phase 29, not Phase 28.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Display `name` should be derived as the last underscore-delimited segment of the compound `comaps_id` (e.g. `Free State of Bavaria` from `Germany_Free State of Bavaria`) | Code Examples / hierarchy.txt structure note | CONTEXT.md doesn't specify the exact derivation rule; if wrong, the admin tree UI (Phase 30) would show redundant repeated prefixes in leaf names. Low risk — easy to fix in `seed_regions.go` alone, doesn't touch the schema. |
| A2 | `countries.txt` is unnecessary — `hierarchy.txt`'s indentation alone can determine group vs. leaf (any line with indented children below it = group) | Code Examples / countries.txt section | If some CoMaps leaf actually has no `.poly` file for a reason not captured by "has children in hierarchy.txt," skipping `countries.txt`'s explicit `s`/`sha1_base64` marker could misclassify a rare edge case as a leaf when it's actually an unbuildable group. Medium-low risk — recommend the planner keep `countries.txt` as a cross-check/fallback signal even if not load-bearing, cheap enough to fetch both. |
| A3 | No live `!`-prefixed hole ring was found in the small sample of `.poly` files inspected (Fiji, Okinawa, Philippines, Indonesia, Croatia checked) | Common Pitfalls / Code Examples | The hole-handling code path (D-04) would go untested against real data during Phase 28 itself; a bug there might only surface later when a country with an actual enclave/lake hole is processed. Recommend a synthetic unit-test fixture using a hand-crafted `!`-prefixed sample, not solely relying on finding a real one. |

## Open Questions

1. **Exact display-`name` derivation rule for leaf/group rows**
   - What we know: `hierarchy.txt`'s only name-like field is the full compound slug (e.g. `Germany_Free State of Bavaria`), not a separate short display name.
   - What's unclear: Whether the planner should strip only the immediate parent prefix (last `_`-segment) or something more elaborate (e.g. stripping *all* ancestor prefixes cumulatively, which is the same thing at depth 1 but could differ if a name itself contains an ancestor's name as a substring in an unexpected way).
   - Recommendation: Use "text after the last `_` matching the parent's full compound id length" (i.e., strip the exact parent `comaps_id` + `_` prefix, not just up to the last underscore) — safer against names that themselves contain underscores as part of a real place name (rare, but `Regierungsbezirk Stuttgart_Heilbronn`-style already-composed sub-splits exist).

2. **Whether to also fetch/cross-check `countries.txt`'s explicit leaf/group marker (see A2)**
   - What we know: `hierarchy.txt`'s tree shape alone is sufficient in every sample checked.
   - What's unclear: Whether any hierarchy.txt leaf-looking node (no indented children) lacks a real `.poly`/`.mwm` leaf in `countries.txt` (i.e., a dead-end entry that isn't actually downloadable).
   - Recommendation: Fetch `countries.txt` too and cross-validate — cheap (one extra HTTP GET at tool-run time), and turns a silent-misclassification risk into a loud parse-time error if the two sources disagree.

3. **`--commit=<hash>` default value (D-02)**
   - What we know: D-02 locks in a CLI flag with "a sensible default baked in."
   - What's unclear: The specific commit hash to bake in as the default — this research fetched from `branch/main` (the moving HEAD), not a pinned commit, since the exact hash to lock in is an implementation/planning decision, not a research one.
   - Recommendation: The planner should have `seed_regions.go`'s implementation task resolve `main`'s current HEAD commit hash at implementation time (e.g. via Codeberg's API `/repos/comaps/comaps/commits?limit=1`) and bake that concrete hash in as the default, rather than defaulting to the literal string `main` (which would defeat D-02's "reproducible without re-editing source" intent).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/running `seed_regions.go` and the migration | ✓ | 1.25.0 (per `db/go.mod`) | — |
| Network access to `codeberg.org` at tool-run time | `seed_regions.go`'s fetch step (D-01) | ✓ (verified — this research fetched from it directly) | — | — |
| `pocketbase`/`cobra` Go modules | Both pieces | ✓ | v0.38.0 / v1.10.2, already vendored | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None — everything needed is already present.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | This phase has no new API/auth surface — `seed_regions.go` is a maintainer-run dev-time CLI tool, not a served endpoint |
| V3 Session Management | No | Same as above |
| V4 Access Control | No | No new PocketBase collection rules are set beyond the default (this phase doesn't define `listRule`/`viewRule` — Phase 29/30 own the API/admin-UI access-control surface for `regions`) |
| V5 Input Validation | Yes | Every parsed `.poly`/`hierarchy.txt` line is untrusted-at-parse-time text (fetched over HTTP); the Go parser must reject malformed lines (wrong field count, unparseable floats, ring never closed by `END`) rather than panicking or silently emitting corrupt geometry |
| V6 Cryptography | No | No cryptographic operations in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| SSRF via an attacker-controlled `--commit=<hash>` value reaching an unexpected URL | Tampering | `seed_regions.go` is a maintainer-run local CLI tool (not a served endpoint reachable by untrusted input) — the `--commit` flag is operator-supplied, not user/network input, so this is a low-severity theoretical concern rather than a real attack surface. Still: validate the commit hash matches a `^[0-9a-f]{7,40}$` pattern before interpolating it into the fetch URL, defense-in-depth against a copy-paste mistake rather than an actual threat actor |
| Path traversal via a crafted `comaps_id` reaching a filesystem path | Tampering | Not applicable in this phase — unlike `db/services/regions/config.go`'s `regionIDPattern` allow-list (used because that `Region.ID` reaches `filepath.Join` for archive paths), this phase's `comaps_id` never touches a filesystem path; it's a PocketBase text field value only. No allow-list regex needed here (Phase 29's filesystem-path-building code, if any, should apply its own guard at that point, mirroring `IsValidRegionID`) |
| Malformed upstream `.poly`/`hierarchy.txt` causing a parser panic that crashes the maintainer's tool run | Denial of Service (locally, dev-time only) | Return parse errors, never `panic`, from every line-parsing function; log which file/line failed so the maintainer can investigate a genuine upstream format change |

## Sources

### Primary (HIGH confidence)
- `$GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0/core/collection_model.go` — `NewBaseCollection`, `AddIndex` — read directly from the vendored source matching `db/go.mod`'s pinned version
- `$GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0/core/field_relation.go` — `RelationField` struct, `CollectionId` self-reference mechanics
- `$GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0/core/field_json.go` — `JSONField`, `DefaultJSONFieldMaxSize` (1MB)
- `$GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0/core/app.go` — `RunInTransaction` interface method
- `$GOMODCACHE/github.com/pocketbase/pocketbase@v0.38.0/migrations/1640988000_init.go` and `core/migrations_list.go` — confirms `m.Register` == `core.AppMigrations.Register`, run via `RunAppMigrations()` on every startup
- `https://codeberg.org/comaps/comaps/raw/branch/main/data/hierarchy.txt` — fetched and byte-inspected directly (1308 lines, indentation confirmed via `awk`)
- `https://codeberg.org/comaps/comaps/raw/branch/main/data/countries.txt` — fetched and JSON-parsed directly (`{"id":"Countries","v":260714,"map_series":"2026.06.28","g":[226 entries]}`)
- `https://codeberg.org/comaps/comaps/raw/branch/main/data/borders/Germany_Baden-Wurttemberg_Regierungsbezirk%20Freiburg.poly` and `.../Fiji.poly` — fetched and inspected directly (single-ring and multi-ring examples)
- `https://wiki.openstreetmap.org/wiki/Osmosis/Polygon_Filter_File_Format` — authoritative Osmosis `.poly` format spec (ring header, `!` hole prefix, `END` terminators, coordinate order)
- `db/commands/dedup.go`, `db/migrations/1780000005_add_other_category.go`, `db/migrations/1784658610_created_region_archives.go`, `db/main.go`, `db/services/regions/{config,builder}.go` — read directly from this repo

### Secondary (MEDIUM confidence)
- None — every claim above was either read directly from vendored source/this repo, or fetched and inspected directly from the CoMaps upstream repo.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; existing versions confirmed against `db/go.mod`/`go.sum`
- Architecture: HIGH — self-relation and JSON-field mechanics verified directly against the vendored PocketBase v0.38.0 source, not training-data recall
- CoMaps format: HIGH — both `hierarchy.txt` and multiple real `.poly` files were fetched and byte-inspected in this session, not assumed
- Pitfalls: HIGH for 1/2/3/5 (directly observed or verified against source); MEDIUM for 4 (winding-order requirement is documented GeoJSON spec, but its practical necessity for `pmtiles extract --region` specifically is still Phase 29's open spike)

**Research date:** 2026-07-24
**Valid until:** CoMaps hierarchy data changes rarely (per `streamlined-region-definition.md`'s own finding — "manual PRs... rare churn"); the PocketBase SDK behavior is pinned to v0.38.0 in `go.mod` and won't drift until that's bumped. Treat this research as valid until either changes (no fixed expiry — re-verify only if `db/go.mod`'s pocketbase version changes, or if `--commit` is bumped past a CoMaps hierarchy restructuring).
