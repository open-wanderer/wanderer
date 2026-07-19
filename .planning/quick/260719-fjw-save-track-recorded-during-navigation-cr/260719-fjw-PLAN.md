---
phase: quick-260719-fjw
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/util/recorded_track_util.dart
  - app/test/util/recorded_track_util_test.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/i18n/app_en.arb
  - app/lib/i18n/app_localizations.dart
  - app/lib/i18n/app_localizations_en.dart
autonomous: true
requirements: [QUICK-260719-fjw]
must_haves:
  truths:
    - "After completing a trail, the navigation completion banner shows a Save track action"
    - "When prematurely exiting navigation, the exit dialog offers Save track alongside Cancel and Exit"
    - "Tapping Save track builds a stub Trail from the recorded breadcrumb and opens trail_create_screen with it"
    - "The stub trail carries the recorded GPS track (expand.gpx + expand.gpxData) and correct map bounds so it previews correctly"
    - "Saving ends the navigation session — the resume row is cleared and the navigation screen is left"
  artifacts:
    - path: "app/lib/util/recorded_track_util.dart"
      provides: "buildRecordedTrackTrail — breadcrumb -> stub Trail"
      contains: "buildRecordedTrackTrail"
    - path: "app/test/util/recorded_track_util_test.dart"
      provides: "Unit tests for buildRecordedTrackTrail"
      contains: "buildRecordedTrackTrail"
    - path: "app/lib/routes/navigation_screen.dart"
      provides: "Two Save track trigger points wired to the handoff"
      contains: "buildRecordedTrackTrail"
    - path: "app/lib/i18n/app_en.arb"
      provides: "save_track localization key"
      contains: "save_track"
  key_links:
    - from: "app/lib/routes/navigation_screen.dart"
      to: "buildRecordedTrackTrail"
      via: "helper call in _saveRecordedTrack"
      pattern: "buildRecordedTrackTrail\\("
    - from: "app/lib/routes/navigation_screen.dart"
      to: "/trail/create/edit"
      via: "go_router pushReplacement with the stub trail as extra"
      pattern: "pushReplacement\\('/trail/create/edit'"
    - from: "app/lib/util/recorded_track_util.dart"
      to: "buildDraftTrail"
      via: "reuse of the existing draft-trail builder"
      pattern: "buildDraftTrail\\("
---

<objective>
Let a user save the track they recorded during navigation. "Saving a track" builds a new, unsaved stub `Trail` from the recorded GPS breadcrumb and hands it to `trail_create_screen` (`/trail/create/edit`), where the user decides whether to persist it. The save option is presented at two trigger points:
1. On the navigation **completion banner** (shown when the user arrives at the end of the trail).
2. On the **premature-exit alert dialog** (shown when the user exits navigation before arriving).

Purpose: A hiker who navigates a trail leaves a real recorded track behind. Today that breadcrumb is discarded on screen exit (`NavigationState.breadcrumb` is explicitly session-only). This lets them keep it as a new trail.

Output: A tested pure helper that converts a breadcrumb into a stub `Trail`, plus the two UI trigger wirings in `navigation_screen.dart`.

Investigation findings (the machinery already exists — this plan reuses it, does not reinvent it):
- The recorded track is `navState.breadcrumb` — a `List<ml.Geographic>` (lat/lon only; no elevation or timestamps are recorded per fix). It is read from `navigationProvider(widget.response, resumeManeuverIndex: _resumeManeuverIndex, resumeBreadcrumb: _resumeBreadcrumb)`.
- `buildGpxFromPoints(List<Geographic>)` in `app/lib/util/gpx_util.dart` already builds a minimal `Gpx` (one Trk > Trkseg > Wpt per point).
- `buildDraftTrail(Gpx, {String? category, double? estimatedDurationSeconds})` in `app/lib/util/route_planner_handoff_util.dart` already builds an unsaved in-memory `Trail` from a `Gpx`: it sets `expand.gpxData` (raw XML uploaded on create), `expand.gpx` (parsed preview object), and derives `lat/lon/minLat/maxLat/minLon/maxLon` from the track bounds. It is a pure function with existing unit tests in `app/test/util/route_planner_handoff_util_test.dart`.
- The handoff pattern is: set the `pendingImportedTrail` global (in `app/lib/util/trail_import_util.dart`) then navigate to `/trail/create/edit` with the trail as `extra`. The route builder in `app/lib/provider/router_provider.dart` reads `state.extra as Trail` and constructs `TrailCreateScreen(trail: extra)`, falling back to `pendingImportedTrail` if `extra` is lost across a router refresh. Both the GPX-file import path and the route-planner handoff already use this exact mechanism.
- `TrailCreateScreen({required Trail trail})` accepts a draft trail that carries only a track (no `id`) — confirmed by the existing import and planner handoff call sites.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@app/lib/routes/navigation_screen.dart
@app/lib/provider/navigation_provider.dart
@app/lib/provider/navigation_stats_provider.dart
@app/lib/util/gpx_util.dart
@app/lib/util/route_planner_handoff_util.dart
@app/lib/util/trail_import_util.dart
@app/lib/provider/router_provider.dart
@app/lib/models/trail.dart
@app/test/util/route_planner_handoff_util_test.dart
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Pure helper buildRecordedTrackTrail (breadcrumb -> stub Trail) with unit tests</name>
  <files>app/lib/util/recorded_track_util.dart, app/test/util/recorded_track_util_test.dart</files>
  <behavior>
    - Returns null when the breadcrumb has fewer than 2 points (empty or single fix — nothing meaningful to save).
    - For a >=2-point breadcrumb, returns a Trail whose expand.gpx.allPoints equals the breadcrumb points in order (lat/lon preserved).
    - The returned Trail has a non-empty expand.gpxData containing "<gpx" (the raw XML uploaded on create).
    - The returned Trail has expand.waypointsViaTrail empty (a recorded track has no named waypoints).
    - lat/lon and the min/max bounds are derived from the track (non-null lat/lon, bounds bracket the input points).
    - When durationSeconds is provided, the returned Trail.duration equals it; when omitted, duration is 0.
  </behavior>
  <action>
Create `app/lib/util/recorded_track_util.dart` exporting one pure top-level function:
`Trail? buildRecordedTrackTrail(List<Geographic> breadcrumb, {double? durationSeconds})`.

Imports: `Geographic` from `package:maplibre/maplibre.dart`; `Trail` from `package:wanderer/models/trail.dart`; `buildGpxFromPoints` from `package:wanderer/util/gpx_util.dart`; `buildDraftTrail` from `package:wanderer/util/route_planner_handoff_util.dart`.

Logic: if `breadcrumb.length < 2` return null (the two callers treat null as "nothing recorded — just leave navigation"). Otherwise build the Gpx with `buildGpxFromPoints(breadcrumb)` and return `buildDraftTrail(gpx, estimatedDurationSeconds: durationSeconds)`. Do NOT pass a category — a recorded track has no travel profile, and `buildDraftTrail` accepts a null category.

Document in the file's doc comment WHY this is thin and reuses `buildDraftTrail`: the breadcrumb carries lat/lon only (no per-fix `ele` or `time` — see `NavigationState.breadcrumb`), so the resulting track has no elevation; `duration` is pre-filled from the recorded time-in-motion so the create-screen preview shows it, and the server recomputes distance from the uploaded track on save. Reusing `buildDraftTrail` keeps the bounds/`expand.gpxData`+`expand.gpx` derivation single-sourced with the route-planner and GPX-import handoffs (a draft that sets only one of gpxData/gpx saves with no track or previews wrong — see `buildDraftTrail`'s own doc comment).

Create `app/test/util/recorded_track_util_test.dart` following the structure of `app/test/util/route_planner_handoff_util_test.dart` (import `package:flutter_test/flutter_test.dart`, `package:maplibre/maplibre.dart`, the model, and the new util). Cover every case in the behavior block: null for `const []` and for a single-point list; a 3-point fixture asserting `expand.gpx` allPoints round-trip, `expand.gpxData` non-empty and contains "<gpx", `expand.waypointsViaTrail` empty, non-null lat/lon within the fixture's coordinate range, and `duration` equals a supplied `durationSeconds` (and 0 when omitted).
  </action>
  <verify>
    <automated>cd app && flutter test test/util/recorded_track_util_test.dart</automated>
  </verify>
  <done>buildRecordedTrackTrail exists as a pure function; all unit test cases pass; null-guard, track round-trip, bounds, and duration pre-fill are all asserted.</done>
</task>

<task type="auto">
  <name>Task 2: Wire Save track into the completion banner and the premature-exit dialog</name>
  <files>app/lib/routes/navigation_screen.dart, app/lib/i18n/app_en.arb, app/lib/i18n/app_localizations.dart, app/lib/i18n/app_localizations_en.dart</files>
  <action>
**Localization first.** Add a new key `"save_track": "Save track"` to `app/lib/i18n/app_en.arb` (place it near the existing `"save"` key; keep valid JSON — a trailing comma on the preceding line). Regenerate the localization sources by running `cd app && flutter gen-l10n`. This rewrites `app/lib/i18n/app_localizations.dart` and the per-locale files, adding `String get save_track`; other locales fall back to the English template value (gen-l10n emits an untranslated-message warning — that is expected and acceptable, matching how prior keys like `resume_navigation_prompt` were added). Do not hand-edit the generated `app_localizations*.dart` files beyond what gen-l10n produces.

**Wiring in `app/lib/routes/navigation_screen.dart`.**

Add imports: `package:wanderer/util/recorded_track_util.dart` (for `buildRecordedTrackTrail`) and `package:wanderer/util/trail_import_util.dart` (for the `pendingImportedTrail` global). `go_router` is already imported.

Add a private method `void _saveRecordedTrack()` on `_NavigationScreenState`:
- Read the current navigation state with the IDENTICAL family seed args used everywhere else in this file — `ref.read(navigationProvider(widget.response, resumeManeuverIndex: _resumeManeuverIndex, resumeBreadcrumb: _resumeBreadcrumb))` — to get `.breadcrumb`. Using different seed args resolves a different (split-brain) provider instance; this is called out repeatedly in this file.
- Read `ref.read(navigationStatsProvider(widget.response, resume: _resumeStats)).elapsed` for the recorded duration.
- Call `buildRecordedTrackTrail(navState.breadcrumb, durationSeconds: stats.elapsed.inSeconds.toDouble())`.
- Saving ends the session either way: call `active_nav.clear(_store)` (same deliberate-exit cleanup `_confirmExit` already does, so no stale resume prompt appears on next launch).
- If the helper returned null (fewer than 2 recorded fixes — nothing to save), just leave navigation: guard `if (context.mounted) context.pop();` and return.
- Otherwise set `pendingImportedTrail = trail;` then `if (context.mounted) context.pushReplacement('/trail/create/edit', extra: trail);`. Use `pushReplacement` (NOT `push`) so the finished/abandoned navigation screen is removed from the stack — otherwise backing out of the create screen would return the user to a live, disposed-on-exit navigation screen. `pushReplacement` disposing this screen also stops GPS/tracelet via the existing `dispose()`.

**Trigger 1 — completion banner.** In `_buildCompletionBannerContent`, add a full-width Save track action below the existing "you have arrived / reached end of trail" text (wrap the current Row plus the button in a Column so the button sits beneath the message). Use a button labeled `localizations.save_track` (a `FilledButton` or `FilledButton.icon` with a save/download icon, executor's discretion for exact widget to match the app's style) whose `onPressed` calls `_saveRecordedTrack`. `_buildCompletionBannerContent` currently takes only `(context, localizations)` — that is sufficient; `_saveRecordedTrack` reads breadcrumb/stats from providers itself.

**Trigger 2 — premature-exit dialog.** Rework `_confirmExit` so its dialog offers three actions instead of two. Introduce a small private enum `enum _NavExitChoice { cancel, exit, saveTrack }` (top-level in this file). Change `showDialog<bool>` to `showDialog<_NavExitChoice>`; the three `TextButton`s pop `_NavExitChoice.cancel` (label `localizations.cancel`), `_NavExitChoice.exit` (label `localizations.exit_navigation`), and `_NavExitChoice.saveTrack` (label `localizations.save_track`). In the `.then` callback, switch on the choice: `saveTrack` -> call `_saveRecordedTrack()`; `exit` -> keep the EXISTING behavior verbatim (`active_nav.clear(_store)` then `if (context.mounted) context.pop();`); `cancel` (and null, from a barrier dismiss) -> do nothing. Preserve the existing `PopScope`/back-button entry into `_confirmExit` unchanged.
  </action>
  <verify>
    <automated>cd app && flutter gen-l10n && grep -q "String get save_track" lib/i18n/app_localizations.dart && grep -c "buildRecordedTrackTrail(" lib/routes/navigation_screen.dart && grep -q "pushReplacement('/trail/create/edit'" lib/routes/navigation_screen.dart && flutter analyze lib/routes/navigation_screen.dart lib/util/recorded_track_util.dart</automated>
    <human-check>On device: (1) navigate a trail to the end — the completion banner shows Save track; tap it and confirm trail_create_screen opens with the recorded route drawn on the map. (2) Start navigation, tap exit before arriving — the dialog shows Cancel / Exit / Save track; Save track opens trail_create_screen with the recorded route; Exit still just leaves; Cancel dismisses.</human-check>
  </verify>
  <done>`save_track` getter exists in generated localizations; both trigger points call `_saveRecordedTrack`; `pushReplacement('/trail/create/edit', ...)` is present; `flutter analyze` reports no issues for the two changed source files; the existing Exit/Cancel and back-button behavior is unchanged.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| navigation_screen -> trail_create_screen | Recorded breadcrumb (user's own in-memory GPS fixes) is serialized to GPX and handed to the existing create screen. No new external input crosses here. |
| trail_create_screen -> server (on user save) | Existing, unchanged upload path — the same one GPX-file import and the route planner already feed. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-fjw-01 | Tampering | GPX XML built from the breadcrumb | accept | XML is produced structurally via `GpxWriter().asString` inside the reused `buildDraftTrail` (no string concatenation of coordinates); coordinates originate from the device's own GPS, not an external party. |
| T-fjw-02 | Information disclosure | Recorded track handed to create screen | accept | The draft is only ever surfaced to the same authenticated user in their own create screen; nothing is transmitted until the user explicitly saves via the existing (already-authorized) upload path. No privacy boundary change. |

No package installs, external services, or new network calls are introduced — this is a Flutter-local change reusing existing app machinery.
</threat_model>

<verification>
- `cd app && flutter test test/util/recorded_track_util_test.dart` passes (Task 1).
- `cd app && flutter analyze lib/routes/navigation_screen.dart lib/util/recorded_track_util.dart` reports no issues.
- `grep "String get save_track" app/lib/i18n/app_localizations.dart` matches (localization regenerated).
- `grep "buildRecordedTrackTrail(" app/lib/routes/navigation_screen.dart` and `grep "pushReplacement('/trail/create/edit'" app/lib/routes/navigation_screen.dart` both match (both trigger points wired).
- Human on-device check per the Task 2 `<human-check>` — both trigger points open trail_create_screen with the recorded route; existing Exit/Cancel/back behavior unchanged.
</verification>

<success_criteria>
- A user who finishes a trail can tap Save track on the completion banner and lands in trail_create_screen with a stub trail carrying their recorded route.
- A user who exits navigation early is offered Save track (alongside Cancel and Exit) and, on choosing it, lands in trail_create_screen with the recorded route.
- The stub trail previews correctly (bounds + track drawn) and can be saved to the database through the unchanged create flow.
- Choosing Save track ends the navigation session (resume row cleared, navigation screen replaced).
- Existing exit-without-saving, cancel, and back-button behavior is unchanged; no new packages or network calls introduced.
</success_criteria>

<output>
Create `.planning/quick/260719-fjw-save-track-recorded-during-navigation-cr/260719-fjw-SUMMARY.md` when done.
</output>
