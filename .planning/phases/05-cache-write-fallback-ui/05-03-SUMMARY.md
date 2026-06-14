---
phase: 05-cache-write-fallback-ui
plan: "03"
subsystem: flutter-app-util
tags: [dart, flutter, navigation, offline, cache, objectbox, dio]
dependency_graph:
  requires:
    - "05-01: buildNavShape helper in gpx_util.dart"
    - "05-02: NavigationScreen.isOffline param and (NavigateResponse, bool) record route"
    - "04-02: navCacheJson field on TrailEntity and objectbox-model.json"
    - "04-01: NavigateResponse.toJson() roundtrip serialization fix"
  provides:
    - "launchNavigation: DioException → ObjectBox cache read → push (cached, true)"
    - "launchNavigation: success → push (response, false) + unawaited re-cache"
    - "_readCachedNav: ObjectBox query by trail id, decode navCacheJson, null on miss/error"
    - "_recacheNav: update navCacheJson via write transaction, swallow all errors"
  affects:
    - "05-04: trail_download_service.dart will also call buildNavShape for cache-write on download"
tech_stack:
  added: []
  patterns:
    - "on DioException catch (_) before generic catch (_) for typed network-error handling"
    - "unawaited() for fire-and-forget background async work after synchronous push"
    - "objectbox.g.dart import provides both Store/TxMode and TrailEntity_ codegen query helpers"
    - "try/catch in _readCachedNav for null-safe JSON decode with cache-miss fallback"
    - "try/catch in _recacheNav that swallows all errors for best-effort write"
key_files:
  created: []
  modified:
    - app/lib/util/navigation_launch_util.dart
decisions:
  - "Use package:wanderer/objectbox.g.dart instead of package:objectbox/objectbox.dart — g.dart re-exports everything from objectbox.dart and also provides the codegen TrailEntity_ query class; importing both causes unnecessary_import warning"
  - "Both tasks implemented in one commit since they modify the same file and the helpers are immediately wired in the same edit pass — separate commits would require splitting identical file states"
metrics:
  duration_minutes: 10
  completed_date: "2026-06-14T15:37:53Z"
  tasks_completed: 2
  files_modified: 1
---

# Phase 05 Plan 03: Cache Fallback + Silent Re-cache in launchNavigation Summary

**One-liner:** `launchNavigation` now catches `DioException` to serve maneuvers from ObjectBox with `isOffline:true`, pushes `(response, false)` on success and fires an unawaited background re-cache, with a generic catch preserving the existing error toast.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Use buildNavShape + add private cache read/write helpers | ab810211 | app/lib/util/navigation_launch_util.dart |
| 2 | DioException fallback + isOffline propagation + unawaited re-cache | ab810211 | app/lib/util/navigation_launch_util.dart |

Note: Both tasks landed in a single commit (ab810211) because they modify the same file and the helpers added in Task 1 are immediately consumed by the control flow added in Task 2.

## What Was Built

### Private cache helpers

`_readCachedNav(Store store, String trailId) -> NavigateResponse?`
- Queries `store.box<TrailEntity>()` using `TrailEntity_.id.equals(trailId)`
- Calls `findFirst()` then closes the query
- Returns null if entity is missing or `navCacheJson` is null
- Wraps `NavigateResponse.fromJson(jsonDecode(json))` in try/catch, returning null on decode error (D-11: undecodable cache = cache miss)

`_recacheNav(Store store, String trailId, NavigateResponse response) -> Future<void>`
- Queries entity by id, exits early if not found (trail not downloaded = no-op)
- Sets `entity.navCacheJson = jsonEncode(response.toJson())`
- Writes via `store.runInTransaction(TxMode.write, () { box.put(entity); })`
- Entire body wrapped in try/catch that swallows errors (D-13: re-cache must never surface to user)

### launchNavigation changes

1. **Shape building:** Replaced the 22-line inline downsampling block (lines 112-138 in original) with `final shape = buildNavShape(points);` — delegates to the shared helper from plan 05-01.

2. **Success path:** `context.push('/trail/${trail.id}/navigate', extra: (response, false));` then `unawaited(_recacheNav(store, trail.id, response));` — online navigation pushes `isOffline:false` and silently refreshes the cache in the background (D-03, D-12, D-13).

3. **DioException catch:** `on DioException catch (_)` block reads cache via `_readCachedNav`; if `cached != null && cached.maneuvers.isNotEmpty && cached.shape.isNotEmpty`, pushes `extra: (cached, true)` with `isOffline:true` (D-09, OFFLINE-02); otherwise shows the existing `couldnt_start_navigation` toast (D-11).

4. **Generic catch:** Retained as `catch (_)` for non-Dio errors (parse failures, etc.) showing the same error toast — preserving today's UX (D-07).

### New imports added
- `dart:async` — for `unawaited()`
- `dart:convert` — for `jsonEncode` / `jsonDecode`
- `package:dio/dio.dart` — for `DioException`
- `package:wanderer/entities/trail_entity.dart` — for `TrailEntity`
- `package:wanderer/objectbox.g.dart` — for `Store`, `TxMode`, `TrailEntity_`
- `package:wanderer/provider/objectbox_store_provider.dart` — for `objectBoxProvider`

## Verification

- `flutter analyze lib/util/navigation_launch_util.dart` — No issues found
- `flutter test test/models/navigate_response_test.dart` — 9/9 tests pass (roundtrip serialization the cache depends on)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] package:objectbox/objectbox.dart causes unnecessary_import warning when used alongside objectbox.g.dart**

- **Found during:** Task 1 (flutter analyze reported `unnecessary_import` info)
- **Issue:** The plan spec lists `import 'package:objectbox/objectbox.dart'` as an import to add. However, `package:wanderer/objectbox.g.dart` (which is required for `TrailEntity_`) already re-exports all symbols from `objectbox.dart` including `Store` and `TxMode`. Having both imports triggers an `unnecessary_import` info warning.
- **Fix:** Omitted `package:objectbox/objectbox.dart`; used only `package:wanderer/objectbox.g.dart` which provides all needed symbols (`Store`, `TxMode`, `TrailEntity_`) from a single import.
- **Files modified:** app/lib/util/navigation_launch_util.dart
- **Commit:** ab810211

## Known Stubs

None — all data paths are fully wired. `_readCachedNav` returns a real `NavigateResponse` decoded from ObjectBox or null; `_recacheNav` writes real JSON to `navCacheJson`; the router (from 05-02) correctly unpacks the `(NavigateResponse, bool)` record.

## Threat Flags

None — no new network endpoints, auth paths, or schema changes introduced. `_recacheNav` only writes to local ObjectBox (device-local storage); `_readCachedNav` only reads from local ObjectBox. Both are scoped to a single trail entity by id.

## Self-Check: PASSED

- FOUND: app/lib/util/navigation_launch_util.dart
- FOUND: .planning/phases/05-cache-write-fallback-ui/05-03-SUMMARY.md
- FOUND: commit ab810211 (feat(05-03): wire buildNavShape, cache fallback, and silent re-cache in launchNavigation)
- flutter analyze: No issues found
- flutter test navigate_response_test: 9/9 passed
