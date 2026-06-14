---
phase: 01-backend-api
plan: 01
subsystem: api
tags: [sveltekit, valhalla, zod, vitest, polyline]

# Dependency graph
requires: []
provides:
  - "POST /api/v1/valhalla/navigate endpoint — authenticated SvelteKit route"
  - "NavigateRequestSchema Zod schema validating waypoints + costing"
  - "ValhallaManeuver type and maneuvers field on Leg interface"
  - "Flutter-facing NavigateManeuver and NavigateResponse types"
affects: [02-flutter-navigation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Auth gate before try/catch (mirrors search/actor analog)"
    - "Explicit upstream body construction from validated fields (no passthrough)"
    - "Polyline decode + lat/lon flip pattern for shape arrays"
    - "res.ok check before res.json() for upstream 502 handling"

key-files:
  created:
    - "web/src/routes/api/v1/valhalla/navigate/+server.ts"
    - "web/src/routes/api/v1/valhalla/navigate/server.test.ts"
    - "web/src/lib/models/api/valhalla_navigate_schema.ts"
  modified:
    - "web/src/lib/models/valhalla.ts"

key-decisions:
  - "directions_type: 'instructions' not 'maneuvers' — 'maneuvers' omits the human-readable instruction string (Pitfall 2)"
  - "bearing field = bearing_after (direction of travel after turn), reconciling D-06 with API-02"
  - "NavigateWaypoint co-located in schema module; NavigateManeuver/NavigateResponse exported from schema module"
  - "Waypoint cap .max(2000) — generous DoS limit per D-03 (accept all trail points)"

patterns-established:
  - "Multi-leg offset: capture offset = shape.length BEFORE appending leg points, then add to each maneuver.begin_shape_index"
  - "Polyline flip: decodePolyline returns [lng, lat]; push [lat, lng] for D-07 lat-first contract"

requirements-completed:
  - API-01
  - API-02

# Metrics
duration: 6m 8s
completed: 2026-06-12
---

# Phase 01: backend-api Summary

**Authenticated SvelteKit POST /api/v1/valhalla/navigate endpoint transforming Valhalla route response into a Flutter-consumable maneuver+shape contract with 8-test Vitest suite**

## Performance

- **Duration:** 6m 8s
- **Completed:** 2026-06-12
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- New endpoint `POST /api/v1/valhalla/navigate` with auth gate (401), Zod validation (400), upstream call, and 502 guard
- Zod `NavigateRequestSchema` with waypoint bounds, min-2 constraint, costing enum defaulting to `"pedestrian"`
- Extended `valhalla.ts` with `ValhallaManeuver` interface and optional `maneuvers?` field on `Leg`
- 8-test Vitest suite covering happy path, coord flip, schema validation, auth 401, invalid-input 400, upstream 502, multi-leg offset, and missing-maneuvers guard — all 8 pass

## Task Commits

1. **Task 1: Types, schema, RED test suite** - `ac9aab90` (test)
2. **Task 2: Implement navigate endpoint (GREEN)** - `54343134` (feat)
3. **Task 3: Error-path and multi-leg tests (REFACTOR)** - `466ebf8c` (refactor)

## Files Created/Modified
- `web/src/routes/api/v1/valhalla/navigate/+server.ts` — authenticated POST endpoint
- `web/src/routes/api/v1/valhalla/navigate/server.test.ts` — 8-test Vitest suite
- `web/src/lib/models/api/valhalla_navigate_schema.ts` — Zod NavigateRequestSchema
- `web/src/lib/models/valhalla.ts` — ValhallaManeuver interface + Leg.maneuvers field

## Decisions Made
- `directions_type: "instructions"` (not `"maneuvers"`) — Valhalla's `"maneuvers"` value omits the human-readable `instruction` string required by API-02 (Pitfall 2 in RESEARCH.md)
- `bearing` field maps to `bearing_after` (direction of travel after the turn), resolving ambiguity between D-06 and API-02
- Upstream Valhalla request body built explicitly from `{ locations: body.waypoints, costing: body.costing, directions_type: "instructions" }` — never spreads raw client body (T-01-03 mitigation)
- Multi-leg shape offset: `offset = shape.length` captured before appending each leg's points, then added to each maneuver's `begin_shape_index`

## Deviations from Plan
None — plan executed exactly as written.

## Issues Encountered
- Worktree was created from `main` base instead of `feature/app` HEAD (base mismatch in cleanup). Resolved by cherry-picking the 3 agent commits directly onto `feature/app` — no conflicts since agent work is in `web/` and `feature/app`'s unique commits are all in `app/` (Flutter).

## User Setup Required
None — no external service configuration required. `VALHALLA_URL` is already used by existing `/api/v1/valhalla/route`.

## Next Phase Readiness
- `POST /api/v1/valhalla/navigate` is ready for Flutter to call
- Response contract: `{ maneuvers: NavigateManeuver[], shape: [number, number][] }` with each maneuver having `instruction`, `length`, `begin_shape_index`, `bearing`
- Phase 2 (Flutter navigation screen) can begin

## Self-Check: PASSED

---
*Phase: 01-backend-api*
*Completed: 2026-06-12*
