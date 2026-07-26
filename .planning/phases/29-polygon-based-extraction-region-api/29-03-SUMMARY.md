---
phase: 29-polygon-based-extraction-region-api
plan: 03
subsystem: api
tags: [sveltekit, zod, region-download-proxy, openapi, region-api]

# Dependency graph
requires:
  - phase: 29-polygon-based-extraction-region-api
    plan: 01
    provides: "Relaxed Go regionIDPattern (^[a-z0-9][a-z0-9_.'-]*$ + '..' guard) that region_archives.region_id/download-route ids must match"
  - phase: 29-polygon-based-extraction-region-api
    plan: 02
    provides: "GET /api/v1/regions now emits kind/parent/path/depth/enabled on every row via the Go RegionsList handler"
provides:
  - "SvelteKit RegionIdSchema (both download proxies) relaxed in lockstep with the Go regionIDPattern, accepting '.'/'\'' while rejecting '..' traversal"
  - "GET /api/v1/regions OpenAPI doc documents the new kind/parent/path/depth/enabled hierarchy fields"
affects: [29-04-checkpoint-verification, 31-flutter-settings-hierarchy]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Region-id Zod schema kept byte-identical across both download proxy routes, mirroring the Go regionIDPattern regex + explicit '..'-substring .refine guard"

key-files:
  created: []
  modified:
    - web/src/routes/api/v1/regions/[id]/download/+server.ts
    - web/src/routes/api/v1/regions/[id]/download-dem/+server.ts
    - web/src/routes/api/v1/regions/[id]/download/server.test.ts
    - web/src/routes/api/v1/regions/[id]/download-dem/server.test.ts
    - web/src/routes/api/v1/regions/+server.ts

key-decisions:
  - "RegionIdSchema regex relaxed to ^[a-z0-9][a-z0-9_.'-]*$ plus a .refine((v) => !v.includes('..')) guard in both download proxies, kept byte-identical to each other and to 29-01's Go regionIDPattern"
  - "GET /api/v1/regions swagger doc only touched — handler stays a pure event.locals.pb.send('/regions') pass-through, per the plan's explicit no-handler-change instruction"

patterns-established:
  - "Any future third copy of a region-id validator (e.g. a Phase 31 Flutter regex) must mirror this same allow-list + '..'-refine shape to stay in lockstep across all three independent copies (Go, SvelteKit, Flutter)"

requirements-completed: [EXTRACT-03]

# Metrics
duration: ~15min
completed: 2026-07-26
---

# Phase 29 Plan 3: SvelteKit Region-ID Lockstep & Region API OpenAPI Doc Summary

**Both SvelteKit download proxies' `RegionIdSchema` relaxed to accept real seeded `.`/`'`-containing `regions.path` ids (matching 29-01's Go `regionIDPattern` byte-for-byte) with an explicit `..`-traversal refine, and `GET /api/v1/regions`'s OpenAPI doc updated to document the `kind`/`parent`/`path`/`depth`/`enabled` hierarchy fields 29-02's handler now emits.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-26T18:00:07Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Relaxed `RegionIdSchema` in both `download/+server.ts` and `download-dem/+server.ts` to `^[a-z0-9][a-z0-9_.'-]*$` plus a `.refine((v) => !v.includes('..'), ...)` guard, keeping the two schemas byte-identical and in lockstep with the Go `regionIDPattern` from 29-01
- Added test coverage in both routes' `server.test.ts` proving a `.`-containing seeded path id (`algeria.algeria_central`) is accepted and forwarded to the upstream Go handler; the vector download route's test file additionally proves a `..` id is rejected with a 400 before `event.fetch` is ever called
- Updated the `GET /api/v1/regions` `@swagger` JSDoc response schema to document `kind`, `parent`, `path`, `depth`, and `enabled` — the hierarchy fields 29-02's `RegionsList` handler now emits for every catalog row — while leaving the pure pass-through handler body untouched

## Task Commits

Each task was committed atomically:

1. **Task 1: Relax + traversal-guard RegionIdSchema in both download proxies** - `7f37adc2` (feat)
2. **Task 2: Document hierarchy fields on the GET /api/v1/regions OpenAPI schema** - `2a76c2ce` (docs)

**Plan metadata:** (this commit)

## Files Created/Modified
- `web/src/routes/api/v1/regions/[id]/download/+server.ts` - `RegionIdSchema.id` regex relaxed + `.refine` traversal guard added
- `web/src/routes/api/v1/regions/[id]/download-dem/+server.ts` - same, kept byte-identical to the vector download route
- `web/src/routes/api/v1/regions/[id]/download/server.test.ts` - new `.`-id acceptance test + new `..`-id 400-rejection test
- `web/src/routes/api/v1/regions/[id]/download-dem/server.test.ts` - new `.`-id acceptance test
- `web/src/routes/api/v1/regions/+server.ts` - `@swagger` response `properties` documents `kind`/`parent`/`path`/`depth`/`enabled`; description updated to reflect the regions-table source of truth; handler body unchanged

## Decisions Made
- Kept both download proxies' `RegionIdSchema` byte-identical (same regex source string, same `.refine` guard) rather than sharing a single exported schema, matching the plan's explicit "byte-identical" instruction and the existing pattern of independent per-route Zod schemas in this codebase
- Did not add a `..`-rejection test to the DEM download route's test file (only to the vector route's), per the plan's task text which scoped that specific assertion to "at least one" of the two files

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `npx vitest run` green for both download route test files (7 tests total, including the new `.`-id acceptance and `..`-rejection cases)
- `svelte-check --threshold error` clean for `regions/+server.ts` (0 errors, 0 warnings)
- The SvelteKit and Go region-id validators are now provably in lockstep (identical allow-list character class, identical `..` guard) — closes the Pitfall 1 gap flagged in 29-RESEARCH.md
- End-to-end confirmation that a real `.`-containing seeded region downloads successfully through the full stack (SvelteKit proxy -> Go handler -> filesystem) is deferred to the 29-04 human-verify checkpoint, per this plan's own `<verification>` section — not re-proven by unit tests here
- The Flutter copy of this same region-id regex (`app/lib/util/region_file_path.dart`) remains on the old, unrelaxed pattern — explicitly out of scope for this plan and deferred to Phase 31, per 29-02's carried-forward note

## Self-Check: PASSED

- `web/src/routes/api/v1/regions/[id]/download/+server.ts` contains the relaxed regex and `.refine` guard (verified via `grep -c`) ✓
- `web/src/routes/api/v1/regions/[id]/download-dem/+server.ts` contains the relaxed regex and `.refine` guard (verified via `grep -c`) ✓
- `web/src/routes/api/v1/regions/+server.ts` contains all 5 new hierarchy-field doc keys (verified via `grep -Ec`) ✓
- `git log --oneline -3` shows both task commits (`7f37adc2` feat, `2a76c2ce` docs) ✓
- All task-level acceptance criteria re-verified: `npx vitest run` on both download `server.test.ts` files passes (7/7); `npx svelte-check --threshold error` on `regions/+server.ts` reports 0 errors/warnings

---
*Phase: 29-polygon-based-extraction-region-api*
*Completed: 2026-07-26*
