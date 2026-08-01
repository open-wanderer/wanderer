---
status: diagnosed
trigger: "26-04-dem-status-not-reflected: DEM package downloaded via missing-coverage sheet does not show as 'downloaded' in Settings -> Offline Regions, while Vector (same flow) correctly shows downloaded. Region byte total includes DEM size."
created: 2026-07-24T13:20:00Z
updated: 2026-07-24T13:40:00Z
---

## Current Focus

hypothesis: CONFIRMED — TileRepositoryManager.startVectorDownload and startDemDownload each hold their own stale RegionEntity snapshot (separate `_regionById(id)` query per function); startVectorDownload performs a SECOND, late `box.put(region)` after successful completion (to set lastDownloadedVersion) using its stale snapshot, which full-row-overwrites the region row (ObjectBox has no partial-update/dirty-tracking — objectToFB always writes both vectorPackage.targetId and demPackage.targetId). When vector+DEM download concurrently for the same region (missing-coverage sheet's default vector-checked + dem-checked selection), DEM's relation FK (set early and fast, before the archive itself even downloads) gets clobbered back to 0/unset by vector's late write, which landed later (vector archives are the slower download). The DEM .pmtiles file itself is on disk (hence disk-usage total, which scans the filesystem directly via region_file_path.dart, correctly includes it) but region.demPackage.target is severed, so Settings reads PackageStatus.notDownloaded.
test: n/a (goal: find_root_cause_only — root cause confirmed via direct source reading, no live repro needed)
expecting: n/a
next_action: none — investigation complete, returning ROOT CAUSE FOUND

## Symptoms

expected: After downloading a region's DEM package via the missing-coverage sheet (missing_coverage_sheet.dart -> trail_download_state_provider.dart's downloadDem call), the DEM package's status in Settings -> Offline Regions (settings_offline_regions_screen.dart) shows "downloaded", identically to how the Vector package (downloadVector) correctly shows "downloaded" after the same flow.
actual: "Eventhough I select dem data in the sheet and it downloads properly, the dem data shows as \"not downloaded\" in the settings screen. The total at the top includes it. The vector data shows up correctly as \"downloaded\"."
errors: None reported by user
reproduction: Trigger the missing-coverage sheet for a trail with a missing region, check the DEM checkbox for that region (leave/check Vector too), tap Download. After it completes, navigate to Settings -> Offline Regions and look at that region's DEM status.
started: Discovered during on-device UAT after Phase 26 (trail-download-guard) + gap-closure plan 26-04 executed.

## Eliminated

- hypothesis: "Settings screen reads a different field/source for DEM status than for the byte total, so the mismatch is purely a display/read bug in settings_offline_regions_screen.dart"
  evidence: "settings_offline_regions_screen.dart's _buildDemTile reads region.demPackage.target?.status — the exact same ObjectBox relation field the disk-usage util's underlying region (via regionListNotifierProvider) also came from. The disk-usage util (region_disk_usage_util.dart) independently scans the filesystem directly (File(regionDemPath(...)).existsSync()/.lengthSync()), never touching the ToOne relation at all — so the two code paths aren't reading 'the same status from different derivations', they're reading fundamentally different sources (relation vs filesystem) by design (documented in region_disk_usage_util.dart's own doc comment). This is consistent with the relation itself being wrong in the persisted store, not a display-layer bug."
  timestamp: 2026-07-24T13:35:00Z
- hypothesis: "Stale ObjectBox ToOne ~in-memory~ ~ Dart-object~ cache in a long-lived RegionEntity instance held by the Settings screen (the known 'ToOne caches per-instance after first read' issue documented in region_tile_status_util.dart, already fixed once for the Settings-driven download flow)"
  evidence: "The user navigates to Settings AFTER the download completes (not mid-download), and regionListNotifierProvider is NOT keepAlive — it has no watchers during the guard's background download, so it is disposed and rebuilds from a completely fresh store.box<RegionEntity>().getAll() query the moment Settings mounts and calls ref.watch(regionListNotifierProvider) for the first time. Fresh RegionEntity instances have never had .target read before, so there is no stale per-instance ToOne cache to explain this case — the persisted DB row itself must be wrong."
  timestamp: 2026-07-24T13:36:00Z

## Evidence

- timestamp: 2026-07-24T13:25:00Z
  checked: app/lib/services/tile_repository_manager.dart — startVectorDownload vs startDemDownload
  found: "startVectorDownload captures `final region = _regionById(id);` ONCE at function entry, and on successful completion does a SECOND persistence step not present in startDemDownload: `region.lastDownloadedVersion = region.version; _store.box<RegionEntity>().put(region);` — using the SAME stale `region` object captured at function entry, never re-queried. startDemDownload has no equivalent final region-level write (matches the documented 'DEM has no staleness concept' design intent), so DEM's ONLY region-row write is the early one inside `_getOrCreatePackage` (to link the newly-created demPackage row)."
  implication: "Vector's download path writes the region row TWICE (early link + late lastDownloadedVersion); DEM's writes it ONCE (early link only). This is the exact asymmetry flagged as hypothesis (b) in the symptom notes."

- timestamp: 2026-07-24T13:28:00Z
  checked: app/lib/services/tile_repository_manager.dart — _getOrCreatePackage
  found: "_getOrCreatePackage(region, toOne) sets toOne.target = created (a brand-new, unpersisted DownloadedTilePackageEntity) then does `_store.box<RegionEntity>().put(region)` to cascade-insert the new package and link its FK on the region row. Called independently for vectorPackage (inside startVectorDownload, using ITS OWN region snapshot) and demPackage (inside startDemDownload, using A SEPARATE region snapshot from a separate _regionById(id) query) — two different Dart object instances for the same underlying row, each capturing only the relation state that existed at ITS OWN query time."
  implication: "Since startVectorDownload and startDemDownload are both invoked fire-and-forget nearly simultaneously (trail_download_state_provider.dart's regionFutures.addAll loop) when a region has both packages checked, each function's `region` object is a point-in-time snapshot that never reflects writes made by the OTHER concurrently-running function afterward."

- timestamp: 2026-07-24T13:30:00Z
  checked: app/lib/objectbox.g.dart — RegionEntity's generated objectToFB (the actual box.put() serialization)
  found: "objectToFB unconditionally writes ALL fields on every put, including `fbb.addInt64(17, object.vectorPackage.targetId); fbb.addInt64(18, object.demPackage.targetId);` — there is no dirty-tracking or partial/diff update. Any box.put(region) call replaces the ENTIRE row's stored values, including both relation FKs, with whatever the in-memory Dart object currently holds for them."
  implication: "This is the missing mechanical link that turns 'two stale snapshots' into an actual data-loss bug: when startVectorDownload's LATE put (region.lastDownloadedVersion write, using its function-entry-stale snapshot) fires AFTER startDemDownload's EARLY put has already linked demPackage in the real DB, the late vector put's stale snapshot still has demPackage.targetId == 0 (unset, from before DEM's download even started) — so it overwrites the just-established demPackageId back to 0/null, severing the relation. Vector archives are the larger/slower download of the two, so vector's completion (and therefore its late put) reliably lands AFTER DEM's fast early-link put — explaining why DEM (not vector) is the one that ends up severed, matching the exact asymmetry reported (\"vector shows correctly, DEM doesn't\")."

- timestamp: 2026-07-24T13:32:00Z
  checked: app/lib/util/region_disk_usage_util.dart, app/lib/util/region_file_path.dart
  found: "regionDiskUsageBytes(root, region) computes bytes by checking `File(regionVectorPath(root,id))`/`File(regionDemPath(root,id))` (and their `.part` siblings) directly on the filesystem — it never reads region.demPackage.target or DownloadedTilePackageEntity.sizeBytesOnDisk. It only uses `region.id` to build the path."
  implication: "This explains the 'total at the top includes it' half of the symptom: the DEM archive genuinely finished downloading and was renamed to its final on-disk path (dem.pmtiles) by TileRepositoryManager BEFORE the relation got clobbered — the file write and the relation write are two separate steps, and only the latter was lost. The orphaned DownloadedTilePackageEntity row (status=downloaded, correct sizeBytesOnDisk) still exists in ObjectBox too, just no longer referenced by region.demPackage — it is unreachable from the UI, which only navigates to it via that ToOne."

- timestamp: 2026-07-24T13:33:00Z
  checked: app/lib/components/trail/missing_coverage_sheet.dart, app/lib/provider/trail/trail_download_state_provider.dart
  found: "The missing-coverage sheet defaults Vector checked / DEM unchecked per region, but both checkboxes are fully independent with no locking (unlike settings_offline_regions_screen.dart's _buildDemTile, which gates/locks the DEM action until vectorAvailable). trail_download_state_provider.dart's DownloadingTrailIds.download() starts every selected region's downloadVector/downloadDem fire-and-forget in the same regionFutures.addAll(...) list, with no sequencing between them."
  implication: "This is the FIRST call site in the codebase where a region's vector and DEM downloads can genuinely run concurrently for the same region — the Settings screen's own DEM tile is locked out until vector finishes, so the pre-existing sequential-only usage pattern never exercised this race. This explains why Phase 23/24 UAT never caught it and it only surfaced during Phase 26 UAT."

## Resolution

root_cause: "TileRepositoryManager.startVectorDownload and startDemDownload each read their own independent RegionEntity snapshot via a separate _regionById(id) query at function entry, and hold it for the whole download. On successful completion, startVectorDownload performs an EXTRA, LATE region-row write (`region.lastDownloadedVersion = region.version; box.put(region)`) that startDemDownload does not have. Because ObjectBox's generated put() serialization (objectToFB) always writes every field including both vectorPackage.targetId and demPackage.targetId (no partial/dirty update), this late vector-side write — built from a snapshot captured BEFORE the concurrently-running DEM download had linked its own package — overwrites/clobbers the demPackage relation FK that startDemDownload had already correctly persisted moments earlier (DEM's link write is fast/early; vector's archive download is slower, so vector's late write reliably lands after DEM's). Net effect: the DEM .pmtiles file downloads successfully and sits on disk (counted correctly by region_disk_usage_util.dart, which scans the filesystem directly, independent of the ObjectBox relation) and its DownloadedTilePackageEntity row exists with status=downloaded, but region.demPackage.target is severed back to null — so Settings → Offline Regions (which reads region.demPackage.target?.status) shows 'not downloaded'. This is a real concurrency/data-loss bug triggered specifically by the Phase 26 missing-coverage sheet, the first call site that starts a region's vector and DEM downloads concurrently for the same region (Settings screen's own DEM tile is gated/locked until vector finishes, so it never raced before)."
fix:
verification:
files_changed: []
