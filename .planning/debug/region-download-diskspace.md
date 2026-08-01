---
status: diagnosed
trigger: "On-device UAT (Phase 24 Settings — Offline Maps/Regions UI), tests 2 and 3: Starting a region's vector download, and separately starting a region's DEM download, both fail immediately. Device log: freeDiskSpaceBytes gives: \"Specified path does not exist\" for /data/user/0/com.openwanderer.wanderer/app_flutter/regions/munich"
created: 2026-07-22T13:48:00Z
updated: 2026-07-22T14:05:00Z
---

## Current Focus

hypothesis: CONFIRMED — `freeDiskSpaceBytes(regionStorageDir(root, id))` is called before the region's storage directory is ever created, so on a region's first-ever download the path-specific disk-space query throws "path does not exist", is swallowed to `null`, and `hasEnoughSpace` fails closed — deterministically, every time, for any never-before-downloaded region. Not a low-disk-space edge case.
test: Read current source of both files line-by-line, cross-referenced against the installed `disk_space_2` v1.0.12 plugin's native Android source, `region_file_path.dart`, and both existing test files.
expecting: N/A — diagnosis phase complete, no further hypothesis testing needed.
next_action: Hand off to gap-closure planning for fix implementation (goal: find_root_cause_only — do not implement fix in this session).

## Symptoms

expected: Tapping "Download vector" / "Download DEM" for a region that has never been downloaded before starts a resumable download and transitions the package to `downloading`.
actual: The package status is immediately set to `error` and the download never starts, for both the vector and the DEM package, on the very first download attempt of a region.
errors: |
  freeDiskSpaceBytes gives: "Specified path does not exist" for
  /data/user/0/com.openwanderer.wanderer/app_flutter/regions/munich
reproduction: On a real device (not simulator/emulator with a permissive filesystem), pick any region whose `regions/<id>/` storage directory has never been created (i.e. no prior download attempt for that region), and tap "Download vector" or "Download DEM". Fails 100% of the time.
started: Always broken — present since `TileRepositoryManager.startVectorDownload`/`startDemDownload` were implemented (Phase 23, TILE-03 disk-space pre-check). Never caught by unit tests; only surfaces on-device (real filesystem semantics), which is why it was found in Phase 24 on-device UAT rather than in Phase 23's own on-device harness pass or CI.

## Eliminated

- hypothesis: The `disk_space_2` plugin itself is broken/misbehaving (e.g. returns null spuriously) rather than throwing on a legitimately missing path.
  evidence: Read the installed plugin's Android native source directly (`~/.pub-cache/hosted/pub.dev/disk_space_2-1.0.12/android/.../DiskSpace_2Plugin.kt`, lines 88-97): `getFreeDiskSpaceForPath(path)` constructs `android.os.StatFs(path)` directly on the caller-supplied path with no existence check of its own — the underlying native `statvfs()` call throws when `path` doesn't exist, which the plugin's `executeAsync` wrapper (lines 62-74) catches generically and surfaces as a `DISK_SPACE_ERROR` `PlatformException`. This is expected/documented plugin behavior, not a plugin bug — matches `disk_space_util.dart`'s own doc comment ("the path-specific query's documented 'path does not exist' exception").
  timestamp: 2026-07-22T13:58:00Z

- hypothesis: Directory creation is missing entirely (never called at all) for the region path.
  evidence: `Directory(regionStorageDir(root, id)).createSync(recursive: true)` IS present in both `startVectorDownload` (tile_repository_manager.dart:129) and `startDemDownload` (tile_repository_manager.dart:214) — it exists in the code, but runs AFTER the disk-space check (lines 116/201) that fails first and returns early (lines 117-124, 202-209), so the creation call is simply never reached on a region's first download.
  timestamp: 2026-07-22T14:00:00Z

## Evidence

- timestamp: 2026-07-22T13:52:00Z
  checked: app/lib/services/tile_repository_manager.dart (full file, 533 lines)
  found: |
    `startVectorDownload` (lines 99-177): line 116 `final free = await freeDiskSpaceBytes(regionStorageDir(root, id));` runs BEFORE line 129 `Directory(regionStorageDir(root, id)).createSync(recursive: true);`. If `hasEnoughSpace` returns false (lines 117-124), the method sets the package to `PackageStatus.error` and `return`s — line 129 (directory creation) is never reached.
    `startDemDownload` (lines 184-257) has the byte-for-byte identical structure: line 201 disk-space check, line 214 directory creation, same early-return-before-creation ordering.
  implication: Confirms the reported diagnosis exactly, in both call sites — explains why UAT tests 2 AND 3 (vector AND DEM) both fail identically.

- timestamp: 2026-07-22T13:54:00Z
  checked: app/lib/util/disk_space_util.dart (full file, 66 lines)
  found: |
    `freeDiskSpaceBytes` (lines 33-45) wraps `DiskSpace.getFreeDiskSpaceForPath(forPath)` in a try/catch that returns `null` on ANY exception (line 43), with a comment explicitly acknowledging "the path-specific query's documented 'path does not exist' exception" is one of the things being swallowed (line 30).
    `hasEnoughSpace` (lines 58-65): `if (freeBytes == null) return false;` (line 63) — TILE-03's deliberate fail-closed contract.
    IMPORTANT ADDITIONAL FINDING: the function's own doc comment (lines 22-24) says it "falls back to the device-wide free space when omitted or when the path-specific query fails" — but the actual implementation does NOT do this. The catch block (lines 40-44) unconditionally returns `null`; it never retries with the parameterless (device-wide) query. The documented contract and the implementation disagree.
  implication: The bug isn't just "check runs before directory exists" — the function that's supposed to gracefully degrade to a device-wide query on path-specific failure was never actually implemented to do so, despite its own doc comment claiming it does. This is the natural place to fix the bug: at the query wrapper, matching its already-written and already-reviewed contract, rather than only reordering the two call sites.

- timestamp: 2026-07-22T13:56:00Z
  checked: app/lib/util/region_file_path.dart (full file, 57 lines)
  found: |
    `regionStorageDir(root, id)` = `p.join(root, 'regions', assertValidRegionId(id))` (line 41). `root` is always `(await getApplicationDocumentsDirectory()).path` at both call sites in tile_repository_manager.dart (lines 112, 197) — the app's documents root, which is created by the OS/Flutter runtime before the app can run and is therefore always guaranteed to exist. Only the `regions/<id>/` subdirectory is missing on a region's first download.
  implication: `root` (or any other path known to exist within the app's own storage, e.g. no path at all / device-wide) is a safe, always-valid fallback query target — it's on the same logical volume as `regionStorageDir(root, id)` would be once created.

- timestamp: 2026-07-22T13:59:00Z
  checked: ~/.pub-cache/hosted/pub.dev/disk_space_2-1.0.12/android/src/main/kotlin/com/example/disk_space_2/DiskSpace_2Plugin.kt (full file, 110 lines)
  found: |
    `getFreeDiskSpaceForPath(path)` (lines 88-97): `StatFs(path)` on the raw caller-supplied path, no existence guard — throws for a nonexistent path via the native `statvfs()` call, caught generically by `executeAsync` (lines 62-74) and turned into a `PlatformException("DISK_SPACE_ERROR", ...)`.
    `getFreeDiskSpace()` (lines 77-86, the no-path/device-wide variant): `StatFs(Environment.getExternalStorageDirectory().path)` — a path that always exists on-device, so this variant is structurally immune to the "path does not exist" failure.
  implication: Confirms both the mechanism of the failure (native `StatFs` construction throws on missing path) AND that the device-wide query (already implemented and already used at the `forPath == null` call site in `freeDiskSpaceBytes`) is a robust, already-proven fallback target — no new plugin capability is needed, only wiring the existing no-arg call as a fallback.

- timestamp: 2026-07-22T14:02:00Z
  checked: app/test/util/disk_space_util_test.dart (full file, 79 lines) and app/test/services/tile_repository_manager_test.dart (94 lines) and app/test/services/tile_repository_manager_harness.dart (324 lines)
  found: |
    `disk_space_util_test.dart`'s own header comment (lines 4-11) states unit tests deliberately cover ONLY the pure `hasEnoughSpace` function — `freeDiskSpaceBytes` "needs a device/platform channel and is covered by Plan 06's on-device human-check instead."
    `tile_repository_manager_test.dart` only unit-tests two pure/static helpers, `resumePlanFor` and `bboxOverlaps` (grep for freeDiskSpaceBytes/hasEnoughSpace/createSync/regionStorageDir in this file returns zero matches) — `startVectorDownload`/`startDemDownload` are never exercised by the automated suite (no Dio/Store/filesystem mocking present).
    `tile_repository_manager_harness.dart` is the only place these methods are exercised end-to-end, and it's an on-device-only manual driver (not part of `flutter test`, not part of CI).
  implication: This bug is structurally invisible to the existing automated test suite — it can only be caught by the on-device harness or real UAT, exactly how it was found. Any fix should not regress this gap; ideally the retry/fallback logic added to `freeDiskSpaceBytes` should be structured so its decision logic (not the platform call itself) becomes unit-testable.

## Resolution

root_cause: |
  Two related defects compound to cause the failure:

  1. (Immediate cause, both call sites) `TileRepositoryManager.startVectorDownload`
     (app/lib/services/tile_repository_manager.dart:116) and `startDemDownload`
     (app/lib/services/tile_repository_manager.dart:201) both call
     `freeDiskSpaceBytes(regionStorageDir(root, id))` BEFORE the corresponding
     `Directory(regionStorageDir(root, id)).createSync(recursive: true)` call
     (lines 129 and 214 respectively, in the success path only). For a region
     that has never been downloaded before, `regions/<id>/` does not yet exist
     on the device filesystem, so the path-specific disk-space query is asked
     to stat a nonexistent directory.

  2. (Root cause, shared wrapper) `freeDiskSpaceBytes`
     (app/lib/util/disk_space_util.dart:33-45) is the sole gate between that
     platform-level failure and the caller. The underlying `disk_space_2`
     plugin's Android implementation constructs `StatFs(path)` directly
     on the caller-supplied path (confirmed by reading
     ~/.pub-cache/hosted/pub.dev/disk_space_2-1.0.12/android/.../DiskSpace_2Plugin.kt:88-97)
     and throws when the path doesn't exist. `freeDiskSpaceBytes`'s catch
     block (disk_space_util.dart:40-44) swallows this (and every other)
     exception to `null` unconditionally — even though the function's OWN doc
     comment (disk_space_util.dart:22-24) already documents an intended
     fallback to the device-wide query "when the path-specific query fails,"
     that fallback was never actually implemented.

  `hasEnoughSpace(freeBytes: null, ...)` then fails closed (`false`,
  disk_space_util.dart:63 — TILE-03's deliberate contract), so the package is
  marked `PackageStatus.error` and the method returns before the directory is
  ever created. This is not a low-disk-space edge case: it is the
  unconditional first-download path for every region, on every real device
  (the plugin's platform-channel exception only occurs on real device
  filesystems, which is why Phase 23's own automated tests — deliberately
  scoped to skip `freeDiskSpaceBytes` per disk_space_util_test.dart's header
  comment — never caught it, and why it surfaced only in on-device UAT).

fix: |
  NOT IMPLEMENTED — diagnosis only, per task instructions. Recommended
  direction for gap-closure planning:

  Primary fix (root-cause level, single change point): implement the
  fallback `freeDiskSpaceBytes`'s doc comment already promises. In the
  catch block (disk_space_util.dart:40-44), when `forPath != null` and the
  path-specific query throws, retry with the parameterless device-wide query
  (`freeDiskSpaceBytes()` / `DiskSpace.getFreeDiskSpace`) before giving up and
  returning `null`. This is safe because `regionStorageDir(root, id)` is
  always a subdirectory of `root` (`getApplicationDocumentsDirectory()`),
  which is always on the same logical volume/filesystem as whatever the
  device-wide query already targets (confirmed: the plugin's own no-path
  variant already queries `Environment.getExternalStorageDirectory()`, a
  different-but-same-volume path, and this pattern is already trusted
  elsewhere in this same file for the `forPath == null` case) — free space is
  effectively filesystem-wide for a single logical volume, so a
  not-yet-existing subdirectory and its guaranteed-to-exist parent report the
  same number.

  This fix requires no change to call-site ordering in
  tile_repository_manager.dart, applies uniformly to both call sites (and any
  future caller) through the shared wrapper, and does not touch
  `hasEnoughSpace` — so the TILE-03 fail-closed guarantee for genuinely low
  disk space is fully preserved: if BOTH the path-specific AND the
  device-wide query fail, `freeDiskSpaceBytes` still returns `null` and
  `hasEnoughSpace` still fails closed.

  Complementary (optional, defense-in-depth) fix: also reorder each call site
  so `Directory(regionStorageDir(root, id)).createSync(recursive: true)` runs
  before the disk-space check. Not required if the primary fix lands, and has
  its own minor tradeoff noted below.

verification: N/A — not yet fixed; this session ends at diagnosis per
  goal: find_root_cause_only. Verification will happen in the fix/gap-closure
  session.

files_changed: []
