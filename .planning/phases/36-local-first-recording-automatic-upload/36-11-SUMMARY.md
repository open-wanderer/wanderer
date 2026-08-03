---
phase: 36-local-first-recording-automatic-upload
plan: 11
subsystem: ui
tags: [riverpod, objectbox, go_router, flutter, l10n]

# Dependency graph
requires:
  - phase: 36-09
    provides: build-tooling ordering only (this plan is the only wave-2 codegen run; 36-09's Task 1 also runs build_runner, and the two cannot run concurrently against the same app/.dart_tool/build lock) -- no semantic dependency
provides:
  - "trailDetailLocation / trailMapLocation (trail_route_location.dart) -- the only sanctioned way to build a trail route path; unsynced trails address by localId, never by the D-06-blanked server id"
  - "readOwnLocalTrail (local_trail_store.dart) -- owner-scoped single-row local-trail lookup for route-parameter input"
  - "localTrailProvider (local_trail_provider.dart) -- synchronous, account-scoped, local-id-validated Trail? read"
  - "/trail/local/:localId route -> dual-mode TrailDetailScreen with unsynced chrome gating (no Download, no Like, Edit instead of Navigate; TrailDropdown mounted unconditionally)"
affects: [profile, trail-detail, offline-ux]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A route-location helper (trailDetailLocation/trailMapLocation) as the single call site every push must go through, returning null to mean 'disable the affordance' rather than pushing an unroutable path"
    - "A synchronous Trail? provider (no AsyncValue) for data that is already on-device and never awaited, distinct from the network-first *_provider.dart family"
    - "Owner-scoped sibling read function alongside an existing unscoped one, for the specific case where the id argument arrives from a route parameter rather than a caller-verified value"

key-files:
  created:
    - app/lib/util/trail_route_location.dart
    - app/lib/provider/trail/local_trail_provider.dart
    - app/lib/provider/trail/local_trail_provider.g.dart
    - app/test/util/trail_route_location_test.dart
    - app/test/provider/trail/local_trail_addressing_gate_test.dart
  modified:
    - app/lib/util/local_trail_store.dart
    - app/lib/provider/router_provider.dart
    - app/lib/routes/trail_detail_screen.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_localizations.dart (+ 13 per-locale files)
    - app/lib/i18n/untranslated_messages.json
    - app/lib/provider/profile/profile_trails_provider.g.dart (build_runner side effect only, no logic change)

key-decisions:
  - "readOwnLocalTrail is a new owner-scoped sibling of the existing unscoped readLocalTrail, not a signature change to it -- readLocalTrail's two existing callers are account-correct by construction, and a route parameter is not (T-36-11-01)"
  - "localTrailProvider is synchronous (Trail?, no AsyncValue) -- the row is already on-device, so there is nothing to await, no AsyncError to auto-retry, and no loading spinner to render, matching 36-09's own-trails-list fix philosophy"
  - "TrailDropdown stays mounted unconditionally in the dual-mode detail screen -- this is the single instantiation site in the app, so D-14/D-17's delete/download-hiding gating becomes reachable for an unsynced trail for the first time, unblocking UAT Test 3"
  - "No /trail/local/:localId/map or /navigate sub-route added -- the map sub-route needs 36-12's TrailDetailMapScreen localId parameter (would not compile here), and navigate needs a Valhalla round trip plus the server-id-keyed nav cache neither of which a local-only trail has"

requirements-completed: [REC-02, REC-05]

# Metrics
duration: ~25min
completed: 2026-08-03
---

# Phase 36 Plan 11: The local-trail read path and the route-location contract Summary

**`/trail/local/<localId>` now resolves to the ordinary trail detail screen for a not-yet-uploaded trail: a new owner-scoped ObjectBox read (`readOwnLocalTrail`) backs a synchronous, account-scoped Riverpod provider (`localTrailProvider`), and `TrailDetailScreen` renders from it in a dual-mode `build()` that hides Download/Like and offers Edit in place of Navigate, mounting `TrailDropdown` for an unsynced trail for the first time.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 2 completed
- **Files modified:** 20 (5 created, 15 modified, one of the 15 a build-tooling side effect)

## Accomplishments
- `trail_route_location.dart`: `trailDetailLocation`/`trailMapLocation`, the single sanctioned way to build a trail route path -- an unsynced trail addresses by `Trail.localId` (`/trail/local/<localId>`), a synced trail by `Trail.id` (`/trail/<id>`), and both return null (never an unroutable `//`-bearing path) when the relevant identity is missing.
- `local_trail_store.dart` gained `readOwnLocalTrail(store, localId:, accountId:)`, an owner-scoped single-row lookup for the case a route parameter (attacker-supplied as far as this layer is concerned) needs to reach the database, distinct from the existing unscoped `readLocalTrail` whose callers are already account-correct by construction.
- `local_trail_provider.dart`'s `localTrailProvider` family is a synchronous `Trail?` read: validates the id shape via `isLocalId`, resolves the signed-in account fresh via `currentAccountId`, and delegates to `readOwnLocalTrail` -- no `AsyncValue`, no spinner, no auto-retry.
- `router_provider.dart` gained `GoRoute(path: '/trail/local/:localId', ...)`, declared immediately before `/trail/:id`, pushing `TrailDetailScreen(id: '', localId: localId)`.
- `trail_detail_screen.dart` is now dual-mode: `build()` branches on `widget.localId`, reading `localTrailProvider` (with a plain "no longer on this device" scaffold on a null read) instead of `trailProvider` when set. The extracted `_buildDetail` gates every server-id-dependent affordance on `isUnsyncedState(trail.syncState)`: `LikeButton` and the Download button are hidden, the bottom bar renders a single full-width Edit button (pushes `/trail/create/edit`, invalidates `localTrailProvider` on return) instead of Navigate, and `TrailDropdown` stays mounted unconditionally -- making D-14/D-17's delete/download-hiding gating reachable for an unsynced trail for the first time (UAT Test 3). `availableOffline`/`isDownloading` now guard on a non-empty trail id so an empty local id can never collide with another empty id in the library/downloading sets.
- l10n: new `trail_not_on_this_device` key added to `app_en.arb`; `flutter gen-l10n` regenerated the full output (14 per-locale files + `untranslated_messages.json`), all committed.

## Task Commits

Each task was committed atomically:

1. **Task 1: The local-trail read path and the route-location contract** - `1c2d2dcb` (test)
2. **Task 2: Route /trail/local/:localId to a dual-mode detail screen with unsynced chrome gating** - `6a02f972` (feat)

## Files Created/Modified
- `app/lib/util/trail_route_location.dart` - `trailDetailLocation`/`trailMapLocation`
- `app/lib/util/local_trail_store.dart` - Added `readOwnLocalTrail`
- `app/lib/provider/trail/local_trail_provider.dart` - `localTrailProvider` family
- `app/lib/provider/trail/local_trail_provider.g.dart` - Generated
- `app/test/util/trail_route_location_test.dart` - Route-location unit coverage
- `app/test/provider/trail/local_trail_addressing_gate_test.dart` - Source-level owner/account-scoping gate
- `app/lib/provider/router_provider.dart` - New `/trail/local/:localId` route
- `app/lib/routes/trail_detail_screen.dart` - Dual-mode `build()`, extracted `_buildDetail`, unsynced chrome gating
- `app/lib/i18n/app_en.arb` + 14 per-locale `app_localizations_*.dart` + `untranslated_messages.json` - `trail_not_on_this_device` key
- `app/lib/provider/profile/profile_trails_provider.g.dart` - Provider hash regenerated as an unavoidable side effect of the whole-package `build_runner` run this plan's Task 1 mandates; no logic change (see Issues Encountered)

## Decisions Made
- `readOwnLocalTrail` added as a new sibling function rather than changing `readLocalTrail`'s signature, keeping `trail_create_screen.dart` (36-10's scope) untouched.
- `localTrailProvider` synchronous by design (`Trail?`, not `AsyncValue<Trail>`) -- the row is already on-device.
- `TrailDropdown` mounted unconditionally for an unsynced trail -- the deliberate point of this plan, per its objective.
- No `map`/`navigate` sub-routes under `/trail/local/:localId` yet -- `map` needs 36-12's `TrailDetailMapScreen` localId parameter; `navigate` needs infrastructure (Valhalla round trip, server-id-keyed nav cache) a local-only trail does not have.

## Deviations from Plan

None - plan executed exactly as written. The plan's own literal action text (route-location functions, `readOwnLocalTrail`, `localTrailProvider`, the new GoRoute, the dual-mode `build()`/`_buildDetail` split, the l10n key) compiled and analyzed cleanly with no adjustments needed.

## Issues Encountered
- `dart run build_runner build --delete-conflicting-outputs` (Task 1's mandated codegen run) regenerated `app/lib/provider/profile/profile_trails_provider.g.dart`'s provider hash constant as a side effect of resolving the whole package's dependency graph -- no logic in that file changed (verified via diff: single hash-string line). This is the exact behavior the plan's `<context>` wave notes predicted ("the `.g.dart` source hashes this plan regenerates will be one revision behind after the wave merges... expected and harmless") and is not a deviation from the plan's own instructions.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `/trail/local/<localId>` is addressable and renders the ordinary detail screen with the correct chrome for an unsynced trail. 36-12-PLAN can now remove `profile_trail_screen.dart`'s `/trail/create/edit` divert and point the own-trails list tap handler at `trailDetailLocation`, and retarget `TrailPanel`'s remaining `/trail/${trail.id}/map` pushes through `trailMapLocation`.
- The behavioural proof that the route is actually reached (own-trails list tap, dropdown menu widget test) is explicitly deferred to 36-12-PLAN/36-13-PLAN per this plan's own `<verification>` section -- this plan's surface has no caller yet.
- `flutter analyze --no-pub` clean for all files this plan touched (35 pre-existing unrelated info-level issues elsewhere, matching 36-10's summary count exactly). `flutter test` green: 841/841 (833 pre-existing + 8 new).
- No blockers for subsequent Phase 36 plans.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED

All created/modified files verified present on disk; both task commit hashes (`1c2d2dcb`, `6a02f972`) verified present in git history.
