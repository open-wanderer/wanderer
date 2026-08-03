---
phase: 36-local-first-recording-automatic-upload
plan: 12
subsystem: ui
tags: [riverpod, go_router, flutter, offline-ux]

# Dependency graph
requires:
  - phase: 36-09
    provides: "the .select/skipLoadingOnReload own-trails-list render path this plan's caller (profile_trail_screen.dart) sits inside, unchanged by this plan"
  - phase: 36-11
    provides: "trailDetailLocation/trailMapLocation (trail_route_location.dart), localTrailProvider, and the /trail/local/:localId route with no map/navigate sub-route yet"
provides:
  - "TrailDetailMapScreen dual-mode (id/localId), mirroring 36-11's TrailDetailScreen treatment -- the map sub-route under /trail/local/:localId now resolves"
  - "profile_trail_screen.dart's own-trails list tap handler routed through trailDetailLocation instead of diverting every unsynced trail to /trail/create/edit"
  - "trail_panel.dart's three map-affordance pushes routed through trailMapLocation, disabled (not pushed) when a trail is unaddressable"
  - "profile_trail_screen_navigation_test.dart -- a behavioural (not source-grep) regression gate on the own-trails-list tap destination"
affects: [profile, trail-detail, offline-ux]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A widget test that pumps the real screen inside a real GoRouter and asserts GoRouter.state.uri (not routerDelegate.currentConfiguration.uri, which was observed not to reflect a push made after MaterialApp.router's initial build in this harness) -- the behavioural counterpart to a source-grep gate, exists specifically because a source-grep gate let the original regression ship unnoticed"

key-files:
  created:
    - app/test/routes/profile_trail_screen_navigation_test.dart
  modified:
    - app/lib/routes/trail_detail_map_screen.dart
    - app/lib/provider/router_provider.dart
    - app/lib/routes/profile_trail_screen.dart
    - app/lib/components/trail/trail_panel.dart
    - .planning/phases/36-local-first-recording-automatic-upload/36-12-PLAN.md

key-decisions:
  - "TrailDetailMapScreen's dual-mode build() resolves both localTrailValue and trailAsync up front (one null per branch) so the AppBar title works in both modes without duplicating the trailAsync!.when(...) call -- mirrors 36-11's TrailDetailScreen build()/_buildDetail split exactly"
  - "The floating Navigate button inside _buildMap is wrapped in `if (!isUnsyncedState(trail.syncState))` rather than disabled -- matches 36-11's precedent (hidden, not greyed out) for an affordance whose backing request (Valhalla + nav cache) is keyed on a server id an unsynced trail does not have"
  - "The widget test asserts on `GoRouter.state.uri`, not `router.routerDelegate.currentConfiguration.uri` as the plan's harness prose suggested -- empirically, in this MaterialApp.router(routerConfig:) harness, currentConfiguration.uri stayed pinned to the initial location after a push while GoRouter.state.uri correctly reflected it (verified by a manual router.push() sanity check before writing the real assertions)"

requirements-completed: [REC-02, REC-05]

# Metrics
duration: ~40min
completed: 2026-08-03
---

# Phase 36 Plan 12: Wiring the own-trails list and trail panel to the local-trail route Summary

**Closes the second half of UAT gap 2: `profile_trail_screen.dart`'s own-trails list tap handler and `trail_panel.dart`'s three map-affordance pushes now go through `trailDetailLocation`/`trailMapLocation` instead of interpolating a blank server id, `TrailDetailMapScreen` gained the same dual-mode (id/localId) treatment 36-11 gave the detail screen, and a new widget test pumps the real list inside a real `GoRouter` to assert the actual pushed location -- the exact automated signal UAT found missing.**

## Performance

- **Duration:** ~40 min
- **Tasks:** 3 completed
- **Files modified:** 5 (1 created, 4 modified, one of the 4 a plan-frontmatter regex fix)

## Accomplishments
- `trail_detail_map_screen.dart` is now dual-mode: `TrailDetailMapScreen` takes an optional `localId`, reads `localTrailProvider` instead of `trailProvider` when set, and extracted `_buildMap(context, trail)` renders identically in both modes. The floating Navigate button is hidden (`if (!isUnsyncedState(trail.syncState))`) for an unsynced trail -- `launchNavigation` and the Valhalla/nav-cache path are all keyed on the server id, which is empty in local mode.
- `router_provider.dart`'s `/trail/local/:localId` route gained a `map` sub-route -> `TrailDetailMapScreen(id: '', localId: localId)`, mirroring `/trail/:id`'s own `map` sub-route. No `navigate` sub-route, matching 36-11's recorded rationale on the parent route.
- `profile_trail_screen.dart`'s `_onTrailSelect` no longer branches on sync state at all: it delegates to `trailDetailLocation(trail)` and pushes the result. The `/trail/create/edit` fallback survives only for the (currently unreachable) case where an unsynced trail has no `localId`.
- `trail_panel.dart` computes `mapLocation = trailMapLocation(trail)` once and uses it at all three former `context.push('/trail/${trail.id}/map')` call sites (inline map tap, expand button, elevation-profile tap), each now disabled (`null` callback) rather than pushed when the trail is unaddressable.
- `profile_trail_screen_navigation_test.dart` pumps the real `ProfileTrailScreen` inside a real `GoRouter` with stubbed providers (`_StubAuth`, `_StubTrailSync`, `_StubProfileTrails`, `_StubFilter`) and asserts: tapping an unsynced row lands on `/trail/local/local-1-0` (and explicitly not `/trail/create/edit`), and tapping a synced row still lands on `/trail/server-1`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Dual-mode trail detail map screen and its local route** - `65c46efc` (feat)
2. **Task 2 (RED): failing navigation test** - `87d76a1a` (test)
2. **Task 2 (GREEN): tapping an unsynced trail opens the detail screen** - `890491b3` (feat)
3. **Task 3: Retarget the trail panel's three map pushes** - `93f3dc62` (feat)

Plus one out-of-task-list fix required by the orchestrator's known-defect note:

4. **Plan-frontmatter regex repair** - `4397c440` (fix)

## Files Created/Modified
- `app/lib/routes/trail_detail_map_screen.dart` - Dual-mode `build()`/`_buildMap` split, `localId` param, hidden Navigate for an unsynced trail
- `app/lib/provider/router_provider.dart` - `map` sub-route added under `/trail/local/:localId`
- `app/lib/routes/profile_trail_screen.dart` - `_onTrailSelect` delegates to `trailDetailLocation`
- `app/lib/components/trail/trail_panel.dart` - `mapLocation` computed once, all three map pushes retargeted
- `app/test/routes/profile_trail_screen_navigation_test.dart` - New behavioural navigation gate
- `.planning/phases/36-local-first-recording-automatic-upload/36-12-PLAN.md` - Third `key_links` pattern's regex escaping repaired (see Deviations)

## Decisions Made
- Dual-mode `TrailDetailMapScreen` build() resolves both possible values up front rather than branching deep inside a single expression, so the `AppBar` title logic reads the same regardless of mode.
- Navigate is hidden (not disabled) for an unsynced trail in the map screen, matching `TrailDetailScreen`'s own established chrome-gating precedent from 36-11.
- The navigation test asserts on `GoRouter.state.uri` rather than the plan's suggested `router.routerDelegate.currentConfiguration.uri` -- see Deviations.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - blocking issue, orchestrator-flagged] Repaired the third `key_links` pattern's invalid regex**
- **Found during:** Pre-flight (flagged by the orchestrator before dispatch via `gsd-tools query verify.key-links`)
- **Issue:** The plan's third `key_link` pattern was authored as `"TrailDetailMapScreen\\(id: '', localId:"` (a doubled backslash before the open paren, following ordinary YAML double-quote escaping conventions). `gsd-tools`' `parseMustHavesBlock` extracts `key_links` values with a custom regex-based mini-parser that does **not** perform YAML's own backslash-unescaping pass -- it copies the text between the quotes verbatim. The doubled backslash therefore reached `new RegExp(...)` as two literal backslash characters followed by an unescaped, unclosed `(`, throwing `Invalid regular expression` on every `verify.key-links` run (confirmed via direct `node -e` reproduction against `frontmatter.cjs`'s `parseMustHavesBlock`).
- **Fix:** Rewrote the pattern as `"TrailDetailMapScreen\(id: '', localId: localId\)"` -- a single backslash before each parenthesis, with a matching closing paren added around `localId: localId` so the extracted string is a syntactically valid, balanced regex. Verified via direct `node -e` reproduction of the tool's exact extraction + `new RegExp(...).test(...)` path against the real `router_provider.dart` content: compiles without error and returns `true`.
- **Files modified:** `.planning/phases/36-local-first-recording-automatic-upload/36-12-PLAN.md`
- **Commit:** `4397c440`
- **Note on the other two `key_links`:** `verify.key-links` still reports `trailDetailLocation\\(trail\\)` and `trailMapLocation\\(trail\\)` as "not found in source or target" even after this plan's own edits land the literal text `trailDetailLocation(trail)` / `trailMapLocation(trail)` in their respective files (confirmed by direct `grep`). This is the *same* double-backslash-preserved-verbatim behavior as the defect above, not a new one: those two patterns were authored the same way, so they will structurally never verify via `verify.key-links` regardless of source content, only via the fallback `to`-string-containment path if `pattern` were removed. The task instructions scoped this deviation to repairing only the third (genuinely *invalid*, exception-throwing) pattern -- "do not work around it by weakening what the link actually checks" -- so the first two `key_links` are left as authored. The underlying source changes for both are directly verified via `grep` in this plan's own verify commands (Tasks 2 and 3's automated `<verify>` blocks both passed), so the plan's `must_haves` intent is satisfied even though `verify.key-links`'s automated check for those two specific entries cannot currently confirm it.

### Auto-fixed Issues (deviation, Rule 4 boundary -- documented, not architectural)

**2. [Test-harness correction] `GoRouter.state.uri` used instead of `routerDelegate.currentConfiguration.uri` for the navigation test's location assertion**
- **Found during:** Task 2, writing the widget test
- **Issue:** The plan's harness prose suggested asserting on `router.routerDelegate.currentConfiguration.uri.toString()`. While developing the test, a manual `router.push('/trail/server-1')` sanity check showed `currentConfiguration.uri` remained pinned to the initial `/profile/@tester/trails` location after the push and a `pumpAndSettle()`, while `router.state.uri` correctly reported `/trail/server-1`.
- **Fix:** Used `router.state.uri.toString()` as the `currentLocation()` helper instead. This is GoRouter's own documented "where am I now" accessor and produces the same assertions the plan's must_haves truths require (the router's actual final location, not source text).
- **Files modified:** `app/test/routes/profile_trail_screen_navigation_test.dart`
- **Commit:** `87d76a1a`

## Issues Encountered

None beyond the two items already documented above.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness
- The full unsynced-trail navigation path (own-trails list -> detail screen -> full-screen map) is wired end to end and unit/widget-tested. `flutter analyze --no-pub` reports zero errors project-wide (35 pre-existing info-level issues in unrelated files, same count as 36-10/36-11's summaries). `flutter test` passes 843/843 (841 pre-existing + 2 new).
- Device re-test (this plan's `<verification>` device step, not run by this plan): in airplane mode, tap an unsynced trail in `/profile/<handle>/trails` and confirm the detail screen, its inline map, expand button, and elevation-profile tap all reach the full-screen map with no error page, feeding UAT Test 2 and unblocking UAT Test 3.
- No blockers for subsequent Phase 36 plans.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED

All created/modified files verified present on disk; all five task commit hashes (`65c46efc`, `87d76a1a`, `890491b3`, `93f3dc62`, `4397c440`) verified present in git history.
