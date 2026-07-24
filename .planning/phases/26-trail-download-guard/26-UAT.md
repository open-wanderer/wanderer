---
status: testing
phase: 26-trail-download-guard
source: [26-VERIFICATION.md]
started: 2026-07-24T12:44:19Z
updated: 2026-07-24T15:15:00Z
---

## Current Test

number: 4
name: Multi-region parallel download + unified notification (26-05 re-run)
expected: |
  On a multi-region trail, check Vector for two regions and DEM for one, tap Download once. The id-42 notification shows one combined progress bar that never jumps downward as each package completes, reaches 100%, and only then finalizes to success.
awaiting: user response

## Tests

### 1. Fully-covered / updateAvailable trail starts immediately
expected: Tapping download on a trail whose overlapping regions are all downloaded/updateAvailable starts the download instantly with no sheet.
result: pass

### 2. Missing-coverage sheet appearance and dismiss-abort
expected: Sheet lists missing region(s) with name/size and Vector(checked)/DEM(unchecked) checkboxes; dismissing starts nothing.
result: pass

### 3. No-region-gap warning
expected: An info/warning toast appears for a trail outside every configured region, and the trail download still proceeds.
result: pass

### 4. Multi-region parallel download + unified notification (26-05 re-run)
expected: On a multi-region trail, check Vector for two regions and DEM for one, tap Download once. The id-42 notification shows one combined progress bar that never jumps downward as each package completes, reaches 100%, and only then finalizes to success.
result: [pending]
note: "Original issue (bar resets 3x, finishes <100%) root-caused and fixed in 26-05 (monotonic latch + deferred gated showSuccess). Statically re-verified in 26-VERIFICATION.md; awaiting on-device re-confirmation."

### 5. Guard does not re-fire after a just-completed region download (CR-02 live regression check)
expected: Trigger the sheet, download a region's Vector package, wait for it to finish, then re-tap download on the same/overlapping trail without navigating away (keep Settings/Offline Regions mounted) — the guard recognizes the region as now covered and does not re-show it in the sheet.
result: pass

### 6. Unified notification stays on aggregate copy through tile generation (WR-01 live check)
expected: On a multi-region trail with packages selected, the id-42 notification does not flash back to plain trail-name/"Generating..." copy during the tile-generation phase.
result: pass

### 7. Download button never permanently stranded (CR-01 live check)
expected: After any download attempt (including one that errors early), the trail's download button re-enables; it never stays permanently disabled for the app session.
result: pass

### 8. Downloaded DEM package reflects as downloaded in Settings → Offline Regions (26-05 re-run)
expected: Trigger the missing-coverage sheet for a trail with a missing region, check both Vector and DEM, tap Download. After both finish, Settings → Offline Regions shows the DEM package as "downloaded" (matching Vector), with no orphaned/severed relation, and the region's totals are consistent.
result: [pending]
note: "Original issue (DEM downloads but shows not-downloaded in Settings) root-caused to a concurrent-write race and fixed in 26-05 (fresh-row read-modify-write in tile_repository_manager.dart). Statically re-verified in 26-VERIFICATION.md; awaiting on-device re-confirmation."

## Summary

total: 8
passed: 6
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps

- truth: "One id-42 notification shows a single combined, advancing progress bar (not one notification per download) during a multi-region parallel trail+package download"
  status: failed
  reason: "User reported: The download bar resets 3 times to different values during download and finally finishes before 100%. The trail is successfully downloaded."
  severity: major
  test: 4
  root_cause: "updateAggregate() in trail_download_state_provider.dart reads each selected package's live progress via packageStates[region.id]?.vectorProgress/.demProgress from tileRepositoryStatusProvider, falling back to 0.0 on null. But downloadVector()/downloadDem() in tile_repository_provider.dart deliberately clear that same field to null in their finally block the instant each package completes (a documented contract for the Settings/Regions screen, where non-null presence IS the 'currently downloading' signal). updateAggregate() can't distinguish 'not started' from 'just completed' -- both read as null -- so each completed package's contribution snaps from 1.0 back to 0.0, producing one downward jump per completed package (3 in the repro: 2 vector + 1 dem), and the combined percentage can never reach 100% before the trail's own showSuccess() overwrites the notification."
  artifacts:
    - path: "app/lib/provider/trail/trail_download_state_provider.dart"
      issue: "updateAggregate() re-reads tileRepositoryStatusProvider's ephemeral per-region progress fields directly instead of latching completed fractions monotonically"
    - path: "app/lib/provider/region/tile_repository_provider.dart"
      issue: "downloadVector()/downloadDem() clear vectorProgress/demProgress to null on completion -- correct for the Settings/Regions screen's original contract, but incompatible with the new aggregate notification consumer reading the same field"
  missing:
    - "Give DownloadingTrailIds.download() its own completion-latching accumulator per selected package (mirroring how lastTrailFraction already latches monotonically 0->1), fed by each region future's completion rather than re-reading tileRepositoryStatusProvider's ephemeral state inside updateAggregate()"
    - "Do NOT change tile_repository_provider.dart's clear-on-completion behavior -- the Settings/Regions screen correctly depends on it"
    - "Secondary: showSuccess() fires as soon as the trail's own download resolves, not gated on regionFutures -- related to the 'finishes before 100%' symptom, worth addressing alongside the main fix"
  debug_session: ".planning/debug/26-04-progress-bar-resets.md"

- truth: "After selecting DEM for a region in the missing-coverage sheet and it downloads successfully, the region's DEM package reflects as downloaded in Settings → Offline Regions"
  status: failed
  reason: "User reported: Eventhough I select dem data in the sheet and it downloads properly, the dem data shows as \"not downloaded\" in the settings screen. The total at the top includes it. The vector data shows up correctly as \"downloaded\"."
  severity: major
  test: 8
  root_cause: "startVectorDownload and startDemDownload (tile_repository_manager.dart) each fetch their own independent RegionEntity snapshot at function entry and hold it for the whole download. On success, startVectorDownload performs an extra, late region-row write (region.lastDownloadedVersion = region.version; box.put(region)) that startDemDownload does not have. ObjectBox's generated objectToFB always serializes every field on put() (no partial/dirty update), including both vectorPackage.targetId and demPackage.targetId. Because vector's snapshot was captured before DEM's concurrently-running download linked its own package, vector's late write lands after DEM's early link and clobbers demPackage.targetId back to unset. The DEM .pmtiles file and its DownloadedTilePackageEntity row both exist correctly on disk/DB, and region_disk_usage_util.dart (which stats the filesystem directly) still counts the bytes -- but region.demPackage.target is severed, so Settings (region.demPackage.target?.status) reads notDownloaded. This is a genuine concurrency race newly exposed by Phase 26's missing-coverage sheet, the first call site that starts a region's Vector and DEM downloads concurrently for the same region."
  artifacts:
    - path: "app/lib/services/tile_repository_manager.dart"
      issue: "startVectorDownload's late box.put(region) for lastDownloadedVersion uses a stale in-memory RegionEntity snapshot that clobbers a concurrently-set demPackage relation FK written by startDemDownload running in parallel for the same region"
  missing:
    - "Before any region-row put() in startVectorDownload/startDemDownload, re-fetch the current row from the box (or re-read just the FK fields) so the write only carries the caller's own intended change instead of a stale full snapshot -- e.g. re-query region immediately before setting lastDownloadedVersion/putting"
    - "Preserve the existing symmetry/independence contracts between vector and DEM (RegionEntity.status doc comments, D-06/D-07) -- do not make the two downloads block/serialize on each other, only fix the stale-snapshot overwrite"
  debug_session: ".planning/debug/26-04-dem-status-not-reflected.md"
