---
phase: 26-trail-download-guard
reviewed: 2026-07-24T12:02:57Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - app/lib/util/trail_coverage_util.dart
  - app/test/util/trail_coverage_util_test.dart
  - app/lib/components/trail/missing_coverage_sheet.dart
  - app/lib/services/download_notification_service.dart
  - app/lib/provider/trail/trail_download_state_provider.dart
findings:
  critical: 2
  warning: 4
  info: 3
  total: 9
status: issues_found
---

# Phase 26: Code Review Report

**Reviewed:** 2026-07-24T12:02:57Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

`trail_coverage_util.dart` is a clean, well-tested pure module — `bboxesOverlap`/`overlappingRegions`/`missingCoverageRegions` are correct and the accompanying test file covers the documented edge cases (degenerate bbox, edge-touch, fully-covered, no-region-gap, `updateAvailable` exclusion). The two remaining new files are less solid:

- `trail_download_state_provider.dart`'s `download()` has a real state-corruption bug: an `await` that can throw sits **outside** the `try/finally` that clears `DownloadingTrailIds.state`, so an exception there permanently strands a trail in the "downloading" set (the provider is `keepAlive: true`, so this persists for the whole app session). It also never re-invalidates `regionListNotifierProvider` after triggering region downloads, contradicting that provider's own documented invalidation contract and undermining the guard's stated goal that already-downloaded regions must not re-trigger the sheet.
- `missing_coverage_sheet.dart` and `download_notification_service.dart` are functionally sound but have several smaller robustness/consistency gaps (unawaited notification calls, an `onGeneratingChanged` path that ignores the new aggregate-notification contract, a shared single notification ID with no per-trail isolation, and a mix of localized/hardcoded strings within one widget).

## Critical Issues

### CR-01: Exception before `try` permanently strands a trail in the "downloading" state

**File:** `app/lib/provider/trail/trail_download_state_provider.dart:146-166`
**Issue:** `state = {...state, trail.id}` is set optimistically at the top of `download()` (line 30), and the *only* code that ever removes it is the `finally` block at lines 185-187, which is attached to the `try` that starts at line 152. But the trail-only branch's notification call runs **before** that `try`:

```dart
if (hasSelectedPackages) {
  updateAggregate();
} else {
  await notificationService.showProgress(trail.name, 0, 0);   // <-- outside try/finally
}

try {
  await trailDownloadService.downloadTrail( ... );
} finally {
  state = {...state}..remove(trail.id);
}
```

`showProgress` calls `_ensureInitialized()`, which calls `_plugin.initialize(...)` and (on Android) `requestNotificationsPermission()` — both can throw (plugin not yet attached, platform channel error, permission API failure, etc.). If that happens, the exception propagates straight out of `download()`, the `finally` block is never reached, and `trail.id` stays in `DownloadingTrailIds.state` forever (the provider is `@Riverpod(keepAlive: true)`). Every download entry point for that trail (detail screen button, dropdown item) is permanently disabled until the app is restarted, with no error toast and no way for the user to recover.

The same class of bug applies to any other statement between the coverage guard and the `try` (e.g. `ref.container.listen(...)`, provider reads) — none of it is covered by the cleanup path.

**Fix:** Wrap the whole post-guard body in `try { ... } finally { state = {...state}..remove(trail.id); }`, not just the `trailDownloadService.downloadTrail` call — e.g. move the `try` up to right after the guard section, and keep only the trail-download-specific success/error notifications in an inner `try/catch`:

```dart
try {
  final trailDownloadService = ref.read(trailDownloadServiceProvider);
  final notificationService = ref.read(downloadNotificationServiceProvider);
  // ...
  if (hasSelectedPackages) {
    updateAggregate();
  } else {
    await notificationService.showProgress(trail.name, 0, 0);
  }

  try {
    await trailDownloadService.downloadTrail(...);
    await notificationService.showSuccess(trail.name);
    // ...
  } catch (e) {
    await notificationService.showError(trail.name);
    // ...
  }
} finally {
  state = {...state}..remove(trail.id);
}
```

### CR-02: Region downloads triggered by the guard never invalidate `regionListNotifierProvider`

**File:** `app/lib/provider/trail/trail_download_state_provider.dart:95-198` (no `ref.invalidate(regionListNotifierProvider)` anywhere in the file)
**Issue:** `region_provider.dart`'s doc comment for `RegionListNotifier` is explicit: "No mutation methods live here; all region mutations flow through `TileRepositoryStatus` ..., **whose callers must `ref.invalidate(regionListNotifierProvider)` after each mutation**." `download()` is exactly such a caller — it invokes `tileRepoNotifier.downloadVector(region.id)` / `downloadDem(region.id)` (lines 105-107) and awaits their completion via `Future.wait(regionFutures)` (line 194) — but it never invalidates `regionListNotifierProvider` afterwards, unlike the only other caller of the same methods, `settings_offline_regions_screen.dart` (which calls `ref.invalidate(regionListNotifierProvider)` after every `downloadVector`/`downloadDem`/`delete`, e.g. lines 661 and 675).

Because `RegionEntity.status` is a getter over a cached `ToOne.target` (documented in this same file's Pitfall 2 in `26-RESEARCH.md`), any `RegionListNotifier` instance that is still alive when the region download finishes (e.g. because the Settings/Offline Regions screen is mounted elsewhere, or because a later `ref.read` in this same provider reuses a live instance) keeps reporting the just-downloaded region as `notDownloaded`/`missing`. A subsequent `download()` call for the same or another trail overlapping that region will needlessly re-show the missing-coverage sheet for a region the user just finished downloading through this exact flow — directly contradicting the guard's own stated goal ("a merely-stale region never re-fires the guard", GUARD-04) and the codebase's own documented invalidation contract.

**Fix:** After the region futures settle, invalidate the snapshot the same way the existing Settings screen does:

```dart
if (regionFutures.isNotEmpty) {
  try {
    await Future.wait(regionFutures);
  } catch (_) {
    // Isolated from the trail download above; nothing to do here.
  } finally {
    ref.invalidate(regionListNotifierProvider);
  }
}
```

## Warnings

### WR-01: `onGeneratingChanged` ignores the aggregate-notification contract

**File:** `app/lib/provider/trail/trail_download_state_provider.dart:155-156`
**Issue:** `onProgress` correctly branches on `hasSelectedPackages` to decide between the aggregate notification and the plain trail notification (lines 158-164), but `onGeneratingChanged` does not:

```dart
onGeneratingChanged: (isGenerating) {
  if (isGenerating) notificationService.showGenerating(trail.name);
},
```

This always calls the single-source `showGenerating(trail.name)`, which shares notification id 42 with `showAggregateProgress`. When `hasSelectedPackages` is true and the trail enters the server-side "generating tiles" phase, the notification content briefly reverts to just the trail name / "Generating map tiles...", discarding the D-10 aggregate title/body/item-count context described in the code's own comments, until the next region-progress event (via `aggregateSub`) or `onProgress` call overwrites it again.
**Fix:** Branch the same way `onProgress` does, e.g. add an aggregate-aware "generating" message, or simply skip calling `showGenerating` when `hasSelectedPackages` is true and let `updateAggregate()` continue to own the notification.

### WR-02: Unawaited notification calls have no error handling

**File:** `app/lib/provider/trail/trail_download_state_provider.dart:127-133` (`updateAggregate`'s `notificationService.showAggregateProgress(...)`), `:156` (`showGenerating`), `:163` (`showProgress`)
**Issue:** These calls are fire-and-forget (`Future<void>` return value discarded, no `await`, no `.catchError`/`try-catch`). If the plugin throws (permission errors, platform channel failures), the rejected `Future` becomes an unhandled async error — at best silently swallowed, at worst surfaced as an uncaught-exception log/crash report unrelated to the actual download outcome, and impossible to distinguish from a real download failure.
**Fix:** Wrap each fire-and-forget notification call, e.g. `notificationService.showAggregateProgress(...).catchError((_) {});`, or centralize notification error handling inside `DownloadNotificationService` itself so callers don't need to think about it.

### WR-03: Single shared notification id has no per-trail isolation

**File:** `app/lib/services/download_notification_service.dart:6` (`_notificationId = 42`), consumed by `app/lib/provider/trail/trail_download_state_provider.dart`
**Issue:** Every notification method (`showProgress`, `showAggregateProgress`, `showGenerating`, `showSuccess`, `showError`) writes to the same fixed notification id. `DownloadingTrailIds.download()` only guards re-entrancy for the *same* trail id (`if (state.contains(trail.id)) return;`) — nothing stops a user from starting downloads for two *different* trails concurrently (e.g. tapping download on Trail A, then immediately on Trail B from the library list). Each invocation captures its own `itemCount`/`vectorRegions`/`demRegions`/`lastTrailFraction` closures, but both write to the same notification id 42, so the two downloads' progress/title/body will interleave and clobber each other non-deterministically, and a `showSuccess`/`showError` from one trail can overwrite the other's still-in-progress notification.
**Fix:** Key notifications by trail id (or at least by a monotonically increasing download session id) instead of a single hardcoded constant, or explicitly serialize/queue concurrent downloads if only one is meant to be visible at a time.

### WR-04: Inconsistent null-fallback defaults for the vector checkbox state

**File:** `app/lib/components/trail/missing_coverage_sheet.dart:133`, `:202`, `:302`
**Issue:** The summary calculation and the Download button's selection filter both use `_vectorChecked[row.region.id] ?? false`:

```dart
final vectorSelected =
    row.vectorStatus != RegionStatus.downloading &&
    (_vectorChecked[row.region.id] ?? false);   // line 133, mirrored at 202
```

but the checkbox's displayed `value:` uses the opposite fallback:

```dart
value: _vectorChecked[row.region.id] ?? true,   // line 302
```

Today this is harmless because `_vectorChecked` is eagerly pre-populated with every `widget.missingRegions` id (defaulting to `true`), so the key always exists and the fallback never actually triggers. But it is a latent inconsistency: if the map and the rows ever diverge (e.g. a future change makes `_vectorChecked` lazily populated, or `rows` starts including ids outside `widget.missingRegions`), the checkbox would visually show "checked" while the summary/Download-button logic treats it as "unchecked" — a silent UI/logic mismatch.
**Fix:** Use the same fallback (`?? true`, matching the map's actual default) in all three places, or better, make `_vectorChecked[row.region.id]!` a non-nullable lookup since the map is guaranteed to contain every row's id.

## Info

### IN-01: Hardcoded English strings mixed with localized strings in the same widget

**File:** `app/lib/components/trail/missing_coverage_sheet.dart:144-146, 174-186, 283-286, 325-328`
**Issue:** The sheet uses `AppLocalizations` for the Download button and the Vector/DEM row titles/error text (`l10n.download`, `l10n.regions_vector_tile_title`, `l10n.regions_dem_tile_title`, `l10n.regions_download_failed`), but hardcodes plain English strings for the sheet title ("Missing offline map coverage"), the description paragraph, the summary text ("Downloading trail only", "$selectedCount region(s) selected..."), and the "Downloading…" progress labels. The RESEARCH.md notes this sheet's shape is modeled on `track_save_options_sheet.dart`, which is fully localized (`l10n.save_recording_options`, `l10n.recalculate_heights`, etc.) — this file only partially follows that precedent.
**Fix:** Add ARB entries for the remaining strings and route them through `l10n`, consistent with the sibling sheet this one is modeled after.

### IN-02: Unusual `dart format off/on` wrapping for a single line

**File:** `app/lib/services/download_notification_service.dart:110-112`
**Issue:**
```dart
// dart format off
Future<void> showAggregateProgress(String title, String body, int done, int total) async {
  // dart format on
```
Disabling the formatter for a single long method signature is unnecessary — `dart format` would simply wrap the parameter list across multiple lines. This adds noise and risks accidentally suppressing formatting for more than intended if the surrounding code changes.
**Fix:** Remove the format-off/on pair and let the formatter wrap the signature normally (or wrap it manually).

### IN-03: No explicit test coverage for `downloading`/`error` regions in `missingCoverageRegions`

**File:** `app/test/util/trail_coverage_util_test.dart`
**Issue:** The `missingCoverageRegions` group covers `notDownloaded` (included) and `downloaded`/`updateAvailable` (excluded), but has no explicit case asserting that `RegionStatus.downloading` and `RegionStatus.error` regions are also included in the missing list — both matter directly for GUARD-02/03's sheet, since `missing_coverage_sheet.dart` renders a progress bar for `downloading` rows and an error subtitle for `error` rows, and both rely on `missingCoverageRegions` surfacing them in the first place.
**Fix:** Add two more table-driven cases (`downloading` region included, `error` region included) alongside the existing `notDownloaded`/`updateAvailable` cases.

---

_Reviewed: 2026-07-24T12:02:57Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
