---
status: complete
---

# Quick Task 260717-t7q: Add a Settings tab to the Route Planner sheet — Summary

**Completed:** 2026-07-17

## What shipped

A third "Settings" tab on the Route Planner's `RouteAnchorSheet`, hosting a single unified 5-option travel-bucket picker (Hiking / Biking-Hybrid / Biking-Road / Biking-Cross / Biking-Mountain) and the relocated auto-routing toggle. The same picker also replaced the Phase-21 entry-point sheet's 2-card Hike/Bike choice. Picking a bucket now sends real `costing_options` to `/valhalla/route` (previously only `costing: travelProfile` was sent, no options at all), and any bucket switch — including within-`bicycle` sub-type switches — re-resolves every existing segment under the new costing.

## Tasks (all 5 complete, executed sequentially, no worktree isolation)

1. **`53b6b712`** — `RouteTravelBucket` model (`app/lib/util/route_travel_bucket.dart`, new) carrying the exact costing profile + `costing_options` payload per bucket (per-type `cycling_speed`: Hybrid 18 / Road 25 / Cross 20 / Mountain 16; `use_roads` 0.5, `use_hills` 0.5/1, `avoid_bad_surfaces` 0.25, `shortest` always false — verbatim from web's `route_editor.svelte` defaults). Added `categoryForBikeBucket` to `gpx_util.dart` (icon-resolution heuristic, sibling of the existing `categoryForTravelProfile`).
2. **`c9ec6e45`** — Rec B provider refactor: `RouteAnchors` converted to `@Riverpod(keepAlive: true)` with no family argument; `travelProfile` + `costingOptions` moved into `RouteAnchorsState`. New notifier methods `switchProfile`, `resolveAllSegments`, `resetForSession`. `_resolveSegment` now includes `costing_options` in the POST body when set. Profile switches clear undo/redo (fresh baseline, not undoable) per locked CONTEXT decision.
3. **`13f220eb`** — Dropped the `travelProfile` family argument across all 6+ consumers (`route_anchor_layer.dart`, `elevation_tab.dart`, `route_anchor_list_tab.dart`, `route_anchor_sheet.dart`, `planned_gpx_provider.dart`, `route_planner_screen.dart`, `route_planner_handoff_util.dart`, `router_provider.dart`), relocated the auto-routing toggle out of the screen's top-right controls column, wired session seeding, restored whole-app compilation.
4. **`497803f0`** — New `SettingsTab` component (own file, `ConsumerWidget`, no shared `scrollController` per the TabBarView constraint) with the auto-routing switch + 5-option bucket picker; `route_anchor_sheet.dart` bumped to a 3-tab `DefaultTabController`.
5. **`cae4e55c`** — Expanded `travel_profile_sheet.dart`'s entry sheet from 2 cards to 5 bucket options; threaded the picked bucket's `costing_options` into the router/planner entry.

## Verification

- `flutter analyze` (whole app): clean — 0 errors, only pre-existing unrelated info/warnings (icon deprecations, etc.), none in route-planner files.
- `flutter test` across all 6 plan-touched test files (`route_travel_bucket_test.dart`, `gpx_util_test.dart`, `route_anchor_provider_test.dart`, `planned_gpx_provider_test.dart`, `settings_tab_test.dart`, `travel_profile_sheet_test.dart`): **63/63 passing**.
- Plan-checker (pre-execution): PASSED, no blockers, all locked CONTEXT.md decisions confirmed honored in the plan.

## Deviations

None from the plan. The executor's background process was cut off after committing all 5 tasks and before it could write this SUMMARY.md or send its completion signal — the orchestrator verified all commits, ran the full analyze/test suite independently (all green), and wrote this summary retroactively rather than re-running any task.

## Scope boundaries respected (per locked CONTEXT.md)

- Finish handoff (`finishPlanning`) left untouched — still uses `categoryForTravelProfile`'s generic guess, not the picker's exact selection.
- No manual-override sliders, no "Reverse direction" button.
- Web's `route_editor.svelte` "more settings" panel was deliberately NOT ported.

## Deferred / out of scope

- Renaming `app/lib/util/gpx_util.dart`'s `costingForCategory`/`categoryForTravelProfile` (and optionally web's `valhalla_anchor_util.ts`) into a dedicated `valhalla_util.dart` — raised by the user mid-execution, explicitly parked for a follow-up task to avoid colliding with this task's in-flight edits to `gpx_util.dart`.
- WR-02 (decode-failure catch widening in `_resolveSegment`), WR-03 (router's silent `(0,0)`/`pedestrian` fallback on malformed `extra`), and WR-04 (no visual "resolving" state during a profile-switch re-resolve batch) from the code review — real, but lower-severity than the two Critical findings fixed below; left as follow-up items.

## Post-execution fixes (commit `345e3e53`)

On-device testing surfaced a crash the plan's static verification couldn't catch: `RoutePlannerScreen.initState` called `resetForSession` synchronously on the new `keepAlive` `routeAnchorsProvider`, tripping Riverpod's "modify provider while widget tree is building" assertion on every planner entry. Fixed by deferring the reset via `addPostFrameCallback`, gating `build()` on a `_sessionReady` flag so the prior session's state is never painted (zero-flash, one blank frame on mount).

The code-reviewer's two Critical findings were fixed in the same commit:
- **CR-01**: `TrailSourceSelectScreen`'s "Design your own route" card wasn't guarded against its own in-flight `_plannerLoading` state — a double-tap during the up-to-4s geolocation window could push two `RoutePlannerScreen` instances that corrupt the single shared `keepAlive` provider. Added the missing self-guard.
- **CR-02**: `retrySegment`/`insertAnchorOnSegment` used unguarded `firstWhere` on anchor/segment ids sourced from the native map's asynchronously-synced hit-test layer — a stale tap could throw an uncaught `StateError`. Switched to `firstWhereOrNull` + early-return.

Also fixed the cheap **WR-01**: re-tapping the already-active Settings-tab bucket no longer wipes undo/redo and re-resolves every segment for no reason.

All fixes verified: `flutter analyze` clean (0 errors), 63/63 plan-scoped tests + full suite (180/184, same 4 pre-existing unrelated failures) still passing.
