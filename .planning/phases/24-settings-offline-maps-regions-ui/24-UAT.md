---
status: diagnosed
phase: 24-settings-offline-maps-regions-ui
source: [24-VERIFICATION.md]
started: 2026-07-22T14:25:00Z
updated: 2026-07-23T00:15:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Re-run 24-UAT.md test 2: on a physical Android device, pick a region that has NEVER been downloaded before, tap 'Download vector', watch it transition to `downloading` (progress bar advances, not `error`). Then pause it, resume it, and delete it via the confirm dialog.
expected: Package transitions to downloading (not error); pause/resume/delete all function as originally specified; the device log line 'freeDiskSpaceBytes gives: Specified path does not exist' no longer appears.
result: issue
reported: "Package does not transition to downloading. It immediately goes into downloaded after clicking the download button."
severity: major

### 2. Re-run 24-UAT.md test 3: toggle DEM on for that same never-downloaded region.
expected: DEM download starts (spinner, in-flight feedback), not error.
result: pass

### 3. Re-run 24-UAT.md test 4 (previously blocked): watch the disk-usage total as packages are added/removed, including while a download is paused mid-transfer.
expected: Total updates after each mutation and includes partial (.part) bytes for a paused/in-progress download.
result: issue
reported: "pass. Paused can not be tested (see test 1)"
severity: minor

### 4. Re-run 24-UAT.md test 5 (previously blocked): with airplane mode on and at least one previously-downloaded region, open the screen.
expected: The previously-downloaded region still appears and is usable; the screen does not blank or show a full-screen error just because the catalog refresh failed offline.
result: pass

## Summary

total: 4
passed: 2
issues: 2
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Package transitions to downloading (not error); pause/resume/delete all function as originally specified; the device log line 'freeDiskSpaceBytes gives: Specified path does not exist' no longer appears."
  status: failed
  reason: "User reported: Package does not transition to downloading. It immediately goes into downloaded after clicking the download button."
  severity: major
  test: 1
  root_cause: "UI-layer staleness bug, not a download-engine bug — the download genuinely succeeds in the background. SettingsOfflineRegionsScreen._buildActiveRow renders progress/status from region.status (RegionEntity.status), which reads region.vectorPackage.target — an ObjectBox ToOne relation that lazily resolves once per Dart object instance and then caches permanently (no TTL/invalidation). The screen's RegionEntity instances come from regionListNotifierProvider and are only refreshed via a single terminal ref.invalidate() in _save()'s finally block, AFTER the whole download resolves — not incrementally as TileRepositoryManager (working on a separately-queried entity) writes downloading -> downloaded internally. So region.status stays frozen at notDownloaded for the entire download; the moment the terminal invalidate fires, a fresh entity appears already at downloaded, producing the observed 'jumps straight to downloaded.' Not a 24-03 regression — pre-existing since 24-02's _buildActiveRow, previously masked because the disk-space bug made downloads fail near-instantly (no visible downloading frame either way)."
  artifacts:
    - path: "app/lib/routes/settings_offline_regions_screen.dart"
      issue: "_save() (513-531) invalidates regionListNotifierProvider too coarsely (once, at the end); _buildActiveRow (309-317, 357, 438) renders from stale region.status instead of the live ephemeral downloadState from tileRepositoryStatusProvider"
    - path: "app/lib/entities/region_entity.dart"
      issue: "status getter (103-123) is logically correct but built on a ToOne relation (vectorPackage.target) that can be permanently frozen on a stale entity instance"
    - path: "app/lib/provider/region/region_provider.dart"
      issue: "RegionListNotifier (157-174) is the only source of fresh (uncached) RegionEntity instances; requires explicit invalidation to refresh — doc comment already documents this ObjectBox ToOne caching pitfall"
  missing:
    - "In _buildActiveRow, prefer the ephemeral downloadState (from tileRepositoryStatusProvider) over region.status whenever a mutation is in flight (downloadState != null) — it's a plain Riverpod state map, immune to ObjectBox ToOne caching. Fall back to region.status only when idle, which is exactly when _save's terminal invalidate guarantees freshness."
  debug_session: ".planning/debug/region-download-stale-toone.md"

- truth: "Total updates after each mutation and includes partial (.part) bytes for a paused/in-progress download."
  status: failed
  reason: "User reported: pass. Paused can not be tested (see test 1) — the paused/in-progress .part-byte accounting could not be exercised because the download never reaches a downloading/paused state (same underlying defect as test 1)."
  severity: minor
  test: 3
  root_cause: "Same root cause as test 1's gap (shared ObjectBox ToOne staleness bug in settings_offline_regions_screen.dart) — fixing test 1's gap (making _buildActiveRow prefer the live ephemeral downloadState) resolves this gap as a side effect, since the paused/in-progress state becomes observable once the UI stops rendering from the frozen region.status."
  artifacts:
    - path: "app/lib/routes/settings_offline_regions_screen.dart"
      issue: "Same as test 1's gap — _buildActiveRow reads stale region.status instead of live downloadState"
  missing:
    - "Same fix as test 1's gap — no separate fix required"
  debug_session: ".planning/debug/region-download-stale-toone.md"
