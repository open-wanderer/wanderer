---
phase: quick-260610-kdc
plan: "01"
subsystem: offline-map-download
tags: [flutter, dart, svelte, security, performance, cancellation]
one_liner: "Fixed four issues in the PMTiles offline-map download flow: added generating enum case, parallelized cell downloads with Future.wait, added input validation guards on all four map cell routes, and threaded CancelToken + onProgress through the entire download stack"
dependency_graph:
  requires: []
  provides: [MapCellStatus.generating, parallel-cell-downloads, cellKey-validation, bbox-validation, download-cancellation, download-progress]
  affects: [app/lib/models/map_cell.dart, app/lib/services/trail_download_service.dart, web/src/routes/api/v1/map/cells]
tech_stack:
  added: []
  patterns: [Future.wait parallel downloads, Dio CancelToken, anchored regex guard, inline 400 guard]
key_files:
  created: []
  modified:
    - app/lib/models/map_cell.dart
    - app/lib/models/map_cell.g.dart
    - app/lib/services/trail_download_service.dart
    - web/src/routes/api/v1/map/cells/+server.ts
    - web/src/routes/api/v1/map/cells/[cellKey]/+server.ts
    - web/src/routes/api/v1/map/cells/[cellKey]/status/+server.ts
    - web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts
decisions:
  - "Thread cancelToken into _downloadPhotos as well so a single cancel stops photo and tile downloads simultaneously"
  - "Delete partial .pmtiles file on DioException cancel before rethrowing to prevent corrupt-tile resume"
  - "Use CELL_KEY_RE module-level constant for cellKey regex (anchored ^...$) in all three [cellKey] routes"
metrics:
  duration: "8m 52s"
  completed_date: "2026-06-10"
  tasks_completed: 3
  tasks_total: 3
  files_modified: 7
---

# Quick Task 260610-kdc: Fix Trail PMTiles Download Summary

## What Was Built

Four focused fixes to the trail PMTiles offline-map download system:

**FIX-1: MapCellStatus.generating enum case**
Added `@JsonValue('generating') generating` to the `MapCellStatus` enum in `app/lib/models/map_cell.dart`. Ran `dart run build_runner build --delete-conflicting-outputs` to regenerate `map_cell.g.dart` so the `"generating"` server response deserializes correctly instead of crashing JSON parsing.

**FIX-2: Parallel cell downloads**
Replaced the sequential `for (final cell in infoList.cells)` loop in `_downloadMapTiles` with the `Future.wait()` pattern that mirrors `_downloadPhotos`. Each cell now downloads independently and concurrently. Per-cell failures return `null` (not an exception) so one failed cell does not abort the batch.

**FIX-3: Input validation on all map cell routes**
Added inline guard-clause validation matching the codebase convention from `geocoding/search/+server.ts`:
- `CELL_KEY_RE` anchored regex (`^-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+$`) in all three `[cellKey]` routes - returns 400 before any `pb.send`/`event.fetch` call
- bbox shape guard in `cells/+server.ts` - exactly 4 comma-separated finite numbers, otherwise 400
- Added `json` import to `download/+server.ts` to enable the 400 response

**FIX-4: CancelToken + onProgress**
Added optional `cancelToken` and `onProgress` parameters to `downloadTrail` and `_downloadMapTiles`. The same `cancelToken` is passed to every `_api.get()` and `_api.download()` call including photo downloads. On cancel: partial `.pmtiles` file is deleted before rethrowing `DioException` to prevent corrupt-tile resume. `onProgress` is called with `(cellsDone, cellsTotal)` after each successful cell download.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 (FIX-1) | `666564ef` | Add generating enum case to MapCellStatus + regenerate codegen |
| Task 2 (FIX-2, FIX-4) | `8ff08147` | Parallel downloads, cancellation, and progress |
| Task 3 (FIX-3) | `22319220` | Validate cellKey and bbox params in map cell routes |

## Verification Results

- `dart run build_runner build --delete-conflicting-outputs` - success, 4 outputs written
- `dart analyze lib/services/trail_download_service.dart lib/models/map_cell.dart` - 8 info-only `avoid_print` warnings (pre-existing in original code), 0 errors, 0 warnings
- `svelte-check --tsconfig ./tsconfig.json` - 0 errors, 0 warnings, 2551 files checked

## Deviations from Plan

### Auto-applied Decisions (from Claude's Discretion)

**1. [Discretion - CancelToken in _downloadPhotos]**
- **Found during:** Task 2
- **Decision:** Also thread `cancelToken` into `_downloadPhotos` so a single cancel stops photo and tile downloads simultaneously
- **Rationale:** RESEARCH.md recommended this; consistent UX; low effort
- **Files modified:** `app/lib/services/trail_download_service.dart`

**2. [Discretion - Partial file cleanup on cancel]**
- **Found during:** Task 2
- **Decision:** Delete partial `.pmtiles` file on `DioException` cancel before rethrowing
- **Rationale:** Prevents corrupt-tile resume - the `File(localPath).exists()` guard would skip re-download of a truncated file; RESEARCH.md Pitfall 3 explicitly recommends this
- **Files modified:** `app/lib/services/trail_download_service.dart`

### Infrastructure Deviation (no code impact)

**[Setup] Worktree rebased onto feature/app**
The executor worktree was created from `main` and did not include the `app/` directory (which only exists on `feature/app`). The worktree branch was rebased onto `feature/app` before starting task execution. This is an orchestrator setup concern, not a plan deviation.

## Known Stubs

None. All changes are functional implementations with no placeholder values.

## Threat Flags

No new security surface introduced beyond what is already in the plan's threat model. The validation guards (FIX-3) mitigate T-quick-01 and T-quick-02 as planned.

## Self-Check: PASSED

- `app/lib/models/map_cell.dart` exists and contains `generating`
- `app/lib/models/map_cell.g.dart` exists and contains `MapCellStatus.generating: 'generating'`
- `app/lib/services/trail_download_service.dart` exists and contains `Future.wait(downloadTasks)` and `cancelToken:`
- `web/src/routes/api/v1/map/cells/+server.ts` exists and contains `status: 400`
- `web/src/routes/api/v1/map/cells/[cellKey]/+server.ts` exists and contains `CELL_KEY_RE`
- `web/src/routes/api/v1/map/cells/[cellKey]/status/+server.ts` exists and contains `CELL_KEY_RE`
- `web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts` exists and contains `CELL_KEY_RE`
- Commits `666564ef`, `8ff08147`, `22319220` verified in git log
