---
task: 260717-t7q-add-a-settings-tab-to-the-route-planner-
verified: 2026-07-17T20:23:38Z
status: human_needed
score: 6/6 must-haves verified
overrides_applied: 0
---

# Quick Task 260717-t7q: Add a Settings tab to the Route Planner — Verification Report

**Task Goal:** Add a Settings tab to the Route Planner sheet: consolidated Valhalla travel-profile picker (category-based, sensible defaults, no manual overrides), relocated auto-routing toggle.
**Verified:** 2026-07-17T20:23:38Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | The route planner sheet has a third Settings tab hosting the auto-routing toggle and a 5-option travel-profile picker | ✓ VERIFIED | `route_anchor_sheet.dart:58-157` — `DefaultTabController(length: 3)`, 3rd `Tab(text: 'Settings')`, `SettingsTab()` as 3rd `TabBarView` child. `settings_tab.dart` renders `SwitchListTile` (auto-routing) + a `for (bucket in RouteTravelBucket.values)` loop generating exactly 5 `_BucketCard`s (Hiking, Biking/Hybrid, Biking/Mountain, Biking/Cross, Biking/Road labels confirmed in `route_travel_bucket.dart:17-30`). Widget test `settings_tab_test.dart` asserts all 5 labels + one `SwitchListTile` present, and passes. |
| 2 | Selecting any bucket (including a within-bicycle sub-type switch) re-resolves every existing route segment under that bucket's costing_options | ✓ VERIFIED | `route_anchor_provider.dart:243-284` — `switchProfile` sets `travelProfile`+`costingOptions` then unconditionally calls `resolveAllSegments()`, which iterates every consecutive anchor pair with an existing segment and either re-dispatches `_resolveSegment` (auto-routing on) or reapplies a straight polyline (auto-routing off) — no distinction between cross-family and within-bicycle switches. Provider test group "Rec B: switchProfile / resolveAllSegments / resetForSession" (4 tests: off/on/undo-clear/reset) all pass, including "switchProfile with auto-routing on re-dispatches a Valhalla resolve for every consecutive-anchor segment" (asserts `callCount == 2` for a 3-anchor/2-segment fixture). |
| 3 | The POST /valhalla/route request body carries a nested costing_options object for the selected bucket | ✓ VERIFIED | `route_anchor_provider.dart:130-146` — request `data` map includes `if (state.costingOptions != null) 'costing_options': {state.travelProfile: state.costingOptions}` alongside the existing `'costing': state.travelProfile`. Matches RESEARCH.md Q1's documented web nesting (`costing_options: { [modeOfTransport]: <optionsObject> }`) exactly. Provider tests "includes costing_options... when non-null" and "omits costing_options... when null (back-compat)" both pass. |
| 4 | The entry-point sheet offers the same 5 options and seeds the planner with the chosen bucket's travel profile plus costing_options | ✓ VERIFIED | `travel_profile_sheet.dart:88-96` — `for (bucket in RouteTravelBucket.values)` generates 5 `_TravelProfileCard`s, `onTap: () => Navigator.pop(context, bucket)`. `trail_source_select_screen.dart:42-55` — `_openPlanner` awaits `showTravelProfileSheet`, pushes `/route-planner` with `extra: {'travelProfile': bucket.costing, 'costingOptions': bucket.costingOptions, ...}`. `router_provider.dart:255-264` reads `extra['costingOptions']` and forwards as `initialCostingOptions`. `route_planner_screen.dart:102-112` `initState` calls `resetForSession(widget.travelProfile, widget.initialCostingOptions)` before first build. Widget test `travel_profile_sheet_test.dart` (3 tests: renders 5 labels, tap Mountain resolves `bikingMountain`, dismiss resolves null) all pass. |
| 5 | The auto-routing toggle no longer appears in the top-right map-controls column; it lives only in the Settings tab | ✓ VERIFIED | `route_planner_screen.dart:276-283` — top-right `Positioned` column now hosts only `_buildUndoButton`/`_buildRedoButton`; `_buildAutoRoutingToggle` method is absent from the file (`grep -c _buildAutoRoutingToggle` = 0). `settings_tab.dart:65-73` hosts the only `SwitchListTile` wired to `toggleAutoRouting()`. |
| 6 | Switching profiles establishes a fresh undo baseline (not undoable) and never migrates anchors across a family key | ✓ VERIFIED | `route_anchor_provider.dart:243-251` — `switchProfile` clears `undoStack`/`redoStack` to `const []` via `copyWith`, does not call `_pushUndo()`. `RouteAnchors` is `@Riverpod(keepAlive: true)` with `build()` taking no arguments (`route_anchor_provider.dart:73-92`) — no family key exists, so no anchor migration is structurally possible; anchors stay in the single provider instance across any bucket switch. Provider test "switchProfile clears both undoStack and redoStack... and does not push a new undo snapshot" passes. |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/route_travel_bucket.dart` | 5 fixed travel buckets with exact costing_options payloads, keyword sets, fallback icons, current-selection resolver | ✓ VERIFIED | `enum RouteTravelBucket` present; all 5 payloads match RESEARCH.md Q1 verbatim (cycling_speed 18/25/20/16, use_roads 0.5, use_hills 0.5/1, avoid_bad_surfaces 0.25, shortest false); `bucketForState` resolver present and unit-tested (8 tests pass). |
| `app/lib/util/gpx_util.dart` | `categoryForBikeBucket` icon-resolution heuristic co-located with existing heuristics | ✓ VERIFIED | Function present at line 77, sibling of `categoryForTravelProfile`; doc comment explicitly notes it is icon-resolution only, never a costing source. 4 unit tests pass. |
| `app/lib/provider/route_anchor_provider.dart` | Rec B single keepAlive provider (no family arg) with costingOptions in state and switchProfile/resolveAllSegments/resetForSession | ✓ VERIFIED | `@Riverpod(keepAlive: true) class RouteAnchors extends _$RouteAnchors { RouteAnchorsState build() }` — no-arg. `costingOptions` field in `RouteAnchorsState`. All 3 methods present and implemented as specified. |
| `app/lib/components/route_planner/settings_tab.dart` | Settings tab widget: auto-routing SwitchListTile + 5-option bucket picker | ✓ VERIFIED | `class SettingsTab extends ConsumerWidget`, no constructor params, own `SingleChildScrollView` (no shared scrollController), `SwitchListTile` + 5 `_BucketCard`s wired to `switchProfile`/`toggleAutoRouting`. |
| `app/lib/components/route_planner/route_anchor_sheet.dart` | Three-tab host (Route Anchors / Elevation / Settings) | ✓ VERIFIED | `DefaultTabController(length: 3)`, 3 `Tab`s, 3 `TabBarView` children; shared `scrollController` passed ONLY to `RouteAnchorListTab`. |
| `app/lib/components/route_planner/travel_profile_sheet.dart` | Entry sheet expanded to 5 bucket cards, returning the selected RouteTravelBucket | ✓ VERIFIED | `Future<RouteTravelBucket?> showTravelProfileSheet(...)`, 5-card loop over `RouteTravelBucket.values`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `settings_tab.dart` | `routeAnchorsProvider.notifier.switchProfile` | bucket card onTap | ✓ WIRED | `onTap: () => notifier.switchProfile(bucket.costing, bucket.costingOptions)` (line 87-88). Confirmed by widget test tapping "Biking / Road" and asserting resulting state. |
| `route_anchor_provider.dart` | POST /valhalla/route body | state.costingOptions nested under state.travelProfile | ✓ WIRED | `'costing_options': {state.travelProfile: state.costingOptions}` conditionally added to POST body (line 142-145). Confirmed by 2 dedicated provider tests. |
| `travel_profile_sheet.dart` | `route_planner_screen.dart initialCostingOptions` | trail_source_select_screen -> router_provider extra['costingOptions'] -> resetForSession | ✓ WIRED | Full chain traced: sheet returns bucket -> `trail_source_select_screen.dart` pushes `extra['costingOptions']` -> `router_provider.dart` reads it and passes `initialCostingOptions:` -> `route_planner_screen.dart initState` calls `resetForSession(widget.travelProfile, widget.initialCostingOptions)`. |
| `route_anchor_sheet.dart` | `SettingsTab` | third TabBarView child WITHOUT shared scrollController | ✓ WIRED | `SettingsTab()` constructed with no arguments as the 3rd `TabBarView` child (line 147); `RouteAnchorListTab(scrollController: scrollController)` is the only tab receiving the controller, matching flutter#55388 mitigation documented in both the sheet's and `SettingsTab`'s doc comments. |

### CONTEXT.md Locked Decisions — Cross-Check

| Decision | Status | Evidence |
|----------|--------|----------|
| Rec B architecture, no family key | ✓ VERIFIED | `@Riverpod(keepAlive: true)` with no-arg `build()`; `routeAnchorsProvider` used with no arguments everywhere (`grep -rnE 'routeAnchorsProvider\(\|plannedGpxProvider\('` in `lib` finds no argument-carrying calls outside `.g.dart`). |
| Full re-resolve of ALL segments on ANY bucket switch (not just cross-profile) | ✓ VERIFIED | `switchProfile` always calls `resolveAllSegments()` unconditionally — no branch distinguishing within-bicycle vs cross-profile switches. |
| Unified 5-option picker in both surfaces | ✓ VERIFIED | Both `settings_tab.dart` and `travel_profile_sheet.dart` iterate `RouteTravelBucket.values` (single source of truth), rendering identical 5 labels/descriptions. |
| Fresh undo baseline on switch (not undoable) | ✓ VERIFIED | `switchProfile` clears `undoStack`/`redoStack`, never calls `_pushUndo()`. |
| Finish handoff untouched | ✓ VERIFIED | `route_planner_handoff_util.dart:107,124` — `finishPlanning` still reads `travelProfile` from state and calls unchanged `categoryForTravelProfile(travelProfile, categories)`; no reference to `RouteTravelBucket` or the picker's selection anywhere in this file. |
| No manual-override sliders | ✓ VERIFIED | `grep -rn "Slider"` across `lib/components/route_planner` and `route_planner_screen.dart` returns no matches; `costingOptions` maps are `const` (compile-time fixed), no user-adjustable fields in `SettingsTab`/`travel_profile_sheet.dart`. |
| No Reverse Direction button | ✓ VERIFIED | No "reverse" UI action found in any touched file (only an unrelated `reverse_geocode_util.dart` import in `route_anchor_list_tab.dart`, pre-existing and unrelated). |
| Exact costing_options values | ✓ VERIFIED | All 5 payloads in `route_travel_bucket.dart` match RESEARCH.md Q1's table verbatim (field names, per-type cycling_speed, use_roads/use_hills/avoid_bad_surfaces constants, shortest always false, int `1` vs double `0.5` literal distinction preserved). |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `flutter analyze` clean across whole app | `cd app && flutter analyze` | 47 pre-existing info/warnings (icon deprecations, unrelated files), 0 errors, none in route-planner files | ✓ PASS |
| Full 6-file test suite for this task passes | `cd app && flutter test test/util/route_travel_bucket_test.dart test/util/gpx_util_test.dart test/provider/route_anchor_provider_test.dart test/provider/planned_gpx_provider_test.dart test/components/route_planner/settings_tab_test.dart test/components/route_planner/travel_profile_sheet_test.dart` | 62/62 passed | ✓ PASS |
| No argument-carrying provider calls remain | `grep -rnE 'routeAnchorsProvider\(\|plannedGpxProvider\(' lib --include=*.dart \| grep -v '.g.dart'` | No matches | ✓ PASS |
| Full app test suite (regression check, not scoped to this task) | `cd app && flutter test` | 176/180 passed — 4 pre-existing failures in `feed_item_test.dart`, `settings_screen_test.dart`, `settings_account_screen_test.dart` (unrelated to route planner; none of these files appear in any of the 5 task commits' diffs) | ℹ️ INFO — pre-existing, not introduced by this task |

### Anti-Patterns Found

None. No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any of the 10 touched source files. No empty implementations, no hardcoded-empty stub returns, no console-log-only handlers.

### Requirements Coverage

Not applicable — this is a quick task (`QUICK-260717-t7q` is a self-referential ID for this task directory, not present in `.planning/REQUIREMENTS.md`).

### Human Verification Required

Automated checks (widget tests, provider tests, `flutter analyze`) confirm the wiring and logic are correct, but the following require a human to actually run the app on a device/simulator, since they involve real-time map/network behavior and visual/gesture UX that cannot be verified by static analysis:

### 1. Settings tab scroll behavior inside the live TabBarView

**Test:** Open the route planner with >=1 anchor, open the Settings tab, and swipe/scroll within it while the Route Anchors tab is also populated with enough items to scroll.
**Expected:** No `"ScrollController attached to multiple scroll views"` runtime exception; the Settings tab scrolls independently via its own `SingleChildScrollView`.
**Why human:** This is the exact `flutter#55388` TabBarView pitfall the plan explicitly called out as "the hardest problem" — it only manifests at runtime with a real `PageView`-backed `TabBarView`, not in a `tester.pump()`-only widget test that doesn't drive the sheet's `DraggableScrollableSheet` + shared controller together.

### 2. Bucket switch visibly re-routes the map

**Test:** With auto-routing ON and an existing multi-segment route, open Settings and tap a different bucket (e.g. Hiking -> Biking/Road).
**Expected:** All existing route segments on the map visibly redraw along road-appropriate paths within a few seconds (not just state changing internally).
**Why human:** Requires a live Valhalla backend response and visual confirmation on the map layer — cannot be verified by unit/widget tests that mock the Dio client.

### 3. Entry-point sheet with 5 cards on a small screen

**Test:** Open the entry-point travel-profile sheet on a small/short device screen.
**Expected:** All 5 cards are reachable via scroll, no visual overflow or clipping at the bottom.
**Why human:** Visual layout/overflow behavior on real screen dimensions; the widget test doesn't assert pixel-level scroll/overflow correctness.

### Gaps Summary

No gaps. All 6 must-have truths, all 6 required artifacts, and all 4 key links are verified in the actual codebase (not just claimed in SUMMARY.md). All CONTEXT.md-locked architectural decisions (Rec B, no family key, full re-resolve on any switch, unified picker, fresh undo baseline, untouched Finish handoff, no sliders, no reverse-direction button, exact costing values) are honored by the code as written. `flutter analyze` is clean and the task's own 62 tests pass. The only outstanding items are 3 human-verification checks for real-device UX behavior (TabBarView scroll interaction, live map re-routing, small-screen sheet layout) that cannot be confirmed by static/automated means — this routes the phase to `human_needed` rather than `passed`, per the verification decision tree (human items take priority over an otherwise-clean score).

A pre-existing, unrelated regression was observed in the full `flutter test` run (4 failures in settings/feed-item tests) — confirmed via `git show --stat` that none of this task's 5 commits touched those files. This is noted for visibility but is not a gap of this task.

---

_Verified: 2026-07-17T20:23:38Z_
_Verifier: Claude (gsd-verifier)_
