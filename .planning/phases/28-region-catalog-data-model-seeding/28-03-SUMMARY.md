---
phase: 28-region-catalog-data-model-seeding
plan: 03
subsystem: database
tags: [pocketbase, go-sdk, migration, self-relation, materialized-path, comaps]

# Dependency graph
requires:
  - phase: 28-02
    provides: "db/migrations/initial_data/regions_seed.json (committed 1306-row flattened CoMaps catalog) + SeedRow field tags"
provides:
  - "regions PocketBase collection (comaps_id, self-referencing parent, path, depth, sort_order, name, kind, polygon, bbox, enabled)"
  - "Auto-run migration that bulk-inserts the full committed CoMaps catalog on every fresh instance boot, zero admin action (SEED-02)"
  - "idx_regions_comaps_id (non-unique lookup), idx_regions_path (unique — the true global key), idx_regions_parent"
affects: [29-polygon-extraction-region-api, 30-admin-region-picker-ui, 31-flutter-settings-hierarchy]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Self-referencing PocketBase RelationField requires two Save() passes (create without the self-relation, then add it) — RelationField validation resolves CollectionId via FindCachedCollectionByNameOrId, which cannot see a collection still being created in the same Save() call"
    - "NumberField.Required means 'value is non-zero' in this PocketBase version, not 'value was provided' — never set Required:true on a NumberField whose valid domain includes 0"
    - "Materialized `path` (not a natural/business id like comaps_id) is the correct globally-unique join key for hierarchical seed data sourced from a real-world taxonomy — business ids can legitimately collide (disputed/shared territories) even when the tree position is always unique"

key-files:
  created:
    - db/migrations/1785000000_create_regions_collection.go
  modified: []

key-decisions:
  - "idx_regions_comaps_id changed from unique (per RESEARCH.md/PATTERNS.md's original spec) to non-unique, and idx_regions_path made unique instead — a live migrate-up smoke test against the real committed regions_seed.json found comaps_id is NOT globally unique: 5 real disputed/shared-territory leaves (Abkhazia, South Ossetia, Jerusalem, Crimea, Campo de Hielo Sur) legitimately appear twice, once under each disputing parent (e.g. Jerusalem under both 'Israel Region' and 'Palestine Region'; Crimea under both 'Russian Federation' and 'Ukraine'). `path` (parent-slug-prefixed) was verified unique across all 1306 rows and carries the uniqueness guarantee instead. None of the 5 duplicated names are ever used as a parent_comaps_id (all are leaves), so parent-link resolution correctness for every other row is unaffected."
  - "Parent-link resolution in the bulk-insert's second pass keys off a row's materialized `path` (stripping the last '.'-segment to find the parent's path), not off parent_comaps_id-to-comaps_id lookup as RESEARCH.md's Pattern 2 originally specified — same root cause as the index change: comaps_id collisions would make a flat comaps_id-keyed map ambiguous/lossy for the 5 duplicated names."
  - "Self-referencing parent RelationField built via a genuine two-pass create-then-patch (Save() without parent, then Fields.Add(parent) + Save() again), not the single-pass 'collection.Id is valid immediately' approach RESEARCH.md's Pattern 1 recommended — a live migrate-up test against real PocketBase v0.38.0 showed RelationField validation calls app.FindCachedCollectionByNameOrId(CollectionId), which only resolves collections already saved/cached in the app, not one still being constructed in the same Save() call."
  - "depth and sort_order NumberFields have Required omitted (not Required: true as literally written in the plan's action text) — PocketBase's NumberField.Required semantically means 'non-zero', and depth=0 (every top-level country) and sort_order=0 (first sibling in any group) are both legitimate, common values that a live migrate-up test confirmed would otherwise be rejected as blank."

requirements-completed: [CATALOG-01, CATALOG-02, CATALOG-03, SEED-02]

# Metrics
duration: ~55min
completed: 2026-07-25
---

# Phase 28 Plan 03: regions Collection Migration + Bulk-Seed Summary

**A single PocketBase migration that creates the `regions` collection (self-referencing hierarchy, leaf-only polygon/bbox/enabled) via the Go SDK and, inside one transaction, bulk-inserts all 1306 committed CoMaps catalog rows on every fresh instance boot — verified against a real `migrate up` run, not just a compile check.**

## Performance

- **Duration:** ~55 min (dominated by standing up a local Meilisearch instance to unblock a full `migrate up` smoke test, and three real bugs found only by actually running it)
- **Completed:** 2026-07-25
- **Tasks:** 2/2 completed
- **Files modified:** 1 (new)

## Accomplishments
- `db/migrations/1785000000_create_regions_collection.go` creates the `regions` collection with all ten CATALOG-01/02/03 fields (`comaps_id`, self-referencing `parent`, `path`, `depth`, `sort_order`, `name`, `kind`, `polygon`, `bbox`, `enabled`) via the Go SDK (`core.NewBaseCollection` + `.Fields.Add`), not a hand-authored JSON blob.
- The same migration bulk-inserts every row from the committed `regions_seed.json` inside `app.RunInTransaction`, resolving parent links via a two-pass insert-then-link and setting `enabled=false`/`polygon`/`bbox` only for `kind=="leaf"` rows (CATALOG-03).
- A `CountRecords("regions") > 0` idempotency guard makes a second `up()` (after a `migrate down`+`up` cycle) a clean no-op.
- **Actually ran the migration** against a real, fresh throwaway `pb_data` (via `go run . migrate up --dir /tmp/regions_migrate_test`, with a locally-spun-up Meilisearch instance to satisfy an unrelated pre-existing migration's dependency) and wrote a SQLite verification script asserting: 1306 rows total, at least one leaf row with a populated `polygon`/`bbox` and `enabled=0`, at least one group row with no `polygon`/`bbox`, and zero orphaned `parent` references (1082 rows resolved a parent link, 224 top-level rows correctly have none). Also exercised a genuine `migrate down` + `migrate up` cycle to confirm the idempotency guard and re-seeding both work.
- This process surfaced and fixed **three real bugs** that a compile-only verification would have missed entirely (see Deviations).

## Task Commits

Each task was committed atomically:

1. **Task 1: regions collection schema + down-migration (Go SDK)** - `e445e5e4` (feat)
2. **Task 2: idempotent two-pass bulk-insert from regions_seed.json + fresh-boot verification** - `8faed6d9` (feat)

## Files Created/Modified
- `db/migrations/1785000000_create_regions_collection.go` — the `regions` collection schema (self-relation, indexes, leaf-only geometry fields) and the bulk-insert-with-parent-linking `up()`/`down()` migration

## Decisions Made
See `key-decisions` above. Summary: three real defects in the plan's literal design (verified only against source code and PocketBase internals, not by actually running a migration) were found via a genuine `migrate up` smoke test and fixed:
1. Self-referencing `RelationField` needs a two-pass create-then-patch, not a single Save() — `checkCollectionId` validation resolves the target collection via `FindCachedCollectionByNameOrId`, which cannot see a collection still being created in the same call.
2. `NumberField.Required` means "non-zero" in this PocketBase version — `depth`/`sort_order` had to drop `Required: true` since 0 is a legitimate value for both.
3. `comaps_id` is not globally unique in real CoMaps data (5 disputed-territory leaves appear under two different parents) — the unique index and the parent-resolution join key both moved from `comaps_id` to the provably-unique materialized `path`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Self-referencing RelationField single-pass construction doesn't work against real PocketBase v0.38.0**
- **Found during:** Task 2 (first `migrate up` smoke test)
- **Issue:** RESEARCH.md's Pattern 1 states `core.NewBaseCollection` assigns `collection.Id` synchronously so a self-referencing `parent` field can be added in the same `Fields.Add(...)` call before the collection is ever saved. A live `migrate up` run against a real throwaway `pb_data` failed with `fields: (2: (collectionId: The relation collection doesn't exist..).)`. Reading PocketBase's `core/field_relation.go` `checkCollectionId` directly showed it resolves `CollectionId` via `app.FindCachedCollectionByNameOrId(v)`, which only finds collections already saved/cached in the running app — a collection still being constructed in the same `Save()` call is invisible to it, even though its `.Id` field already holds a valid value.
- **Fix:** Split collection construction into two `Save()` passes: first save every field except `parent`, then add the `RelationField{CollectionId: collection.Id}` and save again (the collection now genuinely exists and resolves).
- **Files modified:** `db/migrations/1785000000_create_regions_collection.go`
- **Verification:** `go run . migrate up --dir /tmp/regions_migrate_test` completed with all 29 migrations applied cleanly; SQLite check confirmed the `parent` field and self-relation work correctly (1082 rows with resolved parent links).
- **Committed in:** `8faed6d9` (Task 2 commit)

**2. [Rule 1 - Bug] `depth`/`sort_order` NumberFields wrongly marked `Required: true`**
- **Found during:** Task 2 (same smoke test, next failure after fixing #1)
- **Issue:** After fixing the self-relation, insertion failed with `depth: cannot be blank; sort_order: cannot be blank` for `Abkhazia` (depth=0, the first row processed). Reading `core/field_number.go` confirmed `NumberField.Required` is documented as "will require the field value to be non-zero" — not "must be present". Every top-level region (depth=0) and every first-in-group sibling (sort_order=0) would fail this check.
- **Fix:** Dropped `Required: true` from both `depth` and `sort_order` `NumberField` definitions (documented inline with the PocketBase-specific reasoning).
- **Files modified:** `db/migrations/1785000000_create_regions_collection.go`
- **Verification:** Re-ran `migrate up`; insertion proceeded past this point.
- **Committed in:** `8faed6d9` (Task 2 commit)

**3. [Rule 1 - Bug] `comaps_id` is not globally unique in real CoMaps data — unique index and parent-resolution join key both needed to change**
- **Found during:** Task 2 (same smoke test, next failure after fixing #1 and #2)
- **Issue:** Insertion then failed with `insert region Campo de Hielo Sur: comaps_id: Value must be unique.` Inspecting the committed `regions_seed.json` directly (not assumed) found 5 real disputed/shared-territory leaves that CoMaps' own hierarchy.txt legitimately lists twice, once under each disputing parent: Abkhazia (top-level AND under "Georgia Region"), South Ossetia (same pattern), Jerusalem (under both "Israel Region" and "Palestine Region"), Crimea (under both "Russian Federation" and "Ukraine"), and Campo de Hielo Sur (under both "Argentina" and "Chile"). Verified separately that the seed's `path` field is 100% unique across all 1306 rows (parent-slug-prefixed materialized path), while `comaps_id` has exactly these 5 duplicates. None of the 5 duplicated names are ever used as a `parent_comaps_id` anywhere in the seed (all are leaves — no children), so this doesn't create any ambiguity in resolving *other* rows' parent links, but it does hard-block the physical insert under a unique index and would have silently overwritten one entry's id in a `comaps_id`-keyed lookup map.
- **Fix:** Changed `idx_regions_comaps_id` from unique to non-unique (kept as a plain lookup index) and made `idx_regions_path` unique instead. Reworked the bulk-insert's parent-resolution pass to key its lookup map by `path` (not `comaps_id`), deriving each row's parent path by stripping its own path's last `.`-delimited segment, rather than joining `parent_comaps_id` against a `comaps_id`-keyed map.
- **Files modified:** `db/migrations/1785000000_create_regions_collection.go`
- **Verification:** Full `migrate up` completed cleanly (29/29 migrations applied); SQLite verification script confirmed 1306 rows, correct leaf/group polygon/bbox/enabled semantics, and zero orphaned parent references. Re-ran a full `migrate down`+`migrate up` cycle to confirm the idempotency guard and re-seed both work correctly with the new index shape.
- **Committed in:** `8faed6d9` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 — bugs in the plan's literal design that only a real `migrate up` run against real data could surface; none were visible from source-reading or `go build`/`go vet` alone)
**Impact on plan:** All three fixes were necessary for SEED-02's core guarantee ("a fresh instance boots with zero admin action and querying regions returns the full CoMaps group/leaf hierarchy") to actually hold. Without them, the migration would fail outright on every fresh boot. No scope creep — the fixes stay entirely within the single migration file, and CATALOG-01/02/03's field list and semantics are unchanged from the plan's intent (only the uniqueness key and the self-relation construction mechanics changed).

## Issues Encountered
- The sandboxed execution environment had no running Meilisearch instance, and a full `migrate up` run against this repo's `db/main.go` migration chain hits a pre-existing, unrelated migration (`1742167033_init_meilisearch.go`) that requires a live Meilisearch server. Resolved by downloading and running a local Meilisearch binary (via the official install script) bound to a scratch port, purely to unblock verification of this plan's own migration — not a defect in this plan's work, and no repo files were changed to work around it.
- Docker was installed but its daemon wasn't running in this environment, so the initial plan to spin up Meilisearch via `docker run` was abandoned in favor of the official standalone binary installer.

## User Setup Required

None — no external service configuration required. The migration is fully automatic on instance startup, per SEED-02.

## Next Phase Readiness
- The `regions` collection exists, fully seeded, hierarchical, with correct leaf/group/enabled semantics — ready for Phase 29 (polygon-based extraction + region API) and Phase 30 (admin region picker UI), both of which depend only on this table.
- `path`, not `comaps_id`, should be treated as the canonical unique/joinable key for any future code reading this table (e.g. Phase 29's extraction cron, Phase 30's admin tree) — `comaps_id` remains useful as a human-readable/display identifier but is not guaranteed unique.
- No blockers.

---
*Phase: 28-region-catalog-data-model-seeding*
*Completed: 2026-07-25*

## Self-Check: PASSED

All created files and commit hashes verified present on disk / in git log.
