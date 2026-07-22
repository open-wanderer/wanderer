# Phase 22 Plan Check — Region & Package Data Model

**Plans checked:** 22-01-PLAN.md, 22-02-PLAN.md
**Status:** VERIFICATION PASSED (with warnings)

## Design-target confirmation (primary concern)

Both plans correctly target the Phase 21.5 backend-fetch design, not the discarded bundled-asset design:

- No `regions.json` asset, no `rootBundle.loadString` anywhere in either plan.
- `region_catalog_entry.dart` parses one element of the `GET /api/v1/regions` array (bare array, not `{items:[...]}` wrapper) — matches `db/routes/regions_get.go` exactly, field-by-field verified against source: `id`, `name`, `bbox` ([minLon,minLat,maxLon,maxLat], confirmed against `regions_get.go:41`), `status`, `version` (optional), `vector_url`/`vector_size` (optional), `dem_status`/`dem_url`/`dem_size` (optional), `error` (optional).
- `region_repository.dart` calls `ref.watch(apiProvider).get('/regions')` (cookie-authenticated Dio), not an asset loader.
- Plan 01 frontmatter/context loads `db/routes/regions_get.go` directly as ground truth rather than the stale `22-RESEARCH.md`/`22-PATTERNS.md` bundled-asset schema — this is the correct call and the field mapping I independently verified against the Go source matches.

## Requirement Coverage

| Requirement | Plan | Task(s) | Status |
|---|---|---|---|
| REGN-01 (fetch + parse, empty catalog case) | 22-01, 22-02 | 22-01 Task 1; 22-02 Task 1 | Covered |
| REGN-02 (Region entity, explicit-int status, survives restart) | 22-01 | Task 3 | Covered |
| REGN-03 (DownloadedTilePackage, independent vector/DEM) | 22-01 | Task 2, Task 3 | Covered |

All three requirement IDs appear in `requirements:` frontmatter of the plan(s) that implement them. Empty-catalog behavior (an instance with no admin config) is implicitly handled: `parseRegionCatalog([])` returns `[]` and `upsertCatalog([])` flips all existing rows' `inCatalog=false` without erroring — no dedicated empty-catalog test/acceptance line calls this out explicitly, which is a minor gap (see Warnings).

## Context Compliance (22-CONTEXT.md D-01 through D-12)

| Decision | Plan/Task | Verified |
|---|---|---|
| D-01 upsert-by-id, never `removeAll` | 22-02 Task 1 (`applyCatalogEntry`/`box.put`, no `removeAll`) | Yes — acceptance criteria explicitly greps for absence of `removeAll(` |
| D-02 explicit `.code`, never `.index`/`values[index]` | 22-01 Tasks 1-3 | Yes — acceptance criteria grep for absence of `.index` / positional `values[` |
| D-03 typed error, not swallowed | 22-02 Task 1 (`RegionCatalogException`, no bare `catch(_){}`) | Yes |
| D-04/D-05 `catalogStatus` distinct field, `.code`-backed | 22-01 Task 3 | Yes |
| D-06 `lastDownloadedVersion` + `updateAvailable` mapping | 22-01 Task 3 (status getter behavior spec + test) | Yes |
| D-07 no DEM version/staleness concept | 22-01 Task 3 (explicit doc comment, getter only checks vector version) | Yes |
| D-08 `inCatalog` flip, no file/row deletion | 22-02 Task 1 | Yes |
| D-09 no package row until real download begins | Implicit — plan never creates a `DownloadedTilePackageEntity` row during catalog upsert; only `ToOne` targets stay unset | Consistent, not explicitly tested but correctly not contradicted |
| D-10/D-11/D-12 enum shadow pattern, two nullable `ToOne`s, computed getter | 22-01 Task 3 | Yes |

No task references or implements anything from a "Deferred Ideas" section (CONTEXT.md declares none deferred; correct — nothing here is out of scope).

## Task Completeness

All 5 tasks (3 in 22-01, 2 in 22-02) have concrete `<files>`, `<read_first>`, `<action>`, `<verify automated>`, `<acceptance_criteria>`, `<done>`. Actions are unusually specific (literal code shapes, exact field lists, exact grep-able anti-pattern checks) — well above the "implement auth" vagueness bar this dimension screens for.

## Dependency Correctness

`22-02` `depends_on: ["22-01"]`, wave 2; `22-01` `depends_on: []`, wave 1. Acyclic, consistent, no forward references — 22-02 correctly waits for `RegionEntity`/`region_status.dart` from 22-01 before building the repository on top of them.

## Key Links

- `RegionCatalogEntry` → `CatalogStatus` (JsonValue enum decode) — wired in Task 1.
- `RegionEntity` → `DownloadedTilePackageEntity` via `ToOne` — wired in Task 3.
- `RegionEntity.fromCatalogEntry`/`applyCatalogEntry` → `RegionCatalogEntry` — wired in Task 3.
- `RegionRepository` → `apiProvider`/`objectBoxProvider` → `RegionEntity.applyCatalogEntry` — wired across 22-02 Tasks 1-2.

All key_links declared in frontmatter have a concrete implementing action, not just artifact creation in isolation.

## Scope Sanity

- 22-01: 3 tasks, 11 files (incl. 3 generated + 3 test files) — within budget.
- 22-02: 2 tasks, 3 files — within budget.

No plan approaches the 4-5 task warning/blocker threshold.

## must_haves Derivation

Truths are observable/testable ("region can show its vector package downloaded while its DEM package is not", "applyCatalogEntry preserves obxId/links/lastDownloadedVersion") rather than pure implementation trivia, though a few lean toward mechanism-level phrasing (e.g. ".code round-trip") — acceptable here since REGN-02's entire point *is* the persistence mechanism, and the phase has no UI to phrase truths against.

## Dimension 8: Nyquist Compliance

Skipped — no `VALIDATION.md` exists for this phase and `RESEARCH.md` has no "Validation Architecture" section (this is a pre-Nyquist-workflow phase). Not a blocker for this phase given no such gate was established when RESEARCH.md was written; flagging as informational only.

## Dimension 11: Research Resolution — WARNING (stale artifact, not blocking)

`22-RESEARCH.md` has an `## Open Questions` section (not `(RESOLVED)`-suffixed) containing two unresolved questions from the **discarded bundled-asset design** (e.g. "should vector_url/dem_url be identical per region"). These questions are moot under the new backend-fetch design — Phase 21.5's actual API independently answers them (vector and DEM have genuinely distinct URLs/sizes/statuses in the real endpoint). Neither plan cites `22-RESEARCH.md` in its `<context>` or `<read_first>` blocks, so this stale content cannot leak into execution. Recommend: mark the RESEARCH.md Open Questions section `(SUPERSEDED — see 22-CONTEXT.md D-04–D-09)` for future-reader clarity, but this is a documentation-hygiene warning, not an execution risk.

## Dimension 12: Pattern Compliance — WARNING (stale artifact, not blocking)

`22-PATTERNS.md` was mapped against the discarded bundled-asset design and still lists `app/assets/map/regions.json` and `app/lib/models/region_manifest.dart` as files to be created — neither plan creates these. The *core* structural analogs it identifies (`map_cell.dart` for freezed/json_serializable shape, `trail_entity.dart` for dual-id + `ToOne`, `active_navigation_entity.dart` for the enum-shadow structural shape) are still the same analogs both plans correctly cite directly in their own `<read_first>` blocks — so the plans didn't inherit the stale field-level content, only the (still-valid) structural pattern. However, 22-01-PLAN.md's `<context>` block still includes `@.planning/phases/22-region-package-data-model/22-PATTERNS.md` verbatim, which will inject the stale `region_manifest.dart`/`regions.json` guidance into the executor's context alongside the correct `db/routes/regions_get.go` ground truth. Recommend regenerating `22-PATTERNS.md` against the new design before execution, or at minimum removing it from 22-01's `<context>` block, to avoid executor confusion between two contradictory schemas for the same file role.

## Dimension 10: CLAUDE.md Compliance

No conflicts found. CLAUDE.md's conventions section is largely web/TypeScript-focused; the Dart-side conventions it does state (PascalCase types, camelCase functions, boolean `is`-prefix) are followed by both plans.

## Dimension 7c: Architectural Tier Compliance

Skipped — no Architectural Responsibility Map in `22-RESEARCH.md`.

## Dimension 9: Cross-Plan Data Contracts

No conflict — 22-01 defines the entities/model, 22-02 consumes them read/write via `applyCatalogEntry`/`fromCatalogEntry` with no competing transform on the same data.

---

## Issues

```yaml
issues:
  - dimension: pattern_compliance
    severity: warning
    description: "22-PATTERNS.md still describes the discarded bundled-asset design (region_manifest.dart, assets/map/regions.json) and is still referenced in 22-01-PLAN.md's <context> block, risking executor confusion alongside the correct db/routes/regions_get.go ground truth"
    plan: "22-01"
    fix_hint: "Regenerate 22-PATTERNS.md against the backend-fetch design, or remove the @22-PATTERNS.md reference from 22-01-PLAN.md's <context> block"
  - dimension: research_resolution
    severity: warning
    description: "22-RESEARCH.md has an unresolved '## Open Questions' section from the discarded bundled-asset design; questions are moot under the new design and not referenced by either plan, but the section is not marked (RESOLVED) or (SUPERSEDED)"
    file: "22-RESEARCH.md"
    fix_hint: "Mark the Open Questions section '(SUPERSEDED — see 22-CONTEXT.md D-04-D-09)' for documentation hygiene"
  - dimension: requirement_coverage
    severity: info
    description: "No task/test explicitly names the 'empty catalog for a fresh instance' case from REGN-01's exact wording, though the implementation (parseRegionCatalog([]) + orphan-flip-all) correctly handles it as an emergent property of the general logic"
    plan: "22-02"
    fix_hint: "Optional: add an explicit empty-array test case to region_repository_test.dart for documentation/confidence, not required for correctness"
```

## Recommendation

No blockers. Two warnings concern stale planning-phase documentation artifacts (RESEARCH.md/PATTERNS.md left over from the discarded bundled-asset design) rather than the PLAN.md files themselves — the plans correctly bypass the stale content by citing `db/routes/regions_get.go` and the real ObjectBox analog files directly. Recommend cleaning up 22-PATTERNS.md/22-RESEARCH.md before or during execution to prevent executor confusion, but this does not block starting execution of 22-01-PLAN.md / 22-02-PLAN.md.

**Plans verified. Safe to proceed to `/gsd-execute-phase 22`.**
