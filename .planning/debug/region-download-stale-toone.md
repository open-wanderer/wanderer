---
status: diagnosed
trigger: "UAT re-verification (Phase 24 Settings — Offline Maps/Regions UI) Test 1, run immediately after gap-closure plan 24-03 shipped the disk-space fallback fix: on a physical Android device, pick a region that has NEVER been downloaded before, tap Download vector. Expected: transitions to `downloading` (progress bar advances), then pause/resume/delete work. Actual (verbatim): \"Package does not transition to downloading. It immediately goes into downloaded after clicking the download button.\" No error/log line this time. Secondary linked gap: Test 3 (watch disk-usage total while a download is paused mid-transfer) could not be exercised because the download never visibly reaches a downloading/paused state."
created: 2026-07-23T00:20:00Z
updated: 2026-07-23T00:55:00Z
---

## Current Focus

hypothesis: CONFIRMED — `SettingsOfflineRegionsScreen._buildActiveRow` derives its rendered `RegionStatus` from `region.status` (`RegionEntity.status` getter), which reads `region.vectorPackage.target` — an ObjectBox `ToOne` that lazily resolves ONCE per Dart object instance and then permanently caches that value (confirmed by reading the installed `objectbox` 5.3.1 package source). The `region` object held by the screen's `build()` (via `ref.watch(regionListNotifierProvider)`) is fetched once and its `vectorPackage.target` is first accessed (and cached as `null`) on the very first row render, before any download starts. `_save()` (the wrapper around every mutation, including `downloadVector`) only calls `ref.invalidate(regionListNotifierProvider)` ONCE, in a `finally` block AFTER the entire async `startVectorDownload` call resolves — not incrementally while it's in flight. So `region.status` is frozen at `notDownloaded` for the ENTIRE duration of the download (even though the ephemeral `TileRepositoryStatus`/`RegionDownloadState` provider IS correctly tracking `downloading` + live progress and correctly triggers row rebuilds via `ref.watch(tileRepositoryStatusProvider)`), and only becomes accurate the instant the whole operation finishes and the one terminal invalidate fires — producing exactly the reported symptom: UI jumps straight from "not downloaded" to "downloaded" with no visible `downloading` frame, no progress bar, no pause button.
test: Read full call chain end-to-end (UI row builder -> RegionEntity.status getter -> ToOne caching semantics in installed objectbox package source -> TileRepositoryManager status writes -> TileRepositoryStatus ephemeral provider -> _save()'s single terminal invalidate) and cross-referenced against git history to confirm 24-03 (the disk-space fix) never touched any of these files.
expecting: N/A — diagnosis phase complete, no further hypothesis testing needed (goal: find_root_cause_only).
next_action: Hand off to gap-closure planning for fix implementation.

## Symptoms

expected: Tapping "Download vector" on a never-before-downloaded region transitions the package status to `downloading` (progress bar advances, pause/delete controls appear/function), then to `downloaded` on completion.
actual: The UI never renders the `downloading` state at all — no progress bar, no pause button. The row appears to jump directly from "not downloaded" to "downloaded" the instant the tap handler resolves. (The download itself IS actually happening in the background over real time — this is a UI staleness bug, not a data/download bug — but from the user's perspective it looks like the download was skipped.)
errors: None reported. No exception, no log line — this is a NEW, silent symptom, distinct from the previously-diagnosed/fixed "freeDiskSpaceBytes gives: Specified path does not exist" bug (see `.planning/debug/region-download-diskspace.md`).
reproduction: On a real device, pick any region that has never been downloaded before (no `DownloadedTilePackageEntity` row yet), tap "Download vector". 100% reproducible — this is a deterministic caching mechanism, not a timing race.
started: Pre-existing since Phase 24 Plan 02 (commit `90b658ab`, "active region row — 6-state actions, combined progress, DEM toggle") introduced `_buildActiveRow`'s status-driven rendering. Was previously MASKED by the disk-space bug fixed in 24-03 (`region-download-diskspace.md`): before 24-03, every download failed near-instantly with `error` (same "no visible intermediate state" defect, just landing on a different terminal status), so nobody would expect to see a `downloading` frame for a near-instant failure. 24-03 fixed the disk-space bug (commit `8be39c5a`, touched ONLY `app/lib/util/disk_space_util.dart` + pubspec files — confirmed via `git show --stat`), so downloads now genuinely succeed and take real time — which is what makes this pre-existing staleness bug visibly obvious for the first time.

## Eliminated

- hypothesis: 24-03's disk-space device-wide fallback itself is the new root cause (e.g. `hasEnoughSpace` now always returns `true`, short-circuiting some downstream check that previously gated correct behavior).
  evidence: `git show --stat 8be39c5a` shows the 24-03 commit touched only `app/lib/util/disk_space_util.dart`, `app/pubspec.lock`, `app/pubspec.yaml` — it never touched `tile_repository_manager.dart`, `tile_repository_provider.dart`, `region_entity.dart`, or `settings_offline_regions_screen.dart`. The commit message itself states "tile_repository_manager.dart untouched; both call sites now backed by the falling-back wrapper with no reordering required." `resolveFreeDiskSpaceBytes`'s fallback logic (disk_space_util.dart:55-81) is a pure, narrowly-scoped orchestrator that either returns a real byte count or `null` — it does not touch status fields, package entities, or UI state. Read in full; no plausible mechanism by which it could cause a UI-layer staleness bug.
  timestamp: 2026-07-23T00:35:00Z

- hypothesis: The download never actually starts / TileRepositoryManager silently skips straight to writing `PackageStatus.downloaded` without downloading anything (e.g. a bug in `startVectorDownload`'s status-write sequence).
  evidence: Read `startVectorDownload` (tile_repository_manager.dart:99-177) end-to-end. The sequence is correct: `_updatePackageStatus(package, status: PackageStatus.downloading)` (line 127) is written and persisted BEFORE the directory is created (129) and BEFORE `_downloadResumable` (139) actually performs the Dio network transfer with progress callbacks; only after `_isValidPmTiles` (146) passes does it rename `.part` -> final and write `PackageStatus.downloaded` (153-159). No shortcut path exists that could reach `downloaded` without a real transfer. The backend-side status transitions are correct and in the right order — the bug is entirely in how the UI observes them, not in the manager itself.
  timestamp: 2026-07-23T00:40:00Z

## Evidence

- timestamp: 2026-07-23T00:22:00Z
  checked: app/lib/services/tile_repository_manager.dart (startVectorDownload/startDemDownload, full file re-read)
  found: |
    Status-write sequence per download is: create-or-get package row (defaults `PackageStatus.notDownloaded`, persisted) -> write `PackageStatus.downloading` (persisted) -> create region storage dir -> `_downloadResumable` (real Dio network transfer with `onProgress` callbacks) -> validate pmtiles -> rename .part->final -> write `PackageStatus.downloaded` (persisted). Every status write goes through `_store.runInTransaction` on a `RegionEntity`/`DownloadedTilePackageEntity` fetched via `_regionById(id)` — a FRESH ObjectBox query, a DIFFERENT Dart object instance than whatever the UI is holding.
  implication: The backend state machine and its persisted writes are correct and complete; the download genuinely happens and genuinely takes real wall-clock time. The bug must be in how a separate, stale UI-side `RegionEntity` Dart instance fails to observe these writes.

- timestamp: 2026-07-23T00:26:00Z
  checked: app/lib/entities/region_entity.dart (RegionEntity.status getter, lines 103-123) and app/lib/entities/downloaded_tile_package_entity.dart
  found: |
    `RegionEntity.status` is a pure computed getter with no caching of its own — it reads `vectorPackage.target?.status` fresh on every access. `vectorPackage` is `final vectorPackage = ToOne<DownloadedTilePackageEntity>();` (line 86) — an ObjectBox `ToOne` relation field belonging to whichever specific `RegionEntity` Dart object it's attached to.
  implication: The getter itself is correct/pure; the staleness must come from the `ToOne.target` accessor's own caching behavior, not from this getter's logic.

- timestamp: 2026-07-23T00:30:00Z
  checked: ~/.pub-cache/hosted/pub.dev/objectbox-5.3.1/lib/src/relations/to_one.dart (full file, the installed package's actual ToOne implementation)
  found: |
    `ToOne.target` getter (lines 78-94): "Uses lazy initialization, so on first access this will read the target object from the database." On first access, if internal `_value._state == _ToOneState.lazy`, it queries the store ONCE and reassigns `_value` to `_ToOneValue.stored(...)` or `.unresolvable(...)` — a NON-lazy terminal state. Every subsequent call to `.target` on that SAME `ToOne` instance (i.e. the same parent `RegionEntity` Dart object) just returns the already-cached `_value._object` (line 93) without ever re-querying the store. There is no TTL, no invalidation hook, no listener — the cache is permanent for the lifetime of that Dart object.
  implication: Confirms, at the library-source level, that `region.vectorPackage.target` (and therefore `region.status`) is FROZEN at whatever value it first resolved to, for as long as that specific `RegionEntity` Dart instance is alive — regardless of what any other `RegionEntity` instance (fetched via a separate query, e.g. inside `TileRepositoryManager._regionById`) writes to the same underlying database row in the meantime. Matches this codebase's own documented awareness of the issue (see next evidence entry).

- timestamp: 2026-07-23T00:33:00Z
  checked: app/lib/provider/region/region_provider.dart (RegionListNotifier, lines 157-174) and app/lib/routes/settings_offline_regions_screen.dart (_save, lines 507-531; _buildActiveRow, lines 309-317)
  found: |
    `RegionListNotifier`'s own doc comment (lines 162-164) states explicitly: "all region mutations flow through `TileRepositoryStatus` ... whose callers must `ref.invalidate(regionListNotifierProvider)` after each mutation (RESEARCH.md Pitfall 2 -- ObjectBox `ToOne.target` caches per-instance after first read)." This confirms the team was already aware of the exact caching mechanism found above.
    `_save()` (settings_offline_regions_screen.dart:513-531) is the single wrapper used by every mutation handler (`_onDownloadVector`, `_onPause`, `_onResume`, `_onRetry`, `_onDemToggle`, `_onDeleteRegion`). It does `try { await op(); } catch (_) { ... } finally { if (mounted) ref.invalidate(regionListNotifierProvider); }` — the invalidate call happens EXACTLY ONCE, only after `op()` (e.g. `downloadVector(region.id)`, which itself `await`s the entire multi-second/minute `startVectorDownload` call) fully resolves.
    `_buildActiveRow` (line 312) unconditionally computes `final status = region.status;` at the top of every build using the `region` object captured from `ref.watch(regionListNotifierProvider)` at the top-level `build()` (line 95) — the SAME Dart `RegionEntity` instance across every rebuild until `regionListNotifierProvider` itself re-runs its `build()` (which only happens on explicit invalidate, since it depends only on `objectBoxProvider`, a stable singleton).
    Separately, `ref.watch(tileRepositoryStatusProvider)[region.id]` (line 311) DOES correctly update on every `onProgress` callback during the download (`tile_repository_provider.dart:34-64`), triggering `_buildActiveRow` to rebuild repeatedly throughout the download — but each of those rebuilds recomputes `region.status` from the SAME stale, ToOne-cached `region` object, so `status` never changes from `notDownloaded` during the whole download.
  implication: This is the root cause mechanism, fully traced end-to-end: the UI mitigation for the ToOne-caching pitfall (`ref.invalidate` in `_save`'s `finally`) is real and intentional, but too coarse — it only fires once, at the very end of the whole mutation, not incrementally as the mutation's own status transitions happen. The progress-tracking ephemeral provider (`tileRepositoryStatusProvider`) correctly drives rebuilds during the download, but the rendering logic for status-dependent UI (progress bar visibility at line 357, trailing action icon in `_buildTrailingActions` switching on `status`) reads the frozen `region.status` rather than the live, already-correct `downloadState?.status`/`downloadState != null` ephemeral signal. Net effect: the row silently stays rendered as "not downloaded" (download button still shown, tappable but no-ops via `downloadVector`'s own re-entry guard) for the ENTIRE download duration, then snaps directly to the final persisted status (`downloaded`, or previously `error` pre-24-03) the moment the terminal invalidate fires.

- timestamp: 2026-07-23T00:45:00Z
  checked: git show --stat 8be39c5a (the 24-03 fix commit) and git log --oneline --follow -- app/lib/routes/settings_offline_regions_screen.dart
  found: |
    24-03 touched only `app/lib/util/disk_space_util.dart`, `app/pubspec.lock`, `app/pubspec.yaml`. `settings_offline_regions_screen.dart`'s history shows only two prior commits: `90b658ab` (24-02, "active region row — 6-state actions, combined progress, DEM toggle") and `53fe40c2` (24-02, scaffold) — both predate 24-03 entirely.
  implication: This bug is NOT a regression introduced by the disk-space fix. It has been present since 24-02 and was structurally invisible until 24-03 made downloads actually succeed/take real time — before that, the near-instant `error` transition (from the disk-space bug) looked superficially similar (no visible `downloading` frame) and was attributed entirely to the disk-space defect, masking this separate, independent UI-staleness defect underneath it.

## Resolution

root_cause: |
  `SettingsOfflineRegionsScreen._buildActiveRow` renders its download-progress
  UI (progress bar visibility, trailing action icon/button) from
  `region.status` (`RegionEntity.status`, region_entity.dart:103-123), which
  is computed from `region.vectorPackage.target` — an ObjectBox `ToOne`
  relation that lazily resolves ONCE per Dart object instance and then
  PERMANENTLY caches that value for the lifetime of that instance (confirmed
  by reading the installed `objectbox` 5.3.1 package source,
  `~/.pub-cache/hosted/pub.dev/objectbox-5.3.1/lib/src/relations/to_one.dart`
  lines 78-94 — there is no cache invalidation, TTL, or listener).

  The screen fetches its `region` list once via
  `ref.watch(regionListNotifierProvider)` (settings_offline_regions_screen.dart:95)
  and reuses the SAME `RegionEntity` Dart instances across every rebuild
  until that provider is explicitly invalidated. `_save()`
  (settings_offline_regions_screen.dart:507-531), the wrapper around every
  mutation including `downloadVector`, only calls
  `ref.invalidate(regionListNotifierProvider)` ONCE, in a `finally` block
  AFTER the entire `await op()` (the full, multi-second/minute
  `TileRepositoryManager.startVectorDownload` call) resolves — not
  incrementally as the download's own internal status transitions
  (`notDownloaded` -> `downloading` -> `downloaded`) happen on a SEPARATE,
  freshly-queried `RegionEntity`/`DownloadedTilePackageEntity` instance inside
  `TileRepositoryManager` (fetched via its own `_regionById(id)` query, a
  different Dart object than the UI's).

  So for the entire duration of the download, `region.status` on the UI's
  held instance stays frozen at whatever it first resolved to (`notDownloaded`,
  cached the first time that row was ever rendered, before any download
  started) — even though the SEPARATE, correctly-updating ephemeral
  `TileRepositoryStatus` provider (`tile_repository_provider.dart`) IS
  tracking `downloading` + live progress and DOES trigger `_buildActiveRow`
  to rebuild repeatedly via `ref.watch(tileRepositoryStatusProvider)`. Each of
  those rebuilds still recomputes `region.status` from the same
  ToOne-frozen instance, so the progress bar (gated on
  `status == RegionStatus.downloading`, line 357) and the trailing action
  icon (switched on `status` in `_buildTrailingActions`, line 438) never
  update. The moment the download finishes and `_save`'s single terminal
  invalidate fires, `regionListNotifierProvider` re-runs its `build()`
  (region_provider.dart:168-173), producing a BRAND NEW `RegionEntity`
  instance whose `ToOne` is freshly (and now correctly) resolved to
  `downloaded` — this is the very first moment the UI ever reflects reality,
  so it appears to "jump straight to downloaded."

  This is NOT a regression from 24-03's disk-space fix (confirmed via
  `git show --stat 8be39c5a`: that commit touched only
  `disk_space_util.dart` + pubspec files). It has existed since 24-02
  introduced `_buildActiveRow` (commit `90b658ab`) and was masked until now:
  before 24-03, every download failed near-instantly with `error` (the
  disk-space bug), which is ALSO a "no visible downloading frame" outcome —
  so this separate defect was invisible underneath that one. 24-03 made
  downloads genuinely succeed and take real wall-clock time, which is what
  finally exposed this pre-existing UI-staleness bug. The codebase's own doc
  comments (region_provider.dart:162-164) show the team was already aware of
  the general ObjectBox `ToOne` caching pitfall and added the `_save()`
  terminal invalidate as a mitigation — but that mitigation is too coarse
  (once, at the end) to cover a long-running multi-state mutation like a
  region download.

fix: |
  NOT IMPLEMENTED — diagnosis only, per task instructions (goal:
  find_root_cause_only). Suggested fix direction for gap-closure planning:

  Most targeted, minimal-blast-radius option: in `_buildActiveRow`, prefer
  the ephemeral `downloadState` (from `tileRepositoryStatusProvider`) over
  the ObjectBox-derived `region.status` whenever `downloadState != null`
  (i.e. a mutation is actively in flight for this region) — since
  `TileRepositoryStatus` already correctly tracks `RegionStatus.downloading`
  and live per-package progress and is NOT subject to ObjectBox `ToOne`
  caching (it's a plain in-memory Riverpod state map). Fall back to
  `region.status` only when `downloadState == null` (no mutation in flight,
  i.e. terminal/idle state) — which is exactly when the ToOne-cached value
  is guaranteed fresh again anyway, because `_save`'s terminal invalidate
  will have just run. This requires no change to `TileRepositoryManager` or
  the persistence layer, and does not touch the `_save()` invalidate
  contract (still needed for terminal-state correctness, e.g. `error`/
  `paused` after the ephemeral entry is cleared).

  Alternative (more invasive, touches more call sites): have
  `TileRepositoryManager` accept an optional per-transition callback (or have
  `TileRepositoryStatus.downloadVector`/`downloadDem`/`resume` call
  `ref.invalidate(regionListNotifierProvider)` — or a cheaper targeted
  re-fetch of just this one region — immediately after each of
  `TileRepositoryManager`'s own status writes, not just once at the end).
  Riskier: more invalidation points mean more re-renders and more surface
  area for the ToOne pitfall to resurface elsewhere if a future call site
  forgets the pattern.

  Either direction should also address the linked Test 3 gap (pause
  mid-transfer untestable) as a side effect, since the trailing action icon
  (`_buildTrailingActions`) currently switches on the same frozen `status`
  and would start correctly showing the pause button once `downloading` is
  observable.

verification: N/A — not yet fixed; this session ends at diagnosis per
  goal: find_root_cause_only. Verification will happen in the fix/gap-closure
  session.

files_changed: []
