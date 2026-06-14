---
phase: 05-cache-write-fallback-ui
fixed_at: 2026-06-14T00:00:00Z
review_path: .planning/phases/05-cache-write-fallback-ui/05-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 05: Code Review Fix Report

**Fixed at:** 2026-06-14T00:00:00Z
**Source review:** .planning/phases/05-cache-write-fallback-ui/05-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6
- Fixed: 6
- Skipped: 0

## Fixed Issues

### CR-01: shapeAsLatLng index clamped against wrong list length

**Files modified:** `app/lib/provider/navigation_provider.dart`
**Commit:** 8192b659
**Applied fix:** In `onPosition()`, captured `shapeAsLatLng` into a local `validShape`, added an early-return guard when `validShape.isEmpty`, and clamped `rawIndex` against `validShape.length - 1` instead of `shape.length - 1`. The subscript now uses `validShape[clampedIndex]` directly, eliminating the RangeError when the server returns malformed shape entries that are filtered out by the extension getter.

---

### CR-02: Valhalla catch block swallows DioException cancel — partial entity write

**Files modified:** `app/lib/services/trail_download_service.dart`
**Commit:** 5c1e9c7f
**Applied fix:** Changed the Valhalla best-effort catch from `catch (_)` to `catch (e)` and added `if (e is DioException && CancelToken.isCancel(e)) rethrow;` before the swallow comment. This matches the pattern used in `_downloadPhotos` at line 254 and prevents a cancelled download from falling through to `box.put(entity)` with a partial entity.

---

### CR-03: Force-unwrap on nullable getFileUrl() result crashes on empty photo filename

**Files modified:** `app/lib/services/trail_download_service.dart`
**Commit:** 818ff138
**Applied fix:** Replaced both `.map((p) => trail.getFileUrl(baseUrl, p)!).toList()` and `.map((p) => waypoint.getFileUrl(baseUrl, p)!).toList()` with `.map((p) => trail.getFileUrl(baseUrl, p)).whereType<String>().toList()` and the equivalent waypoint form. `whereType<String>()` silently skips null results (empty or null filenames) instead of crashing.

---

### WR-01: Costing logic duplicated between two files

**Files modified:** `app/lib/util/gpx_util.dart`, `app/lib/util/navigation_launch_util.dart`, `app/lib/services/trail_download_service.dart`
**Commit:** 7714c851
**Applied fix:** Added a new top-level `costingForCategory(String? category)` function to `gpx_util.dart` with full doc comment. Removed the private `_costingFor()` function from `navigation_launch_util.dart` and updated its call site to use `costingForCategory`. Replaced the inlined 8-line costing derivation in `trail_download_service.dart` with a single call to `costingForCategory`. Both files already imported `gpx_util.dart`.

---

### WR-02: _downloadMapTiles progress counter mutated from concurrent async tasks

**Files modified:** `app/lib/services/trail_download_service.dart`
**Commit:** 94f52458
**Applied fix:** Removed the `var completed` counter and the two `onProgress?.call(++completed, total)` calls inside the concurrent closures. Added a single progress report after `Future.wait` completes, computing `done` as `results.whereType<String>().length`. This eliminates non-monotonic progress reports from interleaved closure resumptions while preserving the progress callback contract. Trade-off: progress is now reported once at completion rather than incrementally; no existing callers were observed to depend on intermediate values.

---

### WR-03: _buildStatsSheet uses dynamic for trailAsync parameter type

**Files modified:** `app/lib/routes/navigation_screen.dart`
**Commit:** 27778d3b
**Applied fix:** Added `import 'package:wanderer/models/trail.dart';` to the import section. Changed both `_buildStatsSheet` (line 397) and `_buildElevationPage` (line 576) parameter types from `AsyncValue<dynamic>` to `AsyncValue<Trail>`. The type system will now catch any future change to what `trailProvider` returns at compile time.

---

_Fixed: 2026-06-14T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
