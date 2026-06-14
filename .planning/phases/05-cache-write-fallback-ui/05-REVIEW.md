---
phase: 05-cache-write-fallback-ui
reviewed: 2026-06-14T00:00:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - app/lib/provider/router_provider.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/services/trail_download_service.dart
  - app/lib/util/gpx_util.dart
  - app/lib/util/navigation_launch_util.dart
  - app/test/util/gpx_util_test.dart
findings:
  critical: 3
  warning: 3
  info: 2
  total: 8
status: issues_found
---

# Phase 05: Code Review Report

**Reviewed:** 2026-06-14T00:00:00Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Six files covering the cache-write path, offline navigation fallback, and the
navigation UI were reviewed. The core `buildNavShape` algorithm is correct —
size bounds hold for all inputs — and the mounted-guard discipline in
`navigation_launch_util.dart` is solid. However, three bugs that can cause
runtime crashes or silent data corruption were found, along with three
robustness warnings and two minor quality items.

---

## Critical Issues

### CR-01: `shapeAsLatLng[clampedIndex]` index is clamped against the wrong list length — RangeError crash

**File:** `app/lib/provider/navigation_provider.dart:97-100`

**Issue:** `clampedIndex` is bounded by `state.response.shape.length - 1`, but
the index is then used to subscript `state.response.shapeAsLatLng`, which is a
*different, shorter* list. The `shapeAsLatLng` extension getter (
`navigate_response.dart:42-45`) filters out shape entries with fewer than 2
elements via `.where((p) => p.length >= 2)`. If the server or cache ever
returns even a single malformed shape entry (e.g., `[]` or `[lat]`),
`shapeAsLatLng.length < shape.length`, and the clamped index is out-of-bounds
for `shapeAsLatLng`, throwing an uncaught `RangeError` at runtime and crashing
the navigation screen.

**Fix:**

```dart
// navigation_provider.dart — onPosition()
final rawIndex = state.response.maneuvers[next].beginShapeIndex;
// Clamp against shapeAsLatLng, which may be shorter than shape when
// the server returns malformed entries.
final validShape = state.response.shapeAsLatLng;
if (validShape.isEmpty) return;
final clampedIndex = rawIndex.clamp(0, validShape.length - 1).toInt();
final targetLatLng = validShape[clampedIndex];
```

---

### CR-02: Valhalla `catch (_)` swallows `DioException` from `cancelToken` cancellation — entity written after cancelled download

**File:** `app/lib/services/trail_download_service.dart:80-119`

**Issue:** The best-effort Valhalla block passes `cancelToken` to `_api.post`
(line 104) but catches all exceptions with `catch (_)` (line 117). When the
user cancels the download, `Dio` throws a `DioException` with
`DioExceptionType.cancel`. This exception is silently swallowed, and execution
falls through to `box.put(entity)` on line 121, persisting a partially-built
entity that may be missing photos or tile paths. Every other download operation
in the same file explicitly re-throws cancel exceptions (e.g., line 254
`if (CancelToken.isCancel(e)) rethrow`).

**Fix:**

```dart
} catch (e) {
  // Re-throw if the download was cancelled — must not write a partial entity.
  if (e is DioException && CancelToken.isCancel(e)) rethrow;
  // Best-effort: Valhalla outage must not block download (D-06).
}
```

---

### CR-03: Force-unwrap `!` on nullable `getFileUrl()` result will crash on empty photo filename

**File:** `app/lib/services/trail_download_service.dart:39,50`

**Issue:** Both calls force-unwrap the return value of `getFileUrl()` with `!`:

```dart
trail.photos.map((p) => trail.getFileUrl(baseUrl, p)!).toList()    // line 39
waypoint.photos.map((p) => waypoint.getFileUrl(baseUrl, p)!).toList() // line 50
```

`getFileUrl` returns `null` when `filename` is `null` **or empty** (
`record.dart:13-16`). The `photos` field is a `List<String>`, so individual
entries can legally be `""` (e.g., from a server returning a partially-formed
list). A single empty string in either list throws a `Null check operator used
on a null value` error, aborting the entire download without cleanup of the
already-created `trailDir`.

**Fix:**

```dart
// Filter out un-resolvable filenames rather than crashing.
trail.photos
    .map((p) => trail.getFileUrl(baseUrl, p))
    .whereType<String>()
    .toList()
// and similarly for waypoints
```

---

## Warnings

### WR-01: Costing logic is copy-pasted between two files — divergence risk

**File:** `app/lib/services/trail_download_service.dart:89-99`

**Issue:** The three-keyword costing derivation (bike/cycling/bicycle →
`'bicycle'`, else `'pedestrian'`) is duplicated verbatim from
`_costingFor()` in `navigation_launch_util.dart:74-82`. The comment on line
89-91 even acknowledges the duplication but attributes it to `_costingFor`
being private. If the costing logic ever gains a new keyword or a third mode
(e.g., `'hiking'`), the two copies will silently diverge, breaking the
cache-hit guarantee that the cached and live requests are byte-identical (D-08).

**Fix:** Move `_costingFor` to `gpx_util.dart` (which is already the shared
module for this feature) as a top-level function and import it from both call
sites.

```dart
// gpx_util.dart — add top-level function
String costingForCategory(String? category) {
  final lower = (category ?? '').toLowerCase();
  if (lower.contains('bike') ||
      lower.contains('cycling') ||
      lower.contains('bicycle')) {
    return 'bicycle';
  }
  return 'pedestrian';
}
```

---

### WR-02: `_downloadMapTiles` progress counter is mutated from concurrent `async` tasks — data race

**File:** `app/lib/services/trail_download_service.dart:145-185`

**Issue:** `completed` is a local `var int` incremented with `++completed`
inside `.map((cell) async { ... })`. All tile download closures run
concurrently via `Future.wait` (line 184). `++completed` is a non-atomic
read-modify-write on the `int`, and because Dart's event loop is single-
threaded within an isolate the incidentally works today, but the value passed
to `onProgress` can still be incorrect: two closures may interleave at await
points, each reading the same stale `completed` value and producing duplicate
progress reports or skipping counts (e.g., reporting 3/10, 3/10, 5/10 instead
of 3/10, 4/10, 5/10).

**Fix:** Restructure so `completed` is incremented atomically inside a
sequential step, or use an integer tracked outside the concurrent map with
proper ordering:

```dart
// Simple fix: collect results first, then report progress.
final results = await Future.wait(downloadTasks);
// Or: use a separate sequential pass after Future.wait.
```

Alternatively accept the current Dart single-isolate guarantee and document
that the counter can still produce non-monotonic reports when closures resume
in different order than they suspended.

---

### WR-03: `_buildStatsSheet` uses `dynamic` for `trailAsync` parameter type

**File:** `app/lib/routes/navigation_screen.dart:396`

**Issue:** The `_buildStatsSheet` and `_buildElevationPage` methods accept
`AsyncValue<dynamic>` instead of the concrete type `AsyncValue<Trail>`. This
silences type-checker warnings and allows `trail.expand?.gpx` (line 579) to
proceed without type verification, meaning a refactor that changes what
`trailProvider` returns would not produce a compile-time error in these
methods.

```dart
// Line 396 — current
Widget _buildStatsSheet(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
    AsyncValue<dynamic> trailAsync,  // <-- dynamic
```

**Fix:**

```dart
// Import Trail if not already imported
Widget _buildStatsSheet(
    BuildContext context,
    AppLocalizations localizations,
    NavigationStats stats,
    AsyncValue<Trail> trailAsync,
```

---

## Info

### IN-01: `print()` used for error logging in production code

**File:** `app/lib/services/trail_download_service.dart:255,258`

**Issue:** Photo download failures are reported via `print()` which is stripped
in release mode on some platforms and is not structured. The project convention
(CLAUDE.md) specifies `debugPrint()` for development output. The navigation
screen already uses `debugPrint` (line 84 of `navigation_screen.dart`).

**Fix:** Replace both `print(...)` calls with `debugPrint(...)`:

```dart
debugPrint('Failed to download photo $url: $e');
```

---

### IN-02: `GpxStats.totalElevationloss` field name violates Dart `camelCase` convention

**File:** `app/lib/util/gpx_util.dart:51,57`

**Issue:** The field is named `totalElevationloss` (lowercase `l` in `loss`)
while the local variable used to compute it is `totalElevationLoss` (line 85).
This is inconsistent and violates Dart's camelCase convention. The CLAUDE.md
spec requires camelCase for all variable names.

**Fix:** Rename to `totalElevationLoss`:

```dart
// gpx_util.dart
final double totalElevationLoss;
// ...
required this.totalElevationLoss,
// ...
totalElevationLoss: totalElevationLoss,
```

---

_Reviewed: 2026-06-14T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
