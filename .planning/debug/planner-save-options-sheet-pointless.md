---
status: diagnosed
trigger: "b) partial. Why does the save options sheet appear in route planner when online? It has no purpose. The route is following roads and using valhalla elevation anyways"
created: 2026-08-01T00:00:00Z
updated: 2026-08-01T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED — `resolveTrackSaveOptions` is a source-blind shared gate. The planner's new-session Finish calls it unconditionally. `recalcHeights` is a provable no-op for every planner state; `followRoads` is redundant-and-lossy for a fully routed route but NOT a no-op when a leg is `straight` or `blocked`.
test: (complete) traced all three capture sources, all `/route-planner` entry points, `SegmentState` transitions, `elevationProfile` provenance, and every `snapCosting` reference.
expecting: n/a — diagnosis complete
next_action: hand diagnosis to plan-phase --gaps. Do NOT fix here (goal: find_root_cause_only).

## Symptoms

expected: The online route planner's Finish action should not offer save options that cannot change the result.
actual: Tapping Finish online (new-session planning) shows a bottom sheet offering "Recalculate heights" and "Follow roads" toggles.
errors: none
reproduction: Test 1(b) in .planning/phases/34-dart-conversion-port/34-UAT.md — online, plan a 3-anchor route, tap Finish.
started: Phase 34, commit 07794d24 ("feat(34-06): add shared online gate…") wired the planner through the recording flow's sheet.

## Eliminated

- hypothesis: "The sheet is shown because a planner-specific flag requests it"
  evidence: `resolveTrackSaveOptions(WidgetRef, BuildContext)` takes no source parameter at all (track_save_options_util.dart:29-43). No caller can suppress it while online. Disproved.

- hypothesis: "Both toggles are no-ops for the planner, so the sheet is pure noise"
  evidence: `recalcHeights` — yes, provable no-op (see Evidence). `followRoads` — NO. `SegmentState.straight` (auto-routing switched off in Settings tab) and `SegmentState.blocked` (Valhalla returned no shape / non-2xx) both leave a 2-point straight line as the leg polyline. Partially disproved.

- hypothesis: "The re-pin stays reachable via import-then-edit in the planner"
  evidence: `trail_create_screen.dart:322` opens the planner in EDIT mode; `route_planner_screen.dart:523-527` takes the edit branch and calls `buildFinalPlannedGpx(ref)` with default args (`snapCosting: null`). The import flow itself uses `snapShapeToRoads` in `trail_import_util.dart:135`, never `buildFinalPlannedGpx`. Disproved.

## Evidence

- timestamp: T0
  checked: app/lib/util/track_save_options_util.dart:29-43
  found: single shared gate, no source parameter. Offline → `(false,false)` with no sheet. Online → always shows the sheet.
  implication: the planner gets the sheet purely because it calls the same helper as recording/import.

- timestamp: T1
  checked: grep for `resolveTrackSaveOptions`
  found: exactly 3 production callers — `navigation_screen.dart:736` (recording), `trail_import_util.dart:104` (file import), `route_planner_screen.dart:529` (planner, new-session branch only).
  implication: the sheet was designed for the recording flow (it lives in `components/navigation/`, default title `l10n.save_recording_options`) and was extended to two more sources in 34-06 without per-source relevance analysis.

- timestamp: T2
  checked: route_planner_screen.dart:520-558
  found: `_onFinish` already branches. Edit mode (`seedAnchors` non-empty) calls `buildFinalPlannedGpx(ref)` with NO gate and NO options. Only the new-session branch calls `resolveTrackSaveOptions` + `finishPlanning`.
  implication: a per-flow suppression precedent already exists in this exact method.

- timestamp: T3
  checked: route_planner_handoff_util.dart:509-533 and route_anchor_provider.dart:332-372
  found: `_resolveElevation` POSTs `buildNavShape(polyline)` to `/valhalla/height` and stores that EXACT shape as `elevationProfile` alongside `elevations`. `buildFinalPlannedGpx` sets `legPoints[i] = s.elevationProfile ?? s.polyline`, and with `refetchAllHeights: true` calls `fetchHeightsForShape(ref, legPoints[i])` — the identical shape, to the identical deterministic DEM endpoint.
  implication: for a leg with elevations, recalcHeights re-requests the same body and gets the same numbers. For a leg WITHOUT elevations, it is already in `pending` regardless of the flag (line 514-517). "Recalculate heights" is a provable no-op in every planner state.

- timestamp: T4
  checked: route_anchor_provider.dart:436-441, 478-488, 632-651, 681-694; components/route_planner/settings_tab.dart:40-46
  found: `autoRoutingEnabled` is a USER-FACING switch in the planner's Settings tab. With it off, `appendAnchor`/`dragAnchor`/`resolveAllSegments` apply a 2-point straight polyline with `SegmentState.straight` and never call Valhalla.
  implication: a planner route can consist entirely of straight lines. "Follow roads" has genuine work to do. The user's blanket claim is refuted for this case.

- timestamp: T5
  checked: route_anchor_provider.dart:218-288 (`_resolveSegment`), 290-299 (`_markBlocked`)
  found: a Valhalla failure (out-of-range coords, non-2xx from the proxy, response with no `trip.legs[0].shape`, connection error) marks the segment `blocked` and leaves the polyline untouched. For a freshly appended anchor that polyline is the straight 2-point line created at line 632-637.
  implication: an UNROUTABLE anchor pair (anchor in water, no path across a barrier) leaves a straight-line leg while the app is fully ONLINE — `badResponse` never flips `onlineStatusProvider` (online_status_provider.dart:28-34). "Follow roads" is meaningful here with no offline transition involved at all.

- timestamp: T6
  checked: trail_source_select_screen.dart:216,232 (HEAD) and provider/online_status_provider.dart:47-50
  found: the planner card is disabled when `!isOnline`, so the planner has no offline ENTRY point. But `OnlineStatus.build() => true` is optimistic and `_openPlanner` never calls `.refresh()` (unlike `_openRecorder`), so in airplane mode the card is still tappable until some request fails. Once inside, `routeAnchorsProvider` is `keepAlive: true` and the screen never force-exits.
  implication: plan-while-offline → reconnect → Finish IS reachable: every leg planned during the outage is `blocked` with straight-line geometry, and the sheet then appears online with real work for "Follow roads".

- timestamp: T7
  checked: grep `snapCosting` across app/lib
  found: 4 hits, all in route_planner_handoff_util.dart. The only production assignment is line 616, `snapCosting: followRoads ? (bucket?.costing ?? 'pedestrian') : null`, inside `finishPlanning`. `finishPlanning`'s only caller is route_planner_screen.dart:537. `followRoads` originates only from the sheet.
  implication: the leg-boundary anchor re-pin (route_planner_handoff_util.dart:486-496) is reachable from exactly ONE user flow: new-session planner Finish with "Follow roads" enabled. Suppressing the sheet makes it unreachable from every user flow. It remains referenced only by app/test/util/route_planner_handoff_util_test.dart:763,813.

- timestamp: T8
  checked: route_planner_handoff_util.dart:470-497 vs route_anchor_provider.dart:341
  found: `followRoads` on an ALREADY-routed leg sends `buildNavShape(legPoints[i])` — a decimation of `elevationProfile`, itself already a ≤500-point decimation of the Valhalla route — to `/valhalla/trace-route`, and replaces the leg with the map-matched result, discarding good elevations for a refetch.
  implication: for the normal case "Follow roads" is not literally a no-op; it is a lossy re-round-trip of geometry Valhalla already produced. It can only degrade fidelity. The user's intuition is right; the mechanism is "redundant and lossy", not "identical output".

- timestamp: T9
  checked: git status / git diff
  found: sibling debug sessions have already modified `track_save_options_util.dart` (adds `title: "Adjust track"` + bottom padding), `track_save_options_sheet.dart` and `trail_source_select_screen.dart` in the working tree. At HEAD the call was bare `showTrackSaveOptionsSheet(context)`, which falls back to `l10n.save_recording_options` — this is the "Save recordings" title the user reported in gap (c).
  implication: gap (c) is being handled elsewhere; any fix here must not conflict with those edits to the same file.

- timestamp: T10 (adjacent finding, NOT the reported bug)
  checked: route_anchor_provider.dart:301-320 + models/route_anchor.dart:43-64
  found: `_applySegment` calls `segment.copyWith(polyline:, state:, durationSeconds:)`. Freezed preserves unlisted fields, so `elevationProfile`/`elevations` from the PREVIOUS geometry survive the polyline replacement until the new fire-and-forget `_resolveElevation` lands (or fails, or is cancelled). Since `buildFinalPlannedGpx` prefers `elevationProfile` over `polyline`, tapping Finish inside that window saves the leg's PRE-EDIT geometry.
  implication: a genuine latent data bug, independent of this gap, and NOT fixed by `recalcHeights` (which refetches heights over the same stale profile).

## Resolution

root_cause: |
  `resolveTrackSaveOptions` (app/lib/util/track_save_options_util.dart:29) is a source-blind gate.
  Plan 34-06 routed all three capture sources through it ("one code path, not two" for D-15's
  offline handling) and, in doing so, gave the route planner a sheet that was written for the
  recording flow — where a raw GPS breadcrumb genuinely benefits from road-snapping and DEM
  heights. The planner's own pipeline already performs both operations per leg while the user
  builds the route (`_resolveSegment` → /valhalla/route, `_resolveElevation` → /valhalla/height),
  so the gate re-offers work that is already done. The gate has no parameter a caller could use
  to opt out while online, so route_planner_screen.dart:529 cannot suppress it.

fix: (not applied — diagnose-only mode)
verification: (n/a)
files_changed: []
