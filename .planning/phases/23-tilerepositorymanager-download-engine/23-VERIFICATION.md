---
phase: 23-tilerepositorymanager-download-engine
verified: 2026-07-22T12:30:00Z
status: human_needed
score: 5/5 must-haves verified (code/static level); 5 on-device behaviors outstanding
overrides_applied: 0
human_verification:
  - test: "RESUME (TILE-02): On a physical device against a backend with a `ready` region, start the region's vector download; partway through, toggle airplane mode so the transfer fails, then restore connectivity and resume via the harness (`flutter run -t test/services/tile_repository_manager_harness.dart`)."
    expected: "The logged received/total byte count continues from the partial offset (not restarting at 0), the resumed HTTP request hits the server as 206 Partial Content (not a full 200 re-download), the final file passes PmTilesArchive validation, and the package promotes to `downloaded`."
    why_human: "Requires a live network interruption, a real partial `.part` file, and a real HTTP round-trip through the now-Range-forwarding SvelteKit proxy — cannot be exercised by static analysis or the pure `resumePlanFor` unit test."
  - test: "DISK REFUSAL (TILE-03): On a device/emulator with little free space (or an inflated declared size), start a download that exceeds free space minus the 1.75x margin using the harness."
    expected: "The download is refused, the package is marked `error`, and no `.part` file (or only a zero-byte one) is written to disk."
    why_human: "Requires real platform-native free-space semantics (`disk_space_2`'s StatFs/NSFileManager calls) on an actual device/emulator filesystem — the pure `hasEnoughSpace` unit test only proves the margin math, not that `freeDiskSpaceBytes` resolves a real, sane number on-device."
  - test: "BACKGROUNDING (TILE-04): Start a download via the harness and immediately background the app (home button / app switcher); return to foreground."
    expected: "The region shows `paused` (not a silently dead transfer) while backgrounded, and resuming from the foreground continues cleanly from the preserved `.part` file."
    why_human: "Requires a real OS lifecycle signal (iOS suspension / Android Doze) delivered to `AppLifecycleListener.onPause` — cannot be triggered or observed by static analysis or unit tests."
  - test: "DEM INDEPENDENCE (DEM-01/02): Using the harness, toggle the DEM download on/off independently of the vector package for the same region; delete the DEM package and confirm the vector package remains `downloaded`, and vice versa."
    expected: "Vector and DEM package status/lifecycle never affect one another; deleting one leaves the other's status and file untouched."
    why_human: "Requires a live Store + two real concurrent/sequential downloads against a real backend to observe cross-package independence end-to-end; the unit tests only prove the code paths are structurally separate, not that they behave independently under real I/O."
  - test: "QUERY (TILE-05): With a region fully downloaded via the harness, tap 'Query inside bbox' and 'Query outside bbox' for that region."
    expected: "The inside-bbox query prints the region's vector (and DEM, if downloaded) local file path; the outside-bbox query prints an empty result."
    why_human: "Requires a real downloaded region with a real `localFilePath` on disk to observe the query resolving actual paths — the `bboxOverlaps` unit tests only cover the pure geometry function, not the full `localTilePathsForBounds` path against a live Store with real downloaded packages."
---

# Phase 23: TileRepositoryManager — Download Engine Verification Report

**Phase Goal:** Resumable, disk-safe, backgrounding-aware region downloads plus a bbox-to-local-paths query, fully decoupled from Trail.
**Verified:** 2026-07-22
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Every truth below is verified at the code/static/unit-test level (artifacts exist, are substantive, are correctly wired, and their pure logic is unit-tested). None of them has been confirmed against real on-device I/O, real network interruption, real OS backgrounding signals, or real platform disk-space queries — that confirmation is the explicit subject of this project's `human_verify_mode: end-of-phase` workflow and is listed under "Human Verification Required" below, not silently assumed passing.

| # | Truth (Roadmap Success Criterion) | Status | Evidence |
|---|-------|--------|----------|
| 1 | `TileRepositoryManager` starts, pauses, resumes, and deletes a region's vector download; DEM download fetches independently of vector | ✓ VERIFIED (code) | `app/lib/services/tile_repository_manager.dart`: `startVectorDownload`/`startDemDownload`/`pauseRegion`/`resumeRegion`/`deleteRegion` all present and structurally independent — `startDemDownload` never touches `region.lastDownloadedVersion` or the vector package; `deleteRegion` clears vector/DEM `ToOne` targets and files independently. `flutter analyze` clean; not yet confirmed on a real device (see human-check "DEM INDEPENDENCE"). |
| 2 | Interrupting and resuming an in-progress download continues from a partial file via HTTP Range + `FileAccessMode.append`, not from byte 0, within the same app session | ✓ VERIFIED (code) | SvelteKit proxy (`web/src/routes/api/v1/regions/[id]/download{,-dem}/+server.ts`) now forwards the incoming `Range` header and the upstream `status`/`Content-Range`/`Accept-Ranges` — proven by 4 passing vitest assertions (`npx vitest run src/routes/api/v1/regions`: 2 files, 4 tests, all pass). `resumePlanFor` (`tile_repository_manager.dart:27`) returns `(offset:0, write, sendRange:false, deleteOnError:true)` for 0 bytes and `(offset:N, append, sendRange:true, deleteOnError:false)` for N>0 bytes, unit-tested (2 passing cases). `_downloadResumable` wires this into `_api.download(...)` with the `Range: bytes=<offset>-` header. Not yet confirmed against a real interrupted transfer (see human-check "RESUME"). |
| 3 | Before each file write, available disk space is checked with a safety margin; a download that would exceed it is refused with a specific state rather than partially writing | ✓ VERIFIED (code) | `disk_space_util.dart`: `hasEnoughSpace` is pure, fail-closed on `null`, default `1.75x` margin — 5 passing unit tests including the null/fail-closed case. `startVectorDownload`/`startDemDownload` call `hasEnoughSpace` before any directory/file creation and mark the package `PackageStatus.error` + `return` (no bytes written) when it's false. The disk-space package (`disk_space_2`) legitimacy gate was resolved by the user and documented with live pub.dev/GitHub evidence in 23-03-SUMMARY.md. Real platform free-space resolution not yet confirmed on-device (see human-check "DISK REFUSAL"). |
| 4 | The app backgrounding mid-download leaves the download in a deliberate paused state that resumes cleanly on foreground, not a silently dead transfer | ✓ VERIFIED (code) | `AppLifecycleListener(onPause: _pauseAllActiveDownloads)` wired in the constructor; `_pauseAllActiveDownloads` cancels every active `CancelToken` with reason `'app-backgrounded'`; the `CancelToken.isCancel(e)` branch in both `startVectorDownload`/`startDemDownload` preserves the `.part` file and sets `PackageStatus.paused` (never deletes, never marks `error`). `dispose()` cleans up the listener. Real OS lifecycle delivery not yet confirmed on-device (see human-check "BACKGROUNDING"). |
| 5 | `localTilePathsForBounds(bbox)` returns the local vector/DEM file paths for every downloaded region intersecting a given bounding box | ✓ VERIFIED (code) | `bboxOverlaps` is the negated-disjoint axis-aligned test with strict `</>` (edge-touching counts as overlap) — 3 passing unit tests (overlap/disjoint/edge-touching). `localTilePathsForBounds` iterates all regions, skips non-overlapping ones, collects non-null `vectorPackage.target?.localFilePath`/`demPackage.target?.localFilePath`. Not yet confirmed against a real downloaded region + live Store (see human-check "QUERY"). |

**Score:** 5/5 truths verified at the code/static level. 0 failed. 5 outstanding on-device confirmations (see Human Verification Required).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/routes/api/v1/regions/[id]/download/+server.ts` | Range-forwarding vector proxy | ✓ VERIFIED | Forwards `Range` in, `response.status`/`Content-Range`/`Accept-Ranges` out. |
| `web/src/routes/api/v1/regions/[id]/download-dem/+server.ts` | Range-forwarding DEM proxy | ✓ VERIFIED | Byte-for-byte identical fix, DEM-specific path/filename intact. |
| `web/src/routes/api/v1/regions/[id]/download/server.test.ts` + `download-dem/server.test.ts` | vitest coverage | ✓ VERIFIED | 4 tests, all pass. |
| `app/lib/models/region_status.dart` | `PackageStatus.paused(3)/error(4)`, `RegionStatus.paused(4)/error(5)`, append-only | ✓ VERIFIED | Confirmed no renumbering; existing codes 0/1/2/3 unchanged. |
| `app/lib/util/region_file_path.dart` | region-id allow-list + path builders | ✓ VERIFIED | `assertValidRegionId` always precedes `p.join`; regex matches backend's `RegionIdSchema` verbatim. |
| `app/lib/util/disk_space_util.dart` | fail-closed free-space wrapper + pure margin decision | ✓ VERIFIED | `freeDiskSpaceBytes` never throws (try/catch → null); `hasEnoughSpace` pure and tested. |
| `app/lib/services/tile_repository_manager.dart` | resumable download engine, disk-check, validation, backgrounding pause | ✓ VERIFIED | 495 lines; `class TileRepositoryManager` present; all required methods/helpers present; `flutter analyze` clean. |
| `app/lib/provider/region/tile_repository_provider.dart` | construction-only manager seam + keepAlive status notifier | ✓ VERIFIED | `tileRepositoryManager` provider watches `objectBoxProvider` then `apiProvider`, no I/O at build; `TileRepositoryStatus` is `keepAlive`, delegates to the manager. |
| `app/lib/models/region_download_state.dart` | freezed UI-only per-region state | ✓ VERIFIED | `@freezed`, no JSON serialization, matches spec. |
| `app/test/services/tile_repository_manager_harness.dart` | on-device driver exercising all manager methods | ✓ VERIFIED | Present, compiles clean (`flutter analyze`), calls every public method + `localTilePathsForBounds`; confirmed absent from `router_provider.dart`/`main.dart` (zero production references). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `download/+server.ts` | Go backend `/regions/{id}/download` | `event.fetch` with forwarded `Range` | ✓ WIRED | Confirmed by source read + passing vitest assertion checking `event.fetch.mock.calls[0][1].headers.Range`. |
| `tile_repository_manager.dart` | SvelteKit `/regions/{id}/download[-dem]` | `Dio.download` + `FileAccessMode.append` + Range header | ✓ WIRED | `_requestPathFor` builds the path from the validated id (not the catalog's `/api/v1`-prefixed URL strings) — confirmed by source read matching the plan's explicit anti-double-prefix requirement. |
| `tile_repository_manager.dart` | `region_file_path.dart` | `regionVectorPath`/`regionDemPath`/`regionStorageDir` from validated id | ✓ WIRED | Confirmed at every call site (`startVectorDownload`, `startDemDownload`, `deleteRegion`). |
| `tile_repository_manager.dart` | `disk_space_util.dart` | `hasEnoughSpace` pre-write check | ✓ WIRED | Called before any directory/file creation in both download methods. |
| `tile_repository_manager.dart` | `pmtiles` `PmTilesArchive.fromFile` | post-download validation | ✓ WIRED | `_isValidPmTiles` gates every `.part` → final-path promotion. |
| `tile_repository_provider.dart` | `tile_repository_manager.dart` | notifier delegates to manager | ✓ WIRED | Every `TileRepositoryStatus` method calls `ref.read(tileRepositoryManagerProvider)...`. |
| `tile_repository_manager_harness.dart` | `tile_repository_provider.dart` | drives manager + refreshes catalog | ✓ WIRED | Confirmed by source read; harness uses `tileRepositoryManagerProvider` and `regionRepositoryProvider` directly. |

### Data-Flow Trace (Level 4)

This phase has no rendering/UI consumer yet (Phase 24/25 consume it later) — the relevant "data flow" is the download → validate → persist → query pipeline, traced above at the Key Link level. There is no hardcoded/static-empty return masking real behavior: `localTilePathsForBounds` genuinely iterates `_store.box<RegionEntity>().getAll()` and reads `localFilePath` off real `ToOne` targets, not a stub. The one thing this trace *cannot* confirm from source alone is whether `freeDiskSpaceBytes` resolves a real, sane byte count from the platform on a physical device, and whether a real interrupted HTTP transfer actually produces a non-zero `.part` file that resumes correctly — both are exactly the outstanding human-check items.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| SvelteKit Range-forwarding proxy tests | `cd web && npx vitest run src/routes/api/v1/regions` | 2 files, 4 tests, all pass | ✓ PASS |
| Flutter unit tests (all Phase 23 test files) | `cd app && flutter test test/services/tile_repository_manager_test.dart test/models/region_status_test.dart test/entities/region_entity_test.dart test/util/region_file_path_test.dart test/util/disk_space_util_test.dart test/models/region_download_state_test.dart` | 51 tests, all pass | ✓ PASS |
| Static analysis on every phase-touched file | `cd app && flutter analyze lib/services/tile_repository_manager.dart lib/util/region_file_path.dart lib/util/disk_space_util.dart lib/models/region_status.dart lib/entities/region_entity.dart lib/provider/region/tile_repository_provider.dart lib/models/region_download_state.dart test/services/tile_repository_manager_harness.dart` | "No issues found!" | ✓ PASS |
| Harness kept out of production navigation | `grep -rn "tile_repository_manager_harness" app/lib/` | zero matches | ✓ PASS |
| `disk_space_2` actually added only post-gate | `grep -n "disk_space_2" app/pubspec.yaml` | `disk_space_2: ^1.0.12` present | ✓ PASS |
| All claimed task commits exist | `git log --oneline --all \| grep -E "8f4a2540\|8cb6c19c\|1653214c\|3ec812ee\|b092e88c\|b167a1ab\|d5dc4200\|39a79f59\|71ac974a"` | all 9 commit hashes found | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this phase and none are referenced in any PLAN/SUMMARY file. Skipped — no runnable probes declared or conventional. The equivalent verification surface for this phase is the on-device driver harness (`app/test/services/tile_repository_manager_harness.dart`), which is a manual human-driven tool rather than an automated probe script, and is exactly the subject of the outstanding human-check items below.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| TILE-01 | 23-02, 23-04, 23-05 | `TileRepositoryManager` owns start/pause/resume/delete, decoupled from Trail | ✓ SATISFIED (code) | All methods present, no `Trail` import/reference anywhere in `tile_repository_manager.dart`. |
| TILE-02 | 23-01, 23-04 | Resumable downloads via Range + `FileAccessMode.append`, in-session only | ✓ SATISFIED (code) | Proxy fix verified by vitest; `resumePlanFor`/`_downloadResumable` verified by unit test + source read; real 206 round-trip is the "RESUME" human-check item. |
| TILE-03 | 23-03, 23-04 | Disk-space safety-margin check before each write | ✓ SATISFIED (code) | `hasEnoughSpace` fail-closed and tested; real platform free-space resolution is the "DISK REFUSAL" human-check item. |
| TILE-04 | 23-02, 23-04 | Backgrounding treated as deliberate pause | ✓ SATISFIED (code) | `AppLifecycleListener` wired; real OS signal delivery is the "BACKGROUNDING" human-check item. |
| TILE-05 | 23-05 | `localTilePathsForBounds` bbox query | ✓ SATISFIED (code) | `bboxOverlaps` unit-tested; end-to-end query against a real downloaded region is the "QUERY" human-check item. |
| DEM-01 | 23-04 | Per-region DEM toggle downloads pre-built DEM archive from catalog | ✓ SATISFIED (code) | `startDemDownload` present, independent request path/package; real independent behavior is the "DEM INDEPENDENCE" human-check item. |
| DEM-02 | 23-04, 23-05 | DEM tracked as its own `DownloadedTilePackage`, independent of vector | ✓ SATISFIED (code) | Confirmed via Phase 22's `ToOne<DownloadedTilePackageEntity>` x2 (re-verified, unchanged) plus this phase's independent lifecycle code paths. |

No orphaned requirements found — TILE-01..05, DEM-01, DEM-02 are the full requirement set mapped to Phase 23 in ROADMAP.md and REQUIREMENTS.md, and all seven are claimed and satisfied across the six plans at the code/static level. REQUIREMENTS.md's "Complete" markings for all seven IDs are consistent with this evidence, subject to the outstanding human-check.

### Anti-Patterns Found

None. Grepped every phase-touched file (`+server.ts` x2, `region_status.dart`, `region_file_path.dart`, `disk_space_util.dart`, `tile_repository_manager.dart`, `tile_repository_provider.dart`, `region_download_state.dart`, `tile_repository_manager_harness.dart`, `pubspec.yaml`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented"/"not available"/"coming soon" — zero matches. No empty-implementation stubs (`return null`/`return {}`/`=> {}`) found in any download/query/delete code path; every method has real, substantive logic matching its plan spec.

### Human Verification Required

This project runs `human_verify_mode: end-of-phase` (`.planning/config.json`), which deliberately suppressed mid-flight `checkpoint:human-verify` tasks in favor of one consolidated end-of-phase pass — this verification report *is* that pass. 23-06-SUMMARY.md explicitly states: *"Live on-device execution of the 5-behavior human-check is a required human follow-up step, deferred to this project's end-of-phase verification pass... To actually run the check: launch `flutter run -t test/services/tile_repository_manager_harness.dart` on a physical device..."* No evidence in any commit, SUMMARY, or test artifact indicates this device run has actually happened yet. The 5 items are listed in the frontmatter `human_verification` block above and repeated here:

### 1. RESUME (TILE-02)

**Test:** On a physical device against a backend with a `ready` region, start the region's vector download via the harness; partway through, toggle airplane mode so the transfer fails; restore connectivity and resume.
**Expected:** The logged byte count continues from the partial offset (not restarting at 0); the resumed request hits the server as `206` (not `200`); the final file validates and promotes to `downloaded`.
**Why human:** Requires a real network interruption and a real HTTP round-trip through the Range-forwarding proxy — the pure `resumePlanFor` unit test only proves the decision logic, not that a live interrupted transfer actually resumes correctly end-to-end.

### 2. DISK REFUSAL (TILE-03)

**Test:** On a device/emulator with little free space (or an inflated declared size), start a download that exceeds free space minus the 1.75x margin.
**Expected:** Refused with an `error`/blocked state; no partial file (or only a zero-byte one) is written.
**Why human:** Requires real platform-native `StatFs`/`NSFileManager` free-space resolution on an actual device filesystem — untestable by the pure `hasEnoughSpace` unit test alone.

### 3. BACKGROUNDING (TILE-04)

**Test:** Start a download and immediately background the app; return to foreground.
**Expected:** The region shows `paused` (not a silently dead transfer) and resumes cleanly.
**Why human:** Requires a real OS lifecycle signal (iOS suspension / Android Doze) delivered to `AppLifecycleListener` — cannot be triggered by static analysis or unit tests.

### 4. DEM INDEPENDENCE (DEM-01/02)

**Test:** Toggle the DEM download on/off independently of the vector package; delete one and confirm the other is unaffected.
**Expected:** Vector and DEM package status/lifecycle never affect one another.
**Why human:** Requires a live Store + real concurrent/sequential downloads against a real backend to observe true independence, not just structurally-separate code paths.

### 5. QUERY (TILE-05)

**Test:** With a region downloaded, tap "Query inside bbox" / "Query outside bbox" in the harness.
**Expected:** Inside-bbox query returns the region's real local file path(s); outside-bbox query returns nothing.
**Why human:** Requires a real downloaded region with a real on-disk file and live Store to confirm the full query path, not just the pure `bboxOverlaps` geometry test.

### Gaps Summary

No code-level gaps found. Every artifact this phase's six plans specified exists, is substantive, is correctly wired, and its pure/unit-testable logic passes (51 Flutter tests + 4 vitest tests, all green; `flutter analyze` clean across every touched file). All 7 requirement IDs (TILE-01..05, DEM-01, DEM-02) trace to real implementation, not stubs. No anti-patterns, no debt markers, no orphaned requirements.

The phase is not yet fully verified, however, because its headline behaviors — resumable download across a real interruption, real disk-space refusal, real backgrounding pause, real DEM independence, and a real bbox query against downloaded files — have only been exercised through pure-function unit tests and code review. The phase's own research (23-RESEARCH.md) rated platform disk-space and backgrounding semantics at MEDIUM confidence specifically because they are unconfirmed from docs alone, and Plan 06 built an on-device driver harness *for exactly this reason*, explicitly deferring the live run to this end-of-phase verification pass rather than skipping it. 23-06-SUMMARY.md documents that this live run has not happened yet. This is not a code gap to close with a new plan — it is a required human action (run the harness on a physical device and confirm the 5 listed behaviors) before Phase 23 can be considered fully done and before Phase 24 (Settings UI) builds user-facing trust on top of an unconfirmed download engine.

---

*Verified: 2026-07-22*
*Verifier: Claude (gsd-verifier)*
