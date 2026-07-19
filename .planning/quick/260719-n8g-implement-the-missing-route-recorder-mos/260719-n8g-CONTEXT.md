# Quick Task 260719-n8g: Implement the missing route recorder - Context

**Gathered:** 2026-07-19
**Status:** Ready for planning

<domain>
## Task Boundary

Implement the missing "route recorder" (pure GPS track recording with no original/target trail to follow). Recording is essentially navigation minus the maneuvers/original trail: reuse navigation_screen.dart, its providers (navigation_provider.dart breadcrumb collection, navigation_stats_provider.dart timer/motion/stats), and the existing save-to-trail-create-screen handoff (`buildDraftTrail` in route_planner_handoff_util.dart, wired today via `_saveRecordedTrack` in navigation_screen.dart:657-699) rather than building new plumbing.

Current state (confirmed via code exploration, not assumption):
- `trail_source_select_screen.dart:142-150` already has a "Record trail" `_SourceActionCard` (l10n `trail_source_record`) whose `onTap` calls `_comingSoon()` — pure dead-end stub, sits alongside the Route Planner / GPX-import cards.
- `navigation_screen.dart` requires a non-null `NavigateResponse response` and non-null `id` (trail id) in its constructor (`navigation_screen.dart:51-63`), but:
  - Breadcrumb collection (`Navigation.onPosition`, `navigation_provider.dart:122-127`) appends unconditionally, before any route-matching — works with an empty/stub response.
  - `NavigationStatsNotifier` (`navigation_stats_provider.dart`) is keyed on `response` only as a family cache key; timer/pause/motion accumulation is 100% GPS-fix-driven.
  - Maneuver-dependent UI already null/empty-guards to nothing (`_buildActiveBannerContent` returns `SizedBox.shrink()` when `maneuvers.isEmpty`, `navigation_screen.dart:1229-1231`).
  - `isArrived` (`navigation_screen.dart:930-931`) is keyed on `currentIndex >= maneuvers.length - 1 && maneuvers.isNotEmpty` — **always false** with empty maneuvers, so the existing auto-arrival completion trigger never fires for a recording session. This is why a manual finish/stop action is required (see decisions below).
  - Map polyline/waypoints/elevation-profile pull from `ref.watch(trailProvider(widget.id))` (a real fetched trail, separate from `widget.response`) and are all null-guarded already.
- `_saveRecordedTrack` (`navigation_screen.dart:657-699`) already builds a stub `Trail` via `buildGpxFromPoints(breadcrumb)` + `buildDraftTrail(...)` and hands off via `pushReplacement` to `/trail/create/edit` — this exact mechanism is what recording-mode's "save" must reuse; do not re-derive it.
- `ActiveNavigationEntity` (`active_navigation_entity.dart:6-8, 29, 38-39, 43`) already defines `enum ActiveSessionType { nav, rec }` with `rec` explicitly documented as reserved for this feature, and marks `breadcrumbPolyline`/`elevations`/`timestampsUtc`/stats fields as session-agnostic. `main.dart:172-192` (`_maybeResume`) currently only resumes `ActiveSessionType.nav` rows.

</domain>

<decisions>
## Implementation Decisions

### Entry point
- Wire the existing stub: `trail_source_select_screen.dart`'s "Record trail" card becomes the real entry point. Do not add a new/separate UI surface.

### Screen reuse strategy
- Reuse `NavigationScreen` itself in an empty-response "recording mode" (push it with a stub/empty `NavigateResponse` — no maneuvers/shape — and a sentinel/placeholder id), rather than building a separate `RecordScreen` wrapper. Maneuver banner, route polyline, and turn-by-turn UI already degrade to nothing; only the arrival/completion trigger and app-bar/title/button-row need a recording-mode branch.

### Bottom button row (recording mode) — left to right:
1. **Pause/unpause toggle** — same behavior as nav mode's center pause FAB (togglePause on the stats notifier).
2. **Stop recording** — red background, square icon, replaces both nav mode's left "exit" (X) button position ordering AND its center position semantics: this is the finish trigger. Tapping opens a dialog with three options: **Cancel** / **Exit without saving** / **Save**. This directly reuses the existing 3-option exit-dialog machinery (`_confirmExit`/`_NavExitChoice`-equivalent pattern in `navigation_screen.dart`) already built for the premature-exit case — recording mode always uses this 3-option dialog as its *only* finish path (there is no separate auto-arrival completion banner, since `isArrived` can never be true).
3. **Elevation profile toggle** — unchanged, same as nav mode's right button.

(Note: order changes from nav mode's [exit, pause, elevation] to recording mode's [pause, stop, elevation] — pause moves from center to left, stop takes the center "dominant" slot.)

### Resume-after-kill
- Wire `ActiveSessionType.rec` resume too. Extend `main.dart`'s `_maybeResume` (currently `nav`-only, `main.dart:172-192`) to also detect and resume an in-progress `.rec` session, matching the UX already shipped for `.nav` resume (quick task 260712-m9v). Do not defer this to a follow-up.

### Claude's Discretion
- Exact sentinel/placeholder value for `widget.id` and the shape of the "empty" `NavigateResponse` passed to `NavigationScreen` when entering recording mode (e.g. a real empty-shape/empty-maneuvers response vs. a dedicated recording constructor path) — pick whichever requires the least disruption to `NavigationScreen`'s existing non-null-required fields.
- Whether recording mode needs its own app-bar title/copy change (e.g. "Recording" vs. trail name) — not discussed, use judgment matching the rest of the screen's copy conventions.
- Exact resume-dialog copy/behavior differences (if any) between a resumed `.nav` session and a resumed `.rec` session — mirror the existing `.nav` resume dialog's structure unless the `rec` case clearly needs different wording (e.g. no maneuver/trail-name context to show).

</decisions>

<specifics>
## Specific Ideas

No additional specific requirements beyond the decisions above — the explicit intent is maximum reuse of navigation_screen.dart/navigation_provider.dart/navigation_stats_provider.dart/route_planner_handoff_util.dart, with recording mode as a thin variant, not a parallel implementation.

</specifics>

<canonical_refs>
## Canonical References

- Prior quick task `260719-fjw` (save-track-during-navigation) — establishes the save-to-trail-create-screen handoff pattern this task must reuse (superseded internally by `buildDraftTrail` directly per commits `ca063023`/`c521149e`, but the handoff route/mechanism is unchanged).
- Prior quick task `260712-m9v` (resume navigation after app termination) — establishes the `ActiveNavigationEntity`/resume-dialog pattern that `.rec` resume should mirror.
- `active_navigation_entity.dart` — `ActiveSessionType.rec` doc-comments describe the intended reserved shape for this exact feature.

</canonical_refs>
