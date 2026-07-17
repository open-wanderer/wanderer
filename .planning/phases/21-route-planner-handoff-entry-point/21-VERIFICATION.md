---
phase: 21-route-planner-handoff-entry-point
verified: 2026-07-17T15:17:32Z
status: human_needed
score: 12/12 must-haves verified (code-level)
overrides_applied: 0
human_verification:
  - test: "From New Trail, tap 'Plan a route' — confirm the hike/bike sheet appears, is visually correct (drag handle, two cards, icons), and is dismissible via back button/tap-outside with no navigation occurring on dismiss."
    expected: "Sheet appears styled per UI-SPEC; dismissing without selecting a card returns to trail-source-select with no route pushed."
    why_human: "Visual rendering and gesture-dismiss behavior of showModalBottomSheet cannot be confirmed via static analysis."
  - test: "Tap Hike (or Bike) in the sheet with allowAutoGeolocate on — confirm the planner opens centered on the device's real GPS location; with allowAutoGeolocate off (default/unset), confirm it opens at the non-GPS fallback center instead."
    expected: "Map camera centers on GPS fix when opted in, on mapCameraProvider/settings.location/(0,0) fallback otherwise."
    why_human: "Real-time GPS resolution and map camera rendering require a device/emulator and cannot be verified from source alone."
  - test: "With <2 anchors placed in the planner, confirm the Finish app-bar icon is visibly disabled (~38% opacity) and its long-press tooltip reads 'Add at least 2 anchors to finish your route.' Add 2+ anchors and confirm Finish becomes enabled/tappable, and that undo/redo pills in the top-right controls column still function correctly from their new location."
    expected: "Finish visually disabled below 2 anchors with correct tooltip; enabled with correct tooltip at >=2; undo/redo work identically to their pre-move behavior."
    why_human: "Visual opacity state, tooltip long-press interaction, and drag/gesture-driven undo/redo behavior require on-device confirmation."
  - test: "Tap Finish with a real planned route — confirm the trail create/edit screen opens pre-filled with the planned track (map preview renders the track), the category is pre-selected to a hike/bike-named category, and the waypoints section shows its normal empty state. Save and confirm the saved trail has a real non-zero-distance track."
    expected: "Create/edit screen shows the map preview, correct category pre-fill, empty waypoints list; saved trail persists a real track."
    why_human: "End-to-end map preview rendering, category pre-fill UI, and persisted-trail correctness require an on-device/backend round-trip, not verifiable from source alone."
---

# Phase 21: Route Planner Handoff & Entry Point Verification Report

**Phase Goal:** A user reaches the Route Planner from the trail-source-select flow, chooses an initial travel profile up front, and hands off a finished plan as a draft Trail to the existing create/edit screen.
**Verified:** 2026-07-17T15:17:32Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | From trail-source-select, user sees a new "Plan a route" entry point alongside "Import trail file" (ROADMAP SC1) | ✓ VERIFIED | `trail_source_select_screen.dart:123-130` — `_SourceActionCard` with `title: l10n.trail_source_planner`, `onTap: () => _openPlanner(l10n)`, sits alongside the "Import trail file" card (`l10n.trail_source_import`, line 140-147) |
| 2 | Tapping the entry point shows a hike/bike dialog before the planner opens; choice fixes the session's initial travel profile (ROADMAP SC2, HANDOFF-03) | ✓ VERIFIED | `travel_profile_sheet.dart` — `showTravelProfileSheet` returns `'pedestrian'`/`'bicycle'`/`null`; `_openPlanner` awaits it and only proceeds to push `/route-planner` on a non-null profile; `RoutePlannerScreen.travelProfile` is a `final` field set once at construction (no in-planner switch) |
| 3 | User can finish planning and hand off the route as a draft Trail (GPX-only, no Waypoint records, elevation populated regardless of Elevation-tab visit) opening in create/edit (ROADMAP SC3, HANDOFF-01) | ✓ VERIFIED | `route_planner_handoff_util.dart` `finishPlanning` unconditionally attempts the one-time `/valhalla/height` fetch every call (not gated on tab visit), builds `buildDraftTrail` with `waypointsViaTrail: const []`, hands off via `pendingImportedTrail` + `navContext.push('/trail/create/edit', extra: draftTrail)` |
| 4 | `finishPlanning` builds draft Trail from final `plannedGpxProvider` route, hands to `/trail/create/edit` via `pendingImportedTrail`, waypoints empty (D-07, HANDOFF-01) | ✓ VERIFIED | `route_planner_handoff_util.dart:79-108`; `grep -c "Waypoint("` = 0 in file; `waypointsViaTrail: const []` present |
| 5 | One-time `POST /valhalla/height` merges ele; failure hands off silently with pre-elevation GPX, no error UI (D-06) | ✓ VERIFIED | try/catch around the fetch with empty catch body (comment "D-06: proceed silently"); zero `toastProvider` references in the file |
| 6 | `categoryForTravelProfile` maps 'bicycle'→bike category, 'pedestrian'→hike category, null on no match (D-08) | ✓ VERIFIED | `gpx_util.dart` — mirrors `costingForCategory`'s token order; `firstWhereOrNull`; unit-tested (5 passing cases including symmetry + no-match + empty-list) |
| 7 | Settings exposes nullable `behavior` field carrying `allowAutoGeolocate` (D-03) | ✓ VERIFIED | `settings.dart:94-116` — `class Behavior` with `bool? allowAutoGeolocate`; `Settings.behavior` field |
| 8 | `allowAutoGeolocate` persists across restarts via `SettingsEntity.behaviorJson`, same strategy as `privacyJson` | ✓ VERIFIED | `settings_entity.dart:22,60-62,102-107` — identical jsonEncode/jsonDecode shape to `privacyJson` |
| 9 | Settings round-trip preserves behavior; absent behavior resolves to null, never silently `true` | ✓ VERIFIED | `settings_entity.dart:102-107,118` — `Behavior? behav` stays `null` when `behaviorJson == null`; passed straight into `Settings(... behavior: behav)` — no default-to-true path exists |
| 10 | "Plan a route" card opens a dismissible sheet with two Hike/Bike cards (D-01/D-02, HANDOFF-03) | ✓ VERIFIED | `travel_profile_sheet.dart` — `isDismissible: true`, `enableDrag: true`, two `_TravelProfileCard`s (personHiking/bicycle icons), no `isDismissible: false` anywhere |
| 11 | Tapping a card closes the sheet, resolves GPS-gated-or-fallback center, pushes `/route-planner` with fixed travel profile; TEMPORARY stubs replaced (D-03, HANDOFF-02/03) | ✓ VERIFIED | `trail_source_select_screen.dart:40-83` GPS gate on `settings?.behavior?.allowAutoGeolocate != true`; `router_provider.dart:250-265` real registration reading `extra?['travelProfile']`/`lat`/`lon`; zero `TEMPORARY` occurrences remain in `router_provider.dart` |
| 12 | App bar shows single Finish action (≥2-anchor gated); undo/redo relocated to controls column; tapping Finish runs `finishPlanning` (D-04/D-05, HANDOFF-01) | ✓ VERIFIED | `route_planner_screen.dart:150` (`actions: [_buildFinishAction(...)]`), :254-265 (undo/redo pills in `Positioned(top:128,right:0)` Column), :436-459 (`_buildFinishAction`/`_onFinish` gated on `state.anchors.length >= 2`, calling `finishPlanning(ref: ref, navContext: context, travelProfile: widget.travelProfile)`) |

**Score:** 12/12 truths verified at the code level. See Human Verification section below — 4 items require on-device confirmation before the phase can be marked fully passed (visual/gesture/GPS/persisted-data behaviors that static analysis cannot observe).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/route_planner_handoff_util.dart` | `finishPlanning` orchestration + pure helpers | ✓ VERIFIED | Exists, substantive (108 lines), wired (imported/called by `route_planner_screen.dart`), unit-tested (8 passing cases) |
| `app/lib/util/gpx_util.dart` | `categoryForTravelProfile` reverse-lookup | ✓ VERIFIED | Added function, correct inverse logic, imported by `route_planner_handoff_util.dart`, unit-tested (5 passing cases) |
| `app/test/util/route_planner_handoff_util_test.dart` | Unit tests for D-07/gpxData/ele-merge invariants | ✓ VERIFIED | 8 tests, all passing |
| `app/lib/models/settings.dart` | `Behavior` freezed class + `Settings.behavior` field | ✓ VERIFIED | Present, `flutter analyze` clean, codegen regenerated |
| `app/lib/entities/settings_entity.dart` | `behaviorJson` encode/decode | ✓ VERIFIED | 5 occurrences (field, ctor param, encode, decode, return), mirrors `privacyJson` exactly |
| `app/lib/components/route_planner/travel_profile_sheet.dart` | `showTravelProfileSheet` + hike/bike cards | ✓ VERIFIED | Exists, substantive, wired into `trail_source_select_screen.dart` |
| `app/lib/routes/trail_source_select_screen.dart` | "Plan a route" onTap → sheet → GPS-gated center → push | ✓ VERIFIED | `_openPlanner`/`_resolveInitialCenter`/`_fallbackCenter` all present and called |
| `app/lib/provider/router_provider.dart` | Real `/route-planner` registration | ✓ VERIFIED | TEMPORARY stub fully removed; reads `travelProfile`/`lat`/`lon` from `extra` |
| `app/lib/routes/route_planner_screen.dart` | App-bar Finish action + undo/redo relocation + `_onFinish` | ✓ VERIFIED | All present, wired to `finishPlanning`, `_finishing` re-entrancy guard in place |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `route_planner_handoff_util.dart` | `pendingImportedTrail` + `/trail/create/edit` | `navContext.push` | ✓ WIRED | `pendingImportedTrail = draftTrail;` then `navContext.push('/trail/create/edit', extra: draftTrail)` |
| `route_planner_handoff_util.dart` | `POST /valhalla/height` | `apiProvider.post` in try/catch | ✓ WIRED | Silent catch, no toast, D-06 compliant |
| `trail_source_select_screen.dart` | `showTravelProfileSheet` + `/route-planner` | `await` sheet result → `context.push` | ✓ WIRED | `_openPlanner` composes both steps correctly |
| `trail_source_select_screen.dart` | `Settings.behavior?.allowAutoGeolocate` | GPS gate | ✓ WIRED | `_resolveInitialCenter` short-circuits to fallback when gate is not `true` |
| `router_provider.dart` | `RoutePlannerScreen(travelProfile, initialCenter)` | `state.extra` map | ✓ WIRED | Null-safe fallbacks (`'pedestrian'`, `Geographic(0,0)`) if extra missing |
| `route_planner_screen.dart` | `finishPlanning(ref, navContext, travelProfile)` | Finish `IconButton.onPressed` → `_onFinish` | ✓ WIRED | Gated on `anchors.length >= 2 && !_finishing` |
| `route_planner_screen.dart` | `notifier.undo`/`notifier.redo` | Relocated pills in controls Column | ✓ WIRED | `_buildUndoButton`/`_buildRedoButton`, correctly reading `undoStack`/`redoStack` |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Phase-specific unit tests pass | `flutter test test/util/gpx_util_test.dart test/util/route_planner_handoff_util_test.dart` | 19/19 passed | ✓ PASS |
| Whole-app static analysis has no new errors | `flutter analyze` (whole app) | 46 pre-existing info/warning issues, 0 in phase-21 files, 0 errors | ✓ PASS |
| Whole-app test suite has no NEW regressions beyond documented baseline | `flutter test` (whole suite) | 4 failing (`feed_item_test.dart` x2, `settings_screen_test.dart` x1, `settings_account_screen_test.dart` x1) — exactly matches the count and identity logged in `deferred-items.md` (grown from 3 pre-existing to 4, with the 4th explicitly investigated and confirmed unrelated to this phase's diff) | ✓ PASS (no new unexplained regressions) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| HANDOFF-01 | 21-01, 21-04 | Finish planning hands off draft Trail (GPX-only, elevation populated at handoff regardless of tab visit) to create/edit | ✓ SATISFIED | `finishPlanning` + app-bar Finish wiring, both verified above |
| HANDOFF-02 | 21-03 | Route Planner reachable from trail-source-select entry point | ✓ SATISFIED | "Plan a route" card + real `/route-planner` registration |
| HANDOFF-03 | 21-02, 21-03 | Hike/bike dialog before planner opens; fixed travel profile for session | ✓ SATISFIED | `travel_profile_sheet.dart` + `Behavior.allowAutoGeolocate` gate |

No orphaned requirements found — REQUIREMENTS.md maps only HANDOFF-01/02/03 to Phase 21, and all three are claimed by at least one plan.

### Anti-Patterns Found

None. Scanned all 8 modified/created files for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/empty implementations/hardcoded-empty props — no genuine matches (one false-positive grep hit on `toDouble()`/`totalDuration` substring-matching `TODO`/`TBD` case-insensitively, not an actual marker). No `static` fields were added (checked per Pitfall 3 in both `route_planner_handoff_util.dart` and `route_planner_screen.dart`). All prior TEMPORARY Phase-19 stub comments confirmed removed from `router_provider.dart` and `trail_source_select_screen.dart`.

### Human Verification Required

Four items need on-device testing before this phase can be marked fully `passed` — all explicitly flagged as outstanding by the plans' own `<verification>` sections and 21-04-SUMMARY.md itself ("HUMAN on-device verification ... is still outstanding"). These are genuine gaps in evidence, not just formality: visual sheet rendering/dismissal, real GPS resolution behavior, Finish button enabled/disabled visual + tooltip state, and the full save round-trip producing a non-zero-distance trail, cannot be confirmed by static code reading alone.

1. **Hike/bike sheet appearance & dismiss behavior** — see frontmatter `human_verification[0]`
2. **GPS-gated vs fallback initial center** — see frontmatter `human_verification[1]`
3. **Finish button disabled/enabled visual state + undo/redo from new location** — see frontmatter `human_verification[2]`
4. **Full handoff round-trip (map preview, category pre-fill, save produces real track)** — see frontmatter `human_verification[3]`

### Gaps Summary

No code-level gaps. All 12 derived truths (3 ROADMAP success criteria + 9 PLAN-frontmatter must-haves) are verified as implemented and correctly wired in the codebase, all phase-specific unit tests pass, `flutter analyze` reports zero new errors, and the whole-suite `flutter test` regression count (4 failures) exactly matches the documented, investigated, and explicitly-deferred baseline in `deferred-items.md` — no unexplained new failures were introduced by this phase.

The only reason this report is not `passed` is that the phase's own plans (21-03, 21-04) explicitly deferred device-level UI/GPS/persistence confirmation to end-of-phase human verification, and 21-04-SUMMARY.md itself states that verification is still outstanding. This is expected process, not a defect — but per the verification methodology, `passed` is only valid when the human-verification section is empty.

---

*Verified: 2026-07-17T15:17:32Z*
*Verifier: Claude (gsd-verifier)*
