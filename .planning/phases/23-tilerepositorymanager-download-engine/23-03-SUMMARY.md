---
phase: 23-tilerepositorymanager-download-engine
plan: 03
subsystem: infra
tags: [flutter, dio, disk-space, disk_space_2, supply-chain-security, tdd]

# Dependency graph
requires:
  - phase: 22-region-package-data-model
    provides: RegionEntity/DownloadedTilePackageEntity ObjectBox schema (this util is a downstream primitive Plan 04 will call before writing region archive files)
provides:
  - "disk_space_2 ^1.0.12 approved (post-legitimacy-gate) and added to app/pubspec.yaml"
  - "app/lib/util/disk_space_util.dart: freeDiskSpaceBytes([path]) fail-closed plugin wrapper + pure hasEnoughSpace(...) safety-margin decision"
  - "app/test/util/disk_space_util_test.dart: unit coverage of the margin math and the null/fail-closed path"
affects: ["23-04 (TileRepositoryManager download engine — will call hasEnoughSpace before each file write)"]

# Tech tracking
tech-stack:
  added: ["disk_space_2 ^1.0.12 (Flutter plugin, MethodChannel-backed StatFs/NSFileManager wrapper)"]
  patterns: ["Plugin-wrapper + pure-decision split: freeDiskSpaceBytes (impure, plugin-backed, never throws) feeds hasEnoughSpace (pure, synchronous, fully unit-tested) — mirrors map_cache_path.dart's pure/tested-util shape"]

key-files:
  created:
    - app/lib/util/disk_space_util.dart
    - app/test/util/disk_space_util_test.dart
  modified:
    - app/pubspec.yaml
    - app/pubspec.lock

key-decisions:
  - "Task 1 (blocking checkpoint:decision, resolved externally before this execution): disk_space_2 approved over storage_space and a hand-rolled platform channel"
  - "hasEnoughSpace default safetyMultiplier is 1.75 (tunable heuristic per RESEARCH Assumption A3, not a fixed guarantee)"
  - "freeDiskSpaceBytes prefers the path-specific query (DiskSpace.getFreeDiskSpaceForPath) when a path is supplied, falling back to the device-wide query only when no path is given — never silently falls back on a path-specific failure, since that failure is swallowed to null and handled by hasEnoughSpace's fail-closed branch instead"

patterns-established:
  - "Plugin-wrapper + pure-decision split: any future plugin-backed decision (not just disk space) should follow this shape — an async wrapper that never throws, feeding a synchronous pure function that is the actual unit-tested decision logic"

requirements-completed: [TILE-03]

# Metrics
duration: 12min
completed: 2026-07-22
---

# Phase 23 Plan 03: Disk-Space Package Legitimacy Gate + Fail-Closed Util Summary

**Added `disk_space_2` (post-legitimacy-review) and wrapped it in `disk_space_util.dart`, whose pure `hasEnoughSpace` margin decision is unit-tested independently of the plugin.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-22T09:34:00Z
- **Completed:** 2026-07-22T09:46:00Z
- **Tasks:** 2 (Task 1: checkpoint:decision, resolved externally; Task 2: auto, tdd)
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- Resolved the blocking package-legitimacy decision for TILE-03's disk-space check (Task 1) — evidence gathered and recorded below, per the pre-resolved-checkpoint instruction
- Added `disk_space_2 ^1.0.12` to `app/pubspec.yaml` only after that legitimacy review, never before
- Built `disk_space_util.dart`: a fail-closed `freeDiskSpaceBytes([path])` wrapper (never throws) plus a pure, tested `hasEnoughSpace(...)` safety-margin decision that Plan 04's `TileRepositoryManager` will call before every region file write
- Confirmed the installed package's real public API by direct source inspection (not the research doc's WebSearch-derived description) before writing any call site

## Task Commits

Each task was committed atomically:

1. **Task 1: Resolve the disk-space package legitimacy decision** — no code changes (decision-only gate); resolution and evidence documented in this SUMMARY (see "Legitimacy Decision" below). Pre-resolved per the orchestrator's instruction; user had already selected `disk_space_2` outside this execution.
2. **Task 2: Add the approved package and wrap it in a fail-closed util with a pure margin core** - `b092e88c` (feat)

**Plan metadata:** committed alongside this SUMMARY / STATE / ROADMAP update (see final commit below).

## Legitimacy Decision (Task 1 — resolved)

**Decision:** `disk_space_2` (research recommendation), as selected by the user prior to this execution.

**Evidence gathered during this execution** (live queries against pub.dev's registry API and the package's GitHub repository, since the research pass could only WebSearch-derive this data):

- **pub.dev registry** (`https://pub.dev/api/packages/disk_space_2`): latest version `1.0.12`, published `2026-01-05T15:14:34Z`; first version (`0.0.1`) published `2024-06-16`; `repository: https://github.com/tom-ludwig/disk_space_2`; `homepage: https://activecoding.de.cool/`.
- **pub.dev score** (`https://pub.dev/api/packages/disk_space_2/score`): 150/160 pub points, 16 likes, 19,017 downloads in the trailing 30 days; tags confirm `sdk:flutter`, `platform:android`, `platform:ios`, `is:dart3-compatible`, `license:mit`/`osi-approved`.
- **Repository provenance** (`https://api.github.com/repos/tom-ludwig/disk_space_2`): confirms the fork lineage — early versions (`0.0.1`–`1.0.1`) point their `repository` field at `github.com/activcoding/Disk-Space` (the original, now-unmaintained package this is a fork of); later versions repoint to `tom-ludwig/disk_space_2`, matching the research's characterization of it as a community fork.
- **Native Android source** (`android/src/main/kotlin/.../DiskSpace_2Plugin.kt`, read verbatim from GitHub): implements exactly three `MethodChannel` calls — `getFreeDiskSpace`, `getTotalDiskSpace`, `getFreeDiskSpaceForPath` — each backed by `android.os.StatFs` against `Environment.getExternalStorageDirectory()` or a caller-supplied path, run on a single-thread `ExecutorService` and posted back via `Handler(Looper.getMainLooper())`. No network calls, no file writes, no reflection, no shell-out — the entire native surface is a `StatFs` read.
- **Native iOS source** (`ios/.../DiskSpace_2Plugin.swift`, read verbatim from GitHub): same three methods, backed by `FileManager.default.attributesOfFileSystem(forPath:)` (total) and `URL(...).resourceValues(forKeys: [.volumeAvailableCapacityForImportantUsageKey])` (free, both device-wide and path-specific). Same conclusion: read-only filesystem-capacity queries, nothing else.
- **Installed Dart API** (`~/.pub-cache/hosted/pub.dev/disk_space_2-1.0.12/lib/disk_space_2.dart`, read directly rather than trusting RESEARCH.md's WebSearch-derived method names per Open Question 1): confirms `DiskSpace.getFreeDiskSpace` and `DiskSpace.getTotalDiskSpace` are static getters returning `Future<double?>` in mebibytes, and `DiskSpace.getFreeDiskSpaceForPath(String path)` is a static method (also mebibytes) that throws a plain `Exception` if the path doesn't exist. This confirms RESEARCH's Assumption A2 (a path-specific query does exist in the installed `1.0.12` API) and resolves Open Question 1.

**Conclusion:** the native code on both platforms does exactly what its description claims — read free/total filesystem capacity — with no additional capability that would make it a meaningful supply-chain risk beyond "small, low-scrutiny maintainer." Task 1's gate is resolved in favor of adding `disk_space_2`, matching the user's prior selection.

## Files Created/Modified

- `app/pubspec.yaml` - added `disk_space_2: ^1.0.12` (inserted alphabetically after `dio_cookie_manager`)
- `app/pubspec.lock` - resolved by `flutter pub get` after the add
- `app/lib/util/disk_space_util.dart` - `freeDiskSpaceBytes([forPath])` (fail-closed plugin wrapper, converts mebibytes to bytes, never throws) + `hasEnoughSpace({freeBytes, declaredSizeBytes, safetyMultiplier = 1.75})` (pure, fail-closed on `null`)
- `app/test/util/disk_space_util_test.dart` - 5 tests covering the plan's 4 required behaviors (300MB/100MB true, 150MB/100MB false, null fail-closed, default multiplier) plus one additional custom-multiplier case

## Decisions Made

- Confirmed `disk_space_2`'s real installed API (via direct pub-cache inspection) before writing any call site, rather than trusting RESEARCH.md's WebSearch-derived method names — resolves the plan's Open Question 1 / Assumption A2: the path-specific `getFreeDiskSpaceForPath` does exist in `1.0.12`.
- `freeDiskSpaceBytes` prefers the path-specific query when a path argument is given (more precise about which volume/mount is measured, matching RESEARCH's stated preference), falling back to the device-wide query only when no path was supplied at all — a path-specific query failure is not silently retried against the device-wide query; it is swallowed to `null` and handled uniformly by `hasEnoughSpace`'s fail-closed branch.
- Kept `disk_space_util.dart`'s two functions in the same file (rather than splitting wrapper vs. pure decision across files) since the plan's own artifact spec names a single file and the pure function is small enough to unit-test in isolation regardless of file boundary.
- Inserted `disk_space_2` alphabetically in `pubspec.yaml` (after `dio_cookie_manager`, before `duration`) rather than leaving it at the end where `flutter pub add` appended it, matching this codebase's existing near-alphabetical dependency ordering convention.

## Deviations from Plan

None - plan executed exactly as written. Task 1's checkpoint:decision was pre-resolved by the user outside this execution (per explicit orchestrator instruction); this run performed the legitimacy evidence-gathering the task's `<action>` describes and proceeded directly to Task 2 without pausing.

## Issues Encountered

None. `flutter pub add disk_space_2`, `flutter pub get`, `flutter test test/util/disk_space_util_test.dart`, and `flutter analyze lib/util/disk_space_util.dart` all succeeded on the first attempt.

## User Setup Required

None - no external service configuration required. The package is a local Flutter plugin; no API keys or accounts needed.

## Next Phase Readiness

- `hasEnoughSpace(...)` is ready for Plan 04's `TileRepositoryManager` to call before every vector/DEM region file write (TILE-03).
- `freeDiskSpaceBytes` is untested on-device (by design — this plan's `<action>` explicitly scoped device verification to Plan 06's on-device human-check); Plan 04/06 should confirm the plugin actually resolves a sensible free-space number on a real Android and iOS device before shipping.
- No blockers for Plan 04.

---
*Phase: 23-tilerepositorymanager-download-engine*
*Completed: 2026-07-22*
