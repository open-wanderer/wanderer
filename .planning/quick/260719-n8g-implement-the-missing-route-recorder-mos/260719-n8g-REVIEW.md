---
phase: 260719-n8g-implement-the-missing-route-recorder-mos
reviewed: 2026-07-19T15:23:29Z
depth: quick
files_reviewed: 6
files_reviewed_list:
  - app/lib/routes/navigation_screen.dart
  - app/lib/provider/router_provider.dart
  - app/lib/provider/router_provider.g.dart
  - app/lib/routes/trail_source_select_screen.dart
  - app/lib/main.dart
  - app/test/provider/navigation_provider_test.dart
findings:
  critical: 1
  warning: 6
  info: 1
  total: 8
status: issues_found
---

# Phase 260719-n8g: Code Review Report

**Reviewed:** 2026-07-19T15:23:29Z
**Depth:** quick (pattern scan found nothing; escalated to a full read of each file to verify actual behavior — see Summary)
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Pattern-matching (secrets, `eval`/`exec`, empty catch, TODO/FIXME/debug prints) came back clean across all six files. However, a full read of the changed files — required to actually judge whether the "trail-less GPS recording" feature (`isRecording` / `/record` route) behaves correctly — turned up one significant correctness bug in the core recording feature and several robustness/quality issues.

The most important finding: **the pause button does not pause GPS breadcrumb recording.** `NavigationStatsNotifier` (stats: time/distance/elevation/speed) correctly freezes on `isPaused`/`isStationary`, but the separate `Navigation` notifier that owns `breadcrumb` — the data structure that is persisted to disk and later converted into the saved trail's GPX — has no pause gating at all, and `_NavigationScreenState`'s single GPS listener calls both notifiers unconditionally. Users who pause a recording (e.g., at a rest stop, getting into a car) will get a saved trail that silently includes the "paused" segment, while the UI displays a frozen stopwatch/distance that suggests otherwise. This is a functional regression against the reasonable, implied contract of a "Pause" button in a route recorder.

I also traced two async-flow files (`trail_source_select_screen.dart` for the new recorder entry point) end-to-end for `mounted`/re-entrancy safety, since that's exactly the kind of gap grep can't find — and found the new `_openRecorder` flow is missing guards its sibling flows (`_openPlanner`, `_importGpx`) already have.

## Critical Issues

### CR-01: Pausing a recording does not stop the GPS breadcrumb from growing — saved track includes "paused" segments

**File:** `app/lib/routes/navigation_screen.dart:343-374`
**Issue:** The single position-stream listener set up in `initState` unconditionally feeds every GPS fix into `navigationProvider(...).notifier.onPosition(...)` (which appends to `NavigationState.breadcrumb`) regardless of whether the session is paused:

```dart
_sub = _positionStream.listen(
  (pos) {
    final navProviderInstance = navigationProvider(
      widget.response,
      resumeManeuverIndex: _resumeManeuverIndex,
      resumeBreadcrumb: _resumeBreadcrumb,
    );
    final beforeIndex = ref.read(navProviderInstance).currentManeuverIndex;
    ref
        .read(navProviderInstance.notifier)
        .onPosition(...)               // <-- no pause/stationary check
    ...
```

Confirmed by reading `app/lib/provider/navigation_provider.dart:114-127` — `Navigation.onPosition` has zero awareness of pause state and always does `breadcrumb: [...state.breadcrumb, Wpt(...)]`. Meanwhile `NavigationStatsNotifier.onPosition` (`app/lib/provider/navigation_stats_provider.dart:175-182`) *does* correctly early-return `if (state.isPaused || state.isStationary)`.

Since `_saveRecordedTrack()` builds the exported GPX directly from `navState.breadcrumb` (`navigation_screen.dart:682`, `buildGpxFromPoints(navState.breadcrumb)`), and `_persistNow()` persists this same breadcrumb to disk every 10s and on maneuver advance, every GPS fix received while the user believes recording is paused (stats frozen, pause icon shown) is still silently written into the trail that eventually gets saved. This directly undermines the core value of the just-added trail-less recording feature.

**Fix:** Gate the breadcrumb-affecting call on the same frozen condition the stats notifier already uses, e.g.:
```dart
_sub = _positionStream.listen((pos) {
  final stats = ref.read(navigationStatsProvider(widget.response, resume: _resumeStats));
  if (stats.isPaused || stats.isStationary) return; // don't record breadcrumb while frozen
  final navProviderInstance = navigationProvider(...);
  ...
});
```
(or push the same `isPaused`/`isStationary` flags into `Navigation.onPosition` itself, since that provider is the one that owns persistence/GPX-export semantics and currently has no knowledge of pause at all).

## Warnings

### WR-01: New recorder entry point (`_openRecorder`) is missing `mounted` guards present in its sibling flows

**File:** `app/lib/routes/trail_source_select_screen.dart:37-75`
**Issue:** `_openPlanner` and `_importGpx` in the same class both check `if (!mounted) return;` after every `await` before touching `ref`/`context`. `_openRecorder` does not — it calls `showError(...)` (which does `ref.read(toastProvider.notifier)`) after three separate `await` points (`Geolocator.isLocationServiceEnabled()`, `Geolocator.checkPermission()`, `Geolocator.requestPermission()`) with no `mounted` check until the very last line. If the widget is disposed mid-flow (user navigates away while the permission dialog is up), `ref` is used after disposal.
**Fix:** Add `if (!mounted) return;` immediately after each `await` before calling `showError` or `context.push`, mirroring `_openPlanner`'s pattern.

### WR-02: `_openRecorder` has no re-entrancy/loading guard, unlike the other two source actions

**File:** `app/lib/routes/trail_source_select_screen.dart:177-185` (card wiring), `:37-75` (handler)
**Issue:** `_openPlanner` guards with `if (_plannerLoading) return;` and disables its card via `isLoading: _plannerLoading`; `_importGpx` guards with `if (_importLoading) return;`. `_openRecorder` has no equivalent flag — the "Record" card's `onTap` stays enabled for the whole permission-request round trip, so a rapid double-tap can invoke `_openRecorder` twice concurrently, potentially double-pushing `/record` (two concurrent `NavigationScreen`s racing on the same ObjectBox active-session row).
**Fix:** Add a `_recorderLoading` flag (or reuse the existing loading-disable pattern) and short-circuit re-entry the same way the other two actions do.

### WR-03: Card descriptions bypass i18n while their titles use it

**File:** `app/lib/routes/trail_source_select_screen.dart:170, 181, 191`
**Issue:** All three `_SourceActionCard`s correctly localize `title` (`l10n.trail_source_planner`, `l10n.trail_source_record`, `l10n.trail_source_import`) but hardcode the `description` as a literal English string, e.g.:
```dart
description: "Track your live coordinates and log your journey in real-time.",
```
This is inconsistent within the same widget and will not translate for any non-English `supportedLocales` the app ships (`main.dart` wires up `AppLocalizations.supportedLocales`).
**Fix:** Add localization keys (e.g. `l10n.trail_source_record_description`) and use them instead of literal strings.

### WR-04: Null elevation is collapsed to `0.0` before persisting, then re-read as a real value on resume

**File:** `app/lib/routes/navigation_screen.dart:612-622` (persist), `:268-286` (resume decode)
**Issue:** `_persistNow()` writes `elevations: navState.breadcrumb.map((wpt) => wpt.ele ?? 0.0).toList()`. On the next resume, `initState` reads that same list back with no way to distinguish "was actually recorded as 0.0" from "was null and defaulted" — the very defensive-null-handling the resume code has (`i < resumeElevations.length ? resumeElevations[i] : null`) is defeated because the persisted array is always the same length and never contains `null` after the first save. Any breadcrumb point whose altitude reading was unavailable is permanently recorded as sea level once persisted, corrupting the eventual GPX elevation profile for a resumed session.
**Fix:** Persist elevation as nullable (e.g. a sentinel or a parallel "has elevation" bitset) instead of coalescing to `0.0`, so a genuinely-missing altitude round-trips as missing rather than as a fabricated value.

### WR-05: `_saveRecordedTrack()` clears resumable state and mutates global before confirming the widget is still mounted

**File:** `app/lib/routes/navigation_screen.dart:667-709`
**Issue:**
```dart
active_nav.clear(_store);
pendingImportedTrail = trail;
if (context.mounted) {
  context.pushReplacement('/trail/create/edit', extra: trail);
}
```
`active_nav.clear(_store)` (destroying the persisted "resume this session" row) and the assignment to the module-global `pendingImportedTrail` both run unconditionally, before the `context.mounted` check that gates the actual navigation. If the screen is unmounted during the preceding `await buildDraftTrail(...)` (e.g. app backgrounded/killed, or the user otherwise left the screen), the converted trail is silently dropped — the resumable row is gone and the hand-off navigation never happens — and `pendingImportedTrail` is left set globally, where the `/trail/create/edit` route's fallback (`router_provider.dart:301-306`) could later surface it during an unrelated navigation (e.g. a subsequent GPX import) that happens to hit that route without `extra`.
**Fix:** Only clear `active_nav` / set `pendingImportedTrail` after confirming `context.mounted`, or restructure so the row is retained until the hand-off actually completes.

### WR-06: `_saveRecordedTrack()` swallows all exceptions silently, unlike every other catch block in this file

**File:** `app/lib/routes/navigation_screen.dart:695-705`
**Issue:**
```dart
} catch (_) {
  if (!context.mounted) return;
  ref.read(toastProvider.notifier).add(ToastMessage(...));
}
```
Every other `catch` in this file (`_startHeadingSub`'s `onError`, `_composeStyle`, `_onStyleLoaded`, breadcrumb `updateGeoJsonSource`'s `catchError`) logs via `debugPrint('NavigationScreen: ... — $e')`. This one discards the error entirely, making it impossible to diagnose why a "Save track" conversion failed (network error vs. malformed GPX vs. server rejection) from logs.
**Fix:** `catch (e) { debugPrint('NavigationScreen: save recorded track failed — $e'); ... }`.

## Info

### IN-01: Commented-out debug code left in `main.dart`

**File:** `app/lib/main.dart:37`
**Issue:** `// store.box<TrailEntity>().removeAll();` is a leftover local-debug line (wipes cached trails on every launch) committed into `main()`.
**Fix:** Remove the line, or if it's needed for manual debugging, gate it behind a `kDebugMode`/flag rather than leaving it commented directly in the entry point.

---

_Reviewed: 2026-07-19T15:23:29Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
