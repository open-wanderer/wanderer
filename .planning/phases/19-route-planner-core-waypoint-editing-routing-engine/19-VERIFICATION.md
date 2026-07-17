---
phase: 19-route-planner-core-waypoint-editing-routing-engine
verified: 2026-07-16T14:37:14Z
human_verified: 2026-07-16
status: passed
score: 13/13 must-haves verified
human_verification_status: "6/6 items passed (2 required a fix before passing — see human_verification results below)"
overrides_applied: 1
overrides:
  - must_have: "Toggling auto-routing re-resolves every existing segment to match the new mode"
    reason: "User decision (2026-07-16): OFF preserves existing segment state/geometry instead of discarding routed polylines; only new segments (created after the toggle) are straight. ON is unchanged (re-resolves every segment via Valhalla in parallel). ROADMAP.md Success Criterion #4, REQUIREMENTS.md ROUTE-02, and 19-04-PLAN.md must_haves.truths line 16 amended to match."
    accepted_by: "user"
    accepted_at: "2026-07-16"
gaps:
  - truth: "Toggling auto-routing re-resolves every existing segment to match the new mode (ROADMAP Success Criterion #4; REQUIREMENTS.md ROUTE-01/ROUTE-02)"
    status: resolved_via_override
    reason: >
      ROADMAP.md's Phase 19 Success Criterion #4 states: "A user can toggle
      auto-routing on ... or off (straight-line segments); toggling
      re-resolves every existing segment to match the new mode."
      REQUIREMENTS.md's ROUTE-02 states the same: "Toggling auto-routing
      re-resolves all existing segments to the new mode." The shipped
      implementation only does this for the OFF->ON direction (re-resolves
      every segment via Valhalla in parallel). For ON->OFF, `toggleAutoRouting()`
      flips the flag and returns immediately — every existing segment's
      polyline/state is left exactly as it was (e.g. a previously `routed`
      segment stays visually `routed`, solid blue with white casing, not
      converted to a straight line). Only segments created AFTER the toggle
      (via appendAnchor/dragAnchor/insertAnchorOnSegment) become straight.
      This was a deliberate, documented mid-execution correction (19-02-SUMMARY.md
      "Decisions Made") driven by a coordinator-flagged edit to
      19-02-PLAN.md's own must_haves.truths — but ROADMAP.md, REQUIREMENTS.md,
      and 19-04-PLAN.md's own must_haves.truths (line 16: "toggling
      re-resolves every existing segment to the new mode (ROUTE-01/02)")
      were never amended to match, unlike the ROUTE-03 cut earlier in this
      phase's discussion (which formally amended all three planning docs).
      This is a genuine, unresolved conflict between the locked roadmap/
      requirements contract and the shipped behavior, not a false positive.
    artifacts:
      - path: "app/lib/provider/route_anchor_provider.dart"
        issue: "toggleAutoRouting() OFF branch (lines 196-200) returns immediately without touching state.segments — no re-resolve to straight for existing segments"
      - path: "app/lib/routes/route_planner_screen.dart"
        issue: "Auto-routing toggle button (line 301-303) delegates entirely to toggleAutoRouting() with no additional re-resolve logic on OFF"
    missing:
      - "Either: implement OFF->re-resolve-to-straight for every existing segment (matching ROADMAP SC#4 / ROUTE-02 literally), OR formally amend ROADMAP.md Phase 19 Success Criterion #4, REQUIREMENTS.md ROUTE-01/ROUTE-02 wording, and 19-04-PLAN.md's must_haves.truths line 16 to describe the 'OFF leaves existing segments untouched, only new segments are straight' behavior actually shipped — following the same amend-all-docs precedent already used for the ROUTE-03 cut in this phase's own DISCUSSION-LOG.md."
human_verification:
  - test: "Tap empty map space to add a route anchor; verify a numbered marker (1) appears and, on a second tap, marker (2) appears connected by a segment line"
    expected: "Two numbered markers on the map, joined by a visible segment line; auto-routing on by default resolves the segment via Valhalla within a moment"
    why_human: "Requires a real/emulated device — ml.MapLibreMap's native platform channel cannot be constructed in the widget-test harness (no precedent in this codebase per trail_map.dart/trail_create_screen.dart); plans explicitly defer this to end-of-phase human verification (human_verify_mode=end-of-phase)"
    result: "PASSED on first open; FAILED on re-entering the screen a second time in the same app session (2026-07-16) — markers appeared but no segment line rendered at all (straight, routed, or blocked). Root cause: `_segmentLayer` in route_planner_screen.dart was a `static final RouteSegmentLayer()`, so its internal `_added` flag persisted stale across screen exits, causing the second open's fresh native style to skip adding the source/layers. Fixed by making it a non-static instance field. Re-verified on device (2026-07-16): PASSED, including re-entering the screen multiple times."
  - test: "Drag an existing anchor marker to a new position and release"
    expected: "Marker moves smoothly during drag (screen-local, no live route preview); on release, its ≤2 adjacent segments re-resolve to the current routing mode; a distant segment elsewhere in the route is unaffected"
    why_human: "Native GestureDetector.onPanStart/Update/End drag interaction against a real map surface; unit tests cover the state-mutation logic but not the actual on-screen drag feel/timing"
    result: "PASSED (2026-07-16)"
  - test: "Tap an existing plain (non-blocked) segment partway along its length"
    expected: "A new numbered anchor appears at the tap point, splitting the segment into two, with no network call/spinner"
    why_human: "Depends on featuresAtPoint hit-testing against the native GL 'route-segments-hit' layer, which needs a real rendered map"
    result: "FAILED on first device test (2026-07-16): every segment tap fell through to appendAnchor instead of hitting the segment. Root cause: the hit-test layer's line-opacity:0 caused both native maplibre bindings (iOS visibleFeaturesAtPoint, Android queryRenderedFeatures) to exclude it from rendered-feature queries — fully transparent layers are never rendered, so they're never hit-testable. Fixed in route_segment_layer.dart by changing line-opacity to 0.01 (imperceptible, still query-eligible). Re-verified on device (2026-07-16): PASSED."
  - test: "Force a segment into the blocked state (e.g. disconnect network or use out-of-range coordinates), then tap the dashed red segment"
    expected: "A 'Retrying route…' toast appears immediately, followed by either the segment turning routed (solid blue) or, on repeated failure, a 'Still couldn't find a route here...' toast distinct from the first-time 'Couldn't find a route for this segment...' toast"
    why_human: "Requires an actual failing/succeeding Valhalla call and visual toast-copy confirmation; the automated tests exercise the provider logic and grep-confirm the toast strings exist in source, but not the visual dedup timing on a live device"
    result: "FAILED on first device test (2026-07-16) for the same root cause as the segment-insert test above (hit-test layer line-opacity:0) — tapping the blocked segment created a new anchor instead of retrying. Fixed alongside it (route_segment_layer.dart, line-opacity 0.01). Re-verified on device (2026-07-16): PASSED."
  - test: "With 3+ anchors on the route, toggle the auto-routing button OFF, then ON again"
    expected: "Per current shipped behavior: OFF leaves all existing segments visually unchanged (routed segments stay solid blue/routed-looking) and zero network calls fire; ON re-resolves every segment via Valhalla in parallel. Decide whether this matches the intended product behavior, given ROADMAP.md/REQUIREMENTS.md describe OFF as producing 'straight-line segments' for the whole route."
    why_human: "This is the disputed behavior in the Gaps section above — needs a human product decision (fix the code to match the docs, or amend the docs to match the code), not just visual confirmation"
    result: "PASSED (2026-07-16) — confirmed matches the accepted override (OFF leaves existing segments untouched); ROADMAP.md/REQUIREMENTS.md/19-04-PLAN.md already amended to match"
  - test: "Perform several route edits, then tap Undo/Redo app-bar buttons repeatedly"
    expected: "Undo restores the immediately prior anchors+segments snapshot one step at a time; Redo restores what was undone; both buttons are visibly disabled (greyed, non-interactive) when their respective stack is empty; a new edit after an Undo clears the Redo stack"
    why_human: "Visual button-disabled-state and multi-step undo/redo round-tripping is best confirmed interactively even though the underlying state logic is unit-tested"
    result: "PASSED (2026-07-16)"
---

# Phase 19: Route Planner Core — Waypoint Editing & Routing Engine Verification Report

**Phase Goal:** A user can build a route from scratch directly on the map — tapping to add waypoints, dragging to reposition them, inserting mid-segment — with an auto-routing toggle (Valhalla, fixed foot/bike profile set at entry) and undo/redo, all backed by a dedicated route-planner state provider.
**Verified:** 2026-07-16T14:37:14Z
**Status:** passed (1 override accepted)
**Re-verification:** No — initial verification, gap resolved via documented override + doc amendment (ROADMAP.md, REQUIREMENTS.md, 19-04-PLAN.md)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | (WAYP-01/SC1) Tap anywhere on the map adds a route anchor, connected to the previous anchor by a segment | ✓ VERIFIED | `route_planner_screen.dart` `onEvent` fallthrough calls `notifier.appendAnchor(event.point)`; `appendAnchor` in `route_anchor_provider.dart` (lines 230-260) creates the anchor and, if a previous last anchor existed, the connecting segment; covered by passing test `appendAnchor on an empty route creates 1 anchor and 0 segments; a second call creates a 2nd anchor and exactly 1 new segment` |
| 2 | (WAYP-02/SC2) Dragging an anchor re-resolves only its ≤2 adjacent segments; a distant segment is untouched | ✓ VERIFIED | `dragAnchor` (lines 265-300) only iterates `state.segments.where(beforeAnchorId==anchorId || afterAnchorId==anchorId)`; test `dragAnchor moves the target anchor and only its adjacent segments change; a distant, unrelated segment is untouched` passes and asserts `result.segments[1]` (non-adjacent) is `identical` to its pre-drag value |
| 3 | (WAYP-03/SC3) Tapping an existing plain segment inserts a new anchor between its endpoints, geometrically split, no Valhalla call | ✓ VERIFIED | `insertAnchorOnSegment` (lines 307-345) calls `splitSegmentAt` and never `_resolveSegment`; test `insertAnchorOnSegment splits the segment geometrically...issues zero Dio calls` passes; `route_planner_screen.dart` routes non-blocked segment taps here |
| 4 | (ROUTE-01/02/SC4) Toggling auto-routing on re-resolves every existing segment via Valhalla; toggling off leaves existing segments untouched, only new segments go straight | ✓ VERIFIED (override) | `toggleAutoRouting()` (lines 196-212): ON re-resolves every segment in parallel; OFF flips the flag without mutating `state.segments`, zero Dio calls. Matches user-accepted override; ROADMAP.md SC#4, REQUIREMENTS.md ROUTE-02, and 19-04-PLAN.md must_haves.truths amended to match. |
| 5 | (ROUTE-04/SC5 undo) Undo restores the immediately prior snapshot; redo restores what was undone; a new mutation clears redo | ✓ VERIFIED | `undo()`/`redo()` (lines 350-384) implement the immutable-snapshot stack exactly as described; test `undo restores the prior snapshot, redo re-applies it, and a fresh mutation clears the redo stack (D-11)` passes, including empty-stack no-op |
| 6 | (ROUTE-05/SC5 blocked) A segment whose Valhalla call fails is marked blocked, prior polyline preserved, never silently reverted to straight, retryable | ✓ VERIFIED | `_resolveSegment`'s catch branch and `_markBlocked` (lines 144-160) never touch `polyline`, only `state`; test `_resolveSegment failure path marks the segment blocked and leaves its prior polyline unchanged (ROUTE-05)` passes; `retrySegment` re-dispatches and test confirms `blocked` → `routed` transition |
| 7 | An out-of-order (superseded) Valhalla response for a segment is discarded, never applied over a newer request's result | ✓ VERIFIED | Per-segment `CancelToken` + `_generation` counter guard (lines 108-148); test `a stale, superseded dispatch...never corrupts state and never throws uncaught` passes (test scope narrowed from "generation-counter specifically wins" to "observable contract holds", documented and justified in 19-02-SUMMARY.md as an empirically-verified Dio 5.9.2 cancellation-semantics constraint — reasonable, not a gap) |
| 8 | (D-09) Tapping a blocked segment retries instead of inserting; the two are mutually exclusive by segment state | ✓ VERIFIED | `route_planner_screen.dart` `onEvent`: `if (segState == 'blocked') { ...retrySegment... } else { ...insertAnchorOnSegment... }` — `'blocked'` string appears only in the `retrySegment` branch |
| 9 | (D-08) The three segment visual states (routed/straight/blocked) are simultaneously distinguishable by casing + stroke-style + color, never color alone | ✓ VERIFIED | `route_segment_layer.dart`: routed = 5px `#3549bb` + 9px white casing; straight = 3px `#3549bb` at 55% opacity, no casing; blocked = 3px `#EF5350` with `line-dasharray: [2,2]` — three independently distinguishable visual dimensions per state |
| 10 | Segment GeoJSON updates in place via `updateGeoJsonSource` on every mutation — no source remove/re-add flicker | ✓ VERIFIED | `RouteSegmentLayer.update()` branches on `_added`; subsequent calls hit `style.updateGeoJsonSource(id: sourceId, data: data)` only, never `removeSource`/`addSource` again |
| 11 | A wide invisible hit-test layer exists over all segment states so a thin rendered line remains tappable | ✓ VERIFIED | `route-segments-hit` layer added with no `filter`, `line-width: 24`, `line-opacity: 0` — matches every state |
| 12 | (D-02) Route anchors render as numbered markers (1-based, ascending), number always derived from current list order, never stored | ✓ VERIFIED | `route_anchor_layer.dart` line 67: `final number = i + 1;` computed fresh every `build()` from the live `anchors` list; `insertAnchorOnSegment` inserts at the correct list index so downstream numbers shift automatically |
| 13 | Blocked-segment/retry toast copy matches UI-SPEC's Copywriting Contract (first-time error, distinct second-failure copy, transient retrying) | ✓ VERIFIED | `route_planner_screen.dart` contains all three exact strings ("Couldn't find a route for this segment. Tap the dashed line to retry.", "Still couldn't find a route here. Check your connection and try again.", "Retrying route…"), gated by `_blockedNotified`/`_retryAttempted` bookkeeping sets |

**Score:** 13/13 truths verified (1 via accepted override)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/polyline_util.dart` | Precision-parameterized codec | ✓ VERIFIED | `precision` named param on both `decode`/`encode`, default 5; `1E5` literal fully removed (`grep -c '1E5'` = 0) |
| `app/lib/models/route_anchor.dart` | `RouteAnchor`/`RouteSegment`/`SegmentState`/`RouteAnchorsSnapshot`, no "waypoint" string | ✓ VERIFIED | All 4 symbols present via freezed; `grep -ri waypoint` returns nothing |
| `app/lib/provider/route_anchor_provider.dart` | `RouteAnchors` notifier: mutations, routing engine, undo/redo | ✓ VERIFIED | All 7 public methods present (`appendAnchor`, `dragAnchor`, `insertAnchorOnSegment`, `retrySegment`, `toggleAutoRouting`, `undo`, `redo`); wired, tested, `flutter analyze` clean |
| `app/lib/util/route_segment_util.dart` | `segmentKey`, `splitSegmentAt`, `buildSegmentsGeoJson` | ✓ VERIFIED | All 3 functions present and unit-tested (7/7 tests pass) |
| `app/lib/components/map/route_anchor_layer.dart` | Numbered, draggable marker `WidgetLayer` | ✓ VERIFIED | `RouteAnchorLayer` reads `routeAnchorsProvider`, single `dragAnchor` call site inside `onPanEnd` |
| `app/lib/components/map/route_segment_layer.dart` | GeoJSON segment renderer, 3 states + hit-test layer | ✓ VERIFIED | `RouteSegmentLayer` with all 5 layer ids, in-place update, unfiltered hit-test layer |
| `app/lib/routes/route_planner_screen.dart` | Screen wiring every gesture/state to the UI | ✓ VERIFIED | Hosts map, tap disambiguation, toggle, app-bar undo/redo, toast copy — all present and wired |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `route_anchor_provider.dart` | `web/src/routes/api/v1/valhalla/route/+server.ts` | `apiProvider.post('/valhalla/route', ...)` | ✓ WIRED | Confirmed at line 115-126 |
| `route_anchor_provider.dart` | `polyline_util.dart` | `PolylineUtil.decode(shape, precision: 6)` | ✓ WIRED | Confirmed at line 142 |
| `route_anchor_provider.dart` | `route_segment_util.dart` | `splitSegmentAt()` inside `insertAnchorOnSegment` | ✓ WIRED | Confirmed at line 325 |
| `route_anchor_layer.dart` | `route_anchor_provider.dart` | `dragAnchor(...)` from `onPanEnd` | ✓ WIRED | Confirmed at lines 103-107, single call site |
| `route_segment_layer.dart` | `route_segment_util.dart` | `buildSegmentsGeoJson(segments)` feeding `updateGeoJsonSource` | ✓ WIRED | Confirmed at line 39/138 |
| `route_planner_screen.dart` | `route_anchor_provider.dart` | tap routing → `appendAnchor`/`insertAnchorOnSegment`/`retrySegment`; toggle → `toggleAutoRouting`; app bar → `undo`/`redo` | ✓ WIRED | All confirmed present and called in `route_planner_screen.dart` |
| `route_planner_screen.dart` | `route_segment_layer.dart` | `RouteSegmentLayer().update(style, segments)` from `onStyleLoaded` + `ref.listen` | ✓ WIRED | Confirmed at lines 88, 270-276 |
| `route_planner_screen.dart` | `toast_provider.dart` | `toastProvider.notifier.add(ToastMessage(...))` | ✓ WIRED | Confirmed at lines 100-110, 227-235 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Polyline precision codec regression suite | `flutter test test/util/polyline_util_test.dart` | 4/4 passed | ✓ PASS |
| Route-anchor provider (resolution engine, mutations, undo/redo) | `flutter test test/provider/route_anchor_provider_test.dart` | 12/12 passed | ✓ PASS |
| `segmentKey`/`splitSegmentAt`/`buildSegmentsGeoJson` | `flutter test test/util/route_segment_util_test.dart` | 8/8 passed | ✓ PASS |
| Static analysis, all Phase 19 files | `flutter analyze lib/util/polyline_util.dart lib/models/route_anchor.dart lib/provider/route_anchor_provider.dart lib/util/route_segment_util.dart lib/components/map/route_anchor_layer.dart lib/components/map/route_segment_layer.dart lib/routes/route_planner_screen.dart` | No issues found | ✓ PASS |
| Debt-marker scan (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) | `grep -rn` across all 7 phase files | No matches | ✓ PASS |
| "waypoint" terminology leak scan (D-01) | `grep -rni waypoint` across all 6 new/modified route-planner files | No matches | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — no `scripts/*/tests/probe-*.sh` files declared or referenced by this phase's plans/summaries; this is a Flutter mobile-app phase verified via `flutter test`/`flutter analyze`, not a migration/tooling phase with probe scripts.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| WAYP-01 | 19-01, 19-02, 19-04 | Tap map to add a waypoint to the in-progress route | ✓ SATISFIED | `appendAnchor` + `onEvent` fallthrough |
| WAYP-02 | 19-02, 19-03, 19-04 | Drag existing waypoint to reposition, connected segments re-resolve | ✓ SATISFIED | `dragAnchor` + `RouteAnchorLayer.onPanEnd` |
| WAYP-03 | 19-02, 19-03, 19-04 | Tap existing segment to insert a new waypoint between endpoints | ✓ SATISFIED | `insertAnchorOnSegment` + `splitSegmentAt` |
| ROUTE-01 | 19-02, 19-04 | Toggle auto-routing on (Valhalla) or off (straight-line segments for newly-created segments) | ✓ SATISFIED | Toggle exists and correctly gates new-segment behavior; wording amended per accepted override |
| ROUTE-02 | 19-02, 19-04 | Toggling auto-routing on re-resolves all existing segments; toggling off leaves them untouched | ✓ SATISFIED | True for OFF→ON (re-resolve) and ON→OFF (untouched) per accepted override; wording amended |
| ROUTE-04 | 19-02, 19-04 | Undo/redo waypoint add/move/insert/delete/reorder (in-memory) | ✓ SATISFIED | `undo()`/`redo()`, app-bar buttons, disabled-when-empty |
| ROUTE-05 | 19-02, 19-03, 19-04 | Failed segment blocked + retryable, never silently reverts to straight | ✓ SATISFIED | `_resolveSegment`/`_markBlocked`, retry toast copy |

No orphaned requirements found — REQUIREMENTS.md's Phase 19 row (WAYP-01/02/03, ROUTE-01/02/04/05) matches exactly what the four plans collectively declare in their `requirements:` frontmatter.

### Anti-Patterns Found

None. No debt markers (TBD/FIXME/XXX), no TODO/HACK/PLACEHOLDER comments, no empty-return stubs, and no stray "waypoint" terminology in any of the 7 files this phase created/modified.

### Human Verification Required

See `human_verification` in the frontmatter — 6 items, all requiring a real/emulated device because `ml.MapLibreMap`'s native platform channel cannot be constructed in this codebase's widget-test harness (confirmed: no precedent exists for `trail_map.dart`/`trail_create_screen.dart` either). Item 5 additionally required a **product decision**, not just visual confirmation — see Gaps (resolved via accepted override).

**Completed 2026-07-16 — 6/6 PASSED.** Two items failed on first pass and were fixed before re-verification:
- Segment-insert tap and blocked-segment retry (items 3 and 4) both failed identically: the hit-test layer's `line-opacity: 0` made it invisible to both native maplibre bindings' `featuresAtPoint` query (opacity-0 layers are excluded from rendered-feature queries), so every segment tap fell through to `appendAnchor`. Fixed by setting `line-opacity: 0.01` in `route_segment_layer.dart`.
- A related bug surfaced during re-testing item 1 (not originally anticipated): segment lines never rendered on a *second* open of the screen in the same app session. `_segmentLayer` in `route_planner_screen.dart` was `static final`, so its `_added` flag stayed stale across screen exits/re-entries against a fresh native style. Fixed by making it a non-static instance field.

Both fixes are logged in `STATE.md`'s decision log. All 6 items now pass, confirmed by the user.

### Gaps Summary

One gap, rooted in a single design decision made mid-execution of plan 19-02: **auto-routing toggle OFF does not re-resolve existing segments to straight lines**, contradicting ROADMAP.md's Phase 19 Success Criterion #4 and REQUIREMENTS.md's ROUTE-02, both of which explicitly say toggling re-resolves *every existing segment* to the new mode. The corrected behavior — OFF only affects segments created afterward — was deliberately implemented per a mid-execution edit to 19-02-PLAN.md's `must_haves.truths`, and the implementation faithfully matches that corrected plan-level spec (confirmed by direct code read and a passing, purpose-written test). However:

- ROADMAP.md's Success Criterion #4 text was never amended to match (unlike the earlier, explicitly-reconciled ROUTE-03 cut documented in `19-DISCUSSION-LOG.md`).
- REQUIREMENTS.md's ROUTE-02 text was never amended to match.
- 19-04-PLAN.md's own `must_haves.truths` (line 16) still reads "toggling re-resolves every existing segment to the new mode (ROUTE-01/02)" — unreconciled with 19-02's correction — yet 19-04-SUMMARY.md's "Must-Haves Verification" section marks it satisfied by citing only the ON-path behavior, silently narrowing the claim without flagging the gap.

This is a genuine, unresolved conflict between the locked phase contract and the shipped code, not a verification false positive. It needs a human decision: either implement the OFF→straight-line re-resolve for all existing segments (making the code match the currently-written docs), or formally amend ROADMAP.md/REQUIREMENTS.md/19-04-PLAN.md's wording (making the docs match the shipped, arguably more useful, "don't discard prior routing work" behavior) — following the same amend-all-docs precedent already used earlier in this same phase for the ROUTE-03 cut.

**This looks intentional (partially).** To accept the shipped behavior as-is, add to this file's frontmatter:

```yaml
overrides:
  - must_have: "Toggling auto-routing re-resolves every existing segment to match the new mode"
    reason: "OFF now preserves existing segment state/geometry instead of discarding routed polylines; only new segments are straight. Matches 19-02-PLAN.md's corrected must_haves.truths but not yet reflected in ROADMAP.md/REQUIREMENTS.md wording."
    accepted_by: "{your name}"
    accepted_at: "{current ISO timestamp}"
```

Then re-run verification to apply, and separately update ROADMAP.md Phase 19 Success Criterion #4 and REQUIREMENTS.md's ROUTE-01/ROUTE-02 text to match (recommended, for future-phase and future-verifier consistency).

All other observable truths for this phase (WAYP-01/02/03, ROUTE-04, ROUTE-05, and every plan-level must-have around race-guarding, segment visuals, hit-testing, numbering, and toast copy) are verified against actual code and passing tests — not just SUMMARY.md narrative.

---

*Verified: 2026-07-16T14:37:14Z*
*Verifier: Claude (gsd-verifier)*
