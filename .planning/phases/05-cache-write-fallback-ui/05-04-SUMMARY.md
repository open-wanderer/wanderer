---
phase: 05-cache-write-fallback-ui
plan: "04"
subsystem: flutter-app-service
tags: [dart, flutter, navigation, offline, cache, valhalla]
dependency_graph:
  requires:
    - "05-01: buildNavShape helper in gpx_util.dart"
  provides:
    - "Best-effort navCacheJson write in TrailDownloadService.downloadTrail"
  affects:
    - "app/lib/services/trail_download_service.dart: new navCacheJson assignment before box.put"
tech_stack:
  added: []
  patterns:
    - "Best-effort try/catch with bare swallowing catch (no rethrow) for non-critical network steps"
    - "Inlined costing derivation (mirrors _costingFor in navigation_launch_util.dart) to avoid coupling private functions across files"
    - "Shared buildNavShape helper called from service layer to keep cache/online shape payloads byte-identical"
key_files:
  created: []
  modified:
    - app/lib/services/trail_download_service.dart
decisions:
  - "Inline _costingFor logic rather than extracting to shared util — private function in another file cannot be imported; inlining mirrors the exact pattern without coupling"
  - "Pass cancelToken to the Valhalla POST so a user-cancelled download also cancels the cache fetch"
  - "Guard points.length >= 2 (not > 2) to match the online path guard"
metrics:
  duration_minutes: 8
  completed_date: "2026-06-14T16:14:39Z"
  tasks_completed: 1
  files_modified: 1
---

# Phase 05 Plan 04: Best-effort Valhalla Cache Write Summary

**One-liner:** Added best-effort Valhalla cache write in `downloadTrail` using `buildNavShape` helper; any failure (Valhalla outage, null GPX, parse error) is silently swallowed so tile download and entity persistence are never blocked.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Best-effort Valhalla cache write in downloadTrail | 373faf3f | app/lib/services/trail_download_service.dart |

## What Was Built

Inside `TrailDownloadService.downloadTrail`, after map tiles are downloaded and before `_store.runInTransaction`, a best-effort Valhalla cache block:

1. **Guards:** checks `trail.expand?.gpx != null` and `points.length >= 2`; skips silently if either fails (D-07).
2. **Shape:** calls `buildNavShape(points)` — the same shared helper as the online path (D-08), ensuring byte-identical shape payloads.
3. **Costing:** derives `'bicycle'` or `'pedestrian'` by checking the category name for `bike`/`cycling`/`bicycle` (mirrors `_costingFor` in `navigation_launch_util.dart`; inlined since that function is private).
4. **POST:** `_api.post('/valhalla/navigate', data: {'shape': shape, 'costing': costing}, cancelToken: cancelToken)` — reuses the existing Dio client; respects cancel token.
5. **Persist:** `entity.navCacheJson = jsonEncode(response.toJson())` only when `response.maneuvers.isNotEmpty && response.shape.isNotEmpty` (no broken cache on empty response).
6. **Fallback:** the entire block is wrapped in `try { ... } catch (_) { }` — a bare swallowing catch with no rethrow (D-06). The existing tile-download block still rethrows on failure, which is correct and unchanged.

New imports added: `dart:convert` (jsonEncode), `package:wanderer/models/navigate_response.dart`, `package:wanderer/util/gpx_util.dart`.

## Verification

- `flutter analyze lib/services/trail_download_service.dart` — 4 info-level `avoid_print` items, all pre-existing in `_downloadPhotos`; zero errors introduced by this plan.
- `flutter analyze` (whole app) — 80 issues all pre-existing (deprecated icon names in `icon_util.dart`, unused import in a test file); zero new errors.
- Acceptance criteria confirmed:
  - `buildNavShape(` call present
  - POST to `/valhalla/navigate` with `data: {'shape': shape, 'costing': costing}`
  - Cache block wrapped in swallowing `catch` (no `rethrow`), placed before `box.put(entity)`
  - `entity.navCacheJson = jsonEncode(response.toJson())` guarded by `response.maneuvers.isNotEmpty && response.shape.isNotEmpty`
  - Null-GPX guard (`trail.expand?.gpx == null`) skips silently
  - All three imports present

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — this change is entirely internal to the download service; no new network endpoints, auth paths, or trust boundary crossings introduced. The Valhalla POST reuses the existing `/api/v1` Dio client already used by `navigation_launch_util.dart`.

## Self-Check: PASSED

- FOUND: app/lib/services/trail_download_service.dart
- FOUND: commit 373faf3f (feat(05-04): best-effort Valhalla cache write in downloadTrail)
- FOUND: 05-04-SUMMARY.md
