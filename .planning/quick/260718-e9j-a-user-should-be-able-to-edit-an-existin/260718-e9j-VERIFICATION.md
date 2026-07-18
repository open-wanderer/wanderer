---
phase: 260718-e9j
verified: 2026-07-18T09:15:00Z
status: human_needed
score: 6/6 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Open an existing trail with a recorded track in trail create/edit screen"
    expected: "The route button appears left of Save and is enabled; a trail with no track (or <2 seedable anchors) shows it disabled"
    why_human: "Visual app-bar layout/enabled-state rendering on a real device/simulator cannot be confirmed by static analysis"
  - test: "Tap the route button"
    expected: "Route planner opens centered on the route with anchors prepopulated at the track's segment-boundary points"
    why_human: "Map camera centering and visual anchor placement require rendering the native MapLibre view"
  - test: "Move/add an anchor, press Finish"
    expected: "Returns to the create screen (no duplicate screen pushed), map preview shows the edited track, title/description/photos entered earlier are still present"
    why_human: "Navigation-stack behavior and live map-preview rendering require on-device interaction"
  - test: "Enter the planner again and back out via system back gesture without pressing Finish"
    expected: "The trail is unchanged"
    why_human: "System back-gesture behavior and in-memory state persistence across a real navigation pop require on-device interaction"
---

# Quick Task 260718-e9j: Edit an existing route in the trail planner — Verification Report

**Task Goal:** A user should be able to edit an existing route in the trail planner. The trail_create_screen gives the option to open a trail in the route planner. This navigates to the route_planner_screen. Anchors are prepopulated (mirroring the web version). Saving the trail in the route planner returns the edited trail to the awaiting trail_create_screen.

**Verified:** 2026-07-18T09:15:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | trail_create_screen shows an app-bar button (left of Save) enabled only when the trail has a usable recorded track | ✓ VERIFIED | `trail_create_screen.dart:430-450` — `IconButton` inserted before the Save `IconButton` in `AppBar.actions`; `onPressed` gated on `seedAnchors.length >= 2` (post-review-fix, stricter than a bare `trks.isNotEmpty` null-track check — see Anti-Patterns note below) |
| 2 | Opening the planner prepopulates anchors at each track-segment boundary, mirroring web `initRouteAnchors`, no interior sampling, no reverse-geocode at seed time | ✓ VERIFIED | `route_planner_handoff_util.dart:151-170` `anchorsFromTrack` mirrors `web/src/routes/trail/edit/[id]/+page.svelte:553-577` (first pt of every non-empty trkseg + last pt of final non-empty trkseg); `route_anchor_provider.dart:384-425` `seedFromTrack` builds anchors/segments directly (never loops `appendAnchor`) and never calls `_resolveAnchorLocation`; 3 dedicated tests plus 3 CR-01/WR-01/WR-02 regression tests pass |
| 3 | Pressing Finish/Save in the planner returns the edited route to the waiting trail_create_screen via pop-with-result, no duplicate create-screen push | ✓ VERIFIED | `route_planner_screen.dart:481-495` `_onFinish`'s edit-mode branch calls `buildFinalPlannedGpx(ref)` then `context.pop(finalGpx)`; `trail_create_screen.dart:257-267` `_onEditRoute` does `await context.push<Gpx>('/route-planner', ...)` — pop/push pair confirmed, no `navContext.push('/trail/create/edit')` call in the edit-mode path |
| 4 | The returned route replaces the trail's track (expand.gpx AND expand.gpxData both set) while title/description/photos/id/visibility/waypoints stay untouched | ✓ VERIFIED | `route_planner_handoff_util.dart:188-207` `mergeRouteIntoTrail` sets both fields via `copyWith` on `existing.expand`, preserving all non-track `Trail`/`TrailExpand` fields; test `mergeRouteIntoTrail preserves title/description/id and existing waypoints unchanged` passes |
| 5 | Backing out of the planner without saving leaves the in-memory trail unchanged | ✓ VERIFIED | `trail_create_screen.dart:265` — `if (newGpx == null \|\| !mounted) return;` before any `setState`/merge; a pop with no result (back gesture) yields `newGpx == null` |
| 6 | The existing GPX-import finishPlanning forward-push path continues to work unchanged | ✓ VERIFIED | `route_planner_handoff_util.dart:127-142` `finishPlanning` unchanged tail (`categoryForTravelProfile` → `buildDraftTrail` → `pendingImportedTrail` → `navContext.push('/trail/create/edit')`); grep gate `navContext.push('/trail/create/edit'` matches; pre-existing `buildDraftTrail`/`mergeHeightsIntoGpx` tests still pass |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/route_planner_handoff_util.dart` | `anchorsFromTrack`, `mergeRouteIntoTrail`, `buildFinalPlannedGpx` extracted from `finishPlanning` | ✓ VERIFIED | All three present (lines 97-115, 151-170, 188-207); `mergeRouteIntoTrail` contains the expected pattern |
| `app/lib/provider/route_anchor_provider.dart` | `seedFromTrack(points, profile, opts)` on `RouteAnchors` | ✓ VERIFIED | Present at lines 384-425; contains `seedFromTrack` |
| `app/lib/routes/route_planner_screen.dart` | edit-mode seeding + pop-return Finish | ✓ VERIFIED | `seedAnchors` field (line 57), `_editMode` getter (125-126), post-frame branch (133-148), pop branch in `_onFinish` (485-489) |
| `app/lib/routes/trail_create_screen.dart` | app-bar entry button + `await push<Gpx>` + `mergeRouteIntoTrail` on return | ✓ VERIFIED | `IconButton` (439-449), `_onEditRoute` (257-267) awaits `push<Gpx>` and calls `mergeRouteIntoTrail(trail, newGpx)` |
| `app/lib/provider/router_provider.dart` | `/route-planner` builder reads `mode` + `seedAnchors` from extra | ✓ VERIFIED | Lines 251-273; reads `seedAnchors` and forwards to `RoutePlannerScreen` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `trail_create_screen.dart` | `/route-planner` | `context.push<Gpx>` with `extra{mode:edit, seedAnchors}` | ✓ WIRED | `grep -q "push<Gpx>('/route-planner'"` matches; extra map includes `mode`, `seedAnchors`, `travelProfile`, `lat`, `lon` |
| `router_provider.dart` | `RoutePlannerScreen` | `seedAnchors` constructor param | ✓ WIRED | `grep -q "seedAnchors"` matches; passed positionally into `RoutePlannerScreen(...)` |
| `route_planner_screen.dart` | `seedFromTrack` | post-frame callback branch on `seedAnchors` | ✓ WIRED | `grep -q "seedFromTrack"` matches; called inside `initState`'s post-frame callback when `_editMode` is true |
| `route_planner_screen.dart` | `context.pop(finalGpx)` | `_onFinish` edit-mode branch | ✓ WIRED | `grep -q "context.pop(finalGpx)"` matches |
| `trail_create_screen.dart` | `mergeRouteIntoTrail` | `setState` on returned `Gpx` | ✓ WIRED | `grep -q "mergeRouteIntoTrail(trail"` matches; called inside `setState` in `_onEditRoute` |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| New/regression unit tests pass | `cd app && flutter test test/util/route_planner_handoff_util_test.dart test/provider/route_anchor_provider_test.dart` | 54/54 tests passed, including 3 CR-01/WR-01/WR-02 regression tests | ✓ PASS |
| Static analysis clean on all 5 touched files | `cd app && flutter analyze lib/util/route_planner_handoff_util.dart lib/provider/route_anchor_provider.dart lib/routes/route_planner_screen.dart lib/provider/router_provider.dart lib/routes/trail_create_screen.dart` | 1 pre-existing info-level lint (`curly_braces_in_flow_control_structures` at `route_anchor_provider.dart:176`, confirmed pre-existing and untouched by this task) | ✓ PASS |
| Task grep gates (push<Gpx>, mergeRouteIntoTrail, seedFromTrack, context.pop(finalGpx), seedAnchors, forward-push regression) | See Key Link Verification table | All 7 grep gates matched | ✓ PASS |

Full app build/run and on-device navigation flow could not be exercised in this text-only environment — see Human Verification below (this mirrors the `<human-check>` block the plan itself deferred to end-of-phase in Task 3).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| PLANNER-02 | 260718-e9j-PLAN.md | Edit an existing trail's route via the Route Planner | ✓ SATISFIED | All 6 observable truths verified; SUMMARY.md marks it complete and codebase evidence supports that |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | Code review findings CR-01, WR-01, WR-02 (unguarded null-check crash, degenerate-track silent mode-downgrade, dropped final point on trailing empty trkseg) | Resolved | Confirmed fixed in commit `38210d8f` (present at HEAD); regression tests added and passing — see truths #1 and #2 evidence above |
| `app/lib/util/route_planner_handoff_util.dart:181-200` (implied via `trail_create_screen.dart:207-218`) | — | WR-03 (review, not a must-have): `mergeRouteIntoTrail` preserves waypoints' `distanceFromStart` unchanged even though the edited track's shape/length may have changed, leaving cached distances stale until a subsequent server save recomputes them | Info | Not part of this task's must_haves; not blocking. Documented in REVIEW.md as an open warning, left unaddressed by the follow-up fix commit (which addressed only CR-01/WR-01/WR-02 per the task's own scoping) |
| `app/lib/routes/trail_create_screen.dart:441` | 441 | IN-01 (review): hardcoded, non-localized `tooltip: 'Edit route'` string | Info | Pre-existing pattern elsewhere in the file/screen (e.g. `route_planner_screen.dart`'s `'Search location'`); not a new regression, not a must-have |
| `app/lib/routes/trail_create_screen.dart:259` | 259 | IN-03 (review): inert `'mode': 'edit'` extra key never read by `router_provider.dart` | Info | Documented/intentional per SUMMARY.md decisions; not a functional gap |

No `TBD`/`FIXME`/`XXX` debt markers found in any of the 5 modified source files or 2 modified test files.

### Human Verification Required

The plan's own Task 3 `<human-check>` block defers exactly this kind of on-device confirmation to end-of-phase — harvested below.

#### 1. Entry-point button visibility and enabled state

**Test:** Open an existing trail with a recorded track in the trail create/edit screen.
**Expected:** The route button appears left of Save and is enabled; a trail with no track (or a degenerate track yielding fewer than 2 anchors) shows it disabled.
**Why human:** Visual app-bar rendering and enabled/disabled opacity treatment can only be confirmed by looking at a running app.

#### 2. Planner opens seeded and centered

**Test:** Tap the route button.
**Expected:** The route planner opens centered on the existing route, with anchors visible at the track's start and end (and any interior segment boundaries).
**Why human:** Native MapLibre camera centering and anchor marker rendering cannot be verified via static code inspection.

#### 3. Finish returns and merges correctly

**Test:** Move/add an anchor, press Finish.
**Expected:** Returns to the same create screen (no duplicate screen on the nav stack), the map preview shows the edited track, and the title/description/photos entered before opening the planner are still present.
**Why human:** Navigation-stack identity and live map-preview re-render require on-device interaction; cannot be grepped.

#### 4. Back-out leaves trail unchanged

**Test:** Enter the planner again and back out with the system back gesture (no Finish tap).
**Expected:** The trail is unchanged.
**Why human:** System back-gesture behavior triggering a `null` pop result is implemented correctly in code (`newGpx == null` guard), but confirming the actual gesture wiring end-to-end requires a live device/simulator.

### Gaps Summary

No gaps found. All 6 must-have observable truths, all 5 required artifacts, and all 5 key links are verified present, substantive, and wired. The three critical/warning findings from the code review (CR-01, WR-01, WR-02) that this verification was specifically asked to re-check were confirmed fixed in commit `38210d8f`, which is present at the current HEAD of `feature/app` — not merely logged as intended fixes. The full new/regression test suite (54 tests) passes, and static analysis on all 5 touched files reports only a single pre-existing, out-of-scope info-level lint.

Status is `human_needed` rather than `passed` solely because the plan's Task 3 explicitly deferred on-device UI/navigation confirmation (button visibility, map centering, pop-return round trip, back-out no-op) to end-of-phase human verification — none of this is a code gap, it is inherent to what static verification can confirm for a Flutter/native-map screen.

Two review findings (WR-03 stale waypoint distances, IN-01 non-localized tooltip, IN-03 inert extra key) remain open but were never part of this task's must_haves and do not block the goal.

---

_Verified: 2026-07-18T09:15:00Z_
_Verifier: Claude (gsd-verifier)_
