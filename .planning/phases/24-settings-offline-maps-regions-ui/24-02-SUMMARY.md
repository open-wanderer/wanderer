---
phase: 24-settings-offline-maps-regions-ui
plan: 02
subsystem: mobile-offline-maps
tags: [flutter, riverpod, go_router, material3, dart]

# Dependency graph
requires:
  - phase: 24-settings-offline-maps-regions-ui (plan 01)
    provides: regionListNotifierProvider, TileRepositoryManager.deleteDemPackage/TileRepositoryStatus.deleteDemPackage, formatBytes, regionDiskUsageBytes/totalRegionDiskUsageBytes, 18 Phase 24 l10n keys
  - phase: 22-region-package-data-model
    provides: RegionEntity, CatalogStatus/RegionStatus/PackageStatus enums
  - phase: 23-tilerepositorymanager-download-engine
    provides: TileRepositoryStatus (downloadVector/downloadDem/pause/resume/delete), RegionDownloadState
provides:
  - "SettingsOfflineRegionsScreen — the Offline Maps/Regions Settings screen (SETUI-01..06)"
  - "/settings/regions GoRoute, reachable from a new Settings entry ListTile"
affects: [phase-25-map-rendering, phase-26-trail-download-guard]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fresh-install-only full-screen error: AsyncLoader/WandererError is reserved for the case where the synchronous provider snapshot is empty AND the fire-and-forget catalog refresh failed; every other failure with cached data surfaces a toast only (RESEARCH.md Pitfall 4)"
    - "_save wrapper invalidates the read-once ObjectBox snapshot provider in a `finally` block after every mutation attempt (success or failure), not just on success, to guarantee the row always reflects the true post-action state (ObjectBox ToOne.target caching, Pitfall 2)"

key-files:
  created:
    - app/lib/routes/settings_offline_regions_screen.dart
  modified:
    - app/lib/routes/settings_screen.dart
    - app/lib/provider/router_provider.dart
    - app/test/routes/settings_screen_test.dart

key-decisions:
  - "Disk-usage summary FutureBuilder future is recreated only when the region-list snapshot's object identity changes (tracked via a stored field + `identical()` check), not on every widget rebuild — avoids re-stat'ing the filesystem on every keystroke in the search box"
  - "Region-count for the disk-usage sub-text counts a region once it has at least one package (vector or DEM) in downloaded/downloading/paused state — partial `.part` files count toward 'used', matching D-06's real-on-disk-bytes intent"
  - "DEM toggle value derives directly from `region.demPackage.target?.status == PackageStatus.downloaded` (no separate ephemeral toggle state); a small inline CircularProgressIndicator is shown next to the switch while a DEM download's ephemeral progress state exists, so the user gets feedback during the toggle-on -> in-flight -> settled window"
  - "Vector/DEM size-breakdown row copy ('120 MB vector · 45 MB DEM') is a hardcoded English literal, not routed through l10n — Plan 01's 18-key l10n allocation for this phase deliberately did not include size-breakdown unit labels, matching the UI-SPEC's own example copy verbatim"

patterns-established:
  - "Full-region delete (D-02) always confirms via a 2-action AlertDialog (Cancel/Delete); DEM toggle-off (D-01) never does — same screen, deliberately asymmetric destructive-action UX for a lower-cost, instantly-reversible action vs. a genuinely costly one"

requirements-completed: [SETUI-01, SETUI-02, SETUI-03, SETUI-04, SETUI-05, SETUI-06]

# Metrics
duration: ~15min
completed: 2026-07-22
---

# Phase 24 Plan 02: Settings — Offline Maps/Regions UI Summary

**SettingsOfflineRegionsScreen — a searchable A-Z region list with a live disk-usage summary, six-state download rows (download/pause/resume/delete/retry/update), an independent DEM toggle, and a Settings entry wired via `/settings/regions` — the single user-facing deliverable of the v1.6 milestone's UI phase.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-22T11:44:00Z (approx)
- **Completed:** 2026-07-22T11:57:00Z (approx)
- **Tasks:** 3 completed (+ 1 Rule-1 test fix)
- **Files modified:** 4 (1 new screen, 2 modified route/navigation files, 1 test fixed)

## Accomplishments
- `SettingsOfflineRegionsScreen`: flat, name-searchable, A-Z region list rendered unconditionally from `regionListNotifierProvider`'s synchronous snapshot, never blocked on the fire-and-forget `refreshCatalog()` network call (RESEARCH.md Pitfall 4)
- Disk-usage summary card (SETUI-05, D-06): Display-role headline byte figure + Label-role "N downloaded region(s)" sub-text, sourced from `totalRegionDiskUsageBytes` (real on-disk bytes, including `.part` partials)
- `catalogStatus` precedence gate (D-09, RESEARCH Pattern 2): `building`/`error` catalog regions render disabled with a caption and zero download affordance, checked strictly before any `RegionStatus` rendering
- Six distinct `RegionStatus` row states (D-04): notDownloaded (Download), downloading (single combined progress bar + Pause, D-07), paused (Resume), downloaded (green check + Delete), updateAvailable (persistent "Update available" banner + Update button, still behaves as downloaded, D-05), error (red badge + Retry-only, D-03)
- Independent per-region DEM `Switch`, rendered only when `region.demUrl != null` (RESEARCH Pattern 3): toggle-on downloads, toggle-off deletes immediately with no dialog (D-01), asymmetric with full-region delete's confirm dialog (D-02)
- New Settings entry `ListTile` + nested `GoRoute('regions')` under `/settings`, reachable end-to-end

## Task Commits

Each task was committed atomically:

1. **Task 1: OfflineRegionsScreen — scaffold, search, disk-usage summary, list, catalog refresh, empty states** - `53fe40c2` (feat)
2. **Task 2: Active region row — 6-state actions, combined progress, DEM toggle, delete/retry/update** - `90b658ab` (feat)
3. **Task 3: Settings entry ListTile + /settings/regions GoRoute** - `d0a98970` (feat)
4. **[Rule 1] Fix settings_screen_test.dart's stale 6-row expectation** - `7c957471` (fix)

## Files Created/Modified
- `app/lib/routes/settings_offline_regions_screen.dart` (NEW) - the full Offline Maps/Regions screen: scaffold, search, disk-usage summary, empty states, catalog-status gate, six-state active row, DEM toggle, delete/retry/update actions
- `app/lib/routes/settings_screen.dart` - new `ListTile` (`FontAwesomeIcons.map`, size 18) pushing `/settings/regions`, placed alongside the map-adjacent Categories tile
- `app/lib/provider/router_provider.dart` - new nested `GoRoute(path: 'regions', ...)` under `/settings`, building `SettingsOfflineRegionsScreen`
- `app/test/routes/settings_screen_test.dart` - updated row-count expectation from 6 to 7 and added the new tile's label assertion

## Decisions Made
- **Disk-usage FutureBuilder future recreated only on region-list identity change:** stored the last-seen `regions` list reference and its computed future as state fields, comparing via `identical()` in `build()` — satisfies the plan's explicit instruction to recompute after mutations (post-`ref.invalidate`) without re-stat'ing the filesystem on unrelated rebuilds (e.g. search-box typing).
- **Region-count definition for the disk-usage sub-text:** a region counts once it has at least one package (vector or DEM) whose status is `downloaded`, `downloading`, or `paused` — i.e. any package genuinely occupying disk space right now, matching D-06's "partial files genuinely occupy disk space" framing rather than only counting fully-`downloaded` regions.
- **DEM toggle has no separate ephemeral on/off state:** its displayed value is derived purely from `region.demPackage.target?.status == PackageStatus.downloaded` per the plan's literal instruction. Added a small inline `CircularProgressIndicator` next to the switch while `TileRepositoryStatus`'s ephemeral `demProgress` is non-null, so the user isn't left wondering why the switch didn't visibly change immediately after tapping it — a Rule 2 addition (missing UX feedback for an async action already in flight), not a change to the value-derivation contract itself.
- **Size-breakdown row text ("120 MB vector · 45 MB DEM") is a hardcoded English literal:** Plan 01's 18-key l10n allocation for this phase (confirmed via `app_en.arb` and 24-01-SUMMARY.md) deliberately did not include unit-label keys for this line; the UI-SPEC's own copy example shows the same unlocalized "vector"/"DEM" abbreviations. No l10n key existed to route through, so this matches the phase's actual scope rather than introducing an undocumented l10n gap.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated `settings_screen_test.dart`'s stale row-count assertion**
- **Found during:** Post-Task-3 overall verification (`flutter test`)
- **Issue:** Task 3 intentionally adds a 7th `ListTile` ("Offline Maps/Regions") to `settings_screen.dart`. The pre-existing test hardcoded `expect(find.byType(ListTile), findsNWidgets(6))` and a fixed list of six English labels, both written before this phase existed — a direct, expected consequence of Task 3's own change, not a pre-existing failure.
- **Fix:** Updated the test's expectation to 7 `ListTile`s, added an assertion for the new "Offline Maps/Regions" label, and renamed the test description to reference SETUI-01.
- **Files modified:** `app/test/routes/settings_screen_test.dart`
- **Verification:** `flutter test test/routes/settings_screen_test.dart` passes; full `flutter test` shows no new failures beyond the confirmed-pre-existing `settings_tab_test.dart` failures (see Issues Encountered)
- **Committed in:** `7c957471`

---

**Total deviations:** 1 auto-fixed (1 bug fix — stale test assertion, direct consequence of this plan's own intentional UI change, no scope creep)
**Impact on plan:** Necessary correction so the plan's own change doesn't leave a red test behind; no behavior change beyond what Task 3 already specified.

## Issues Encountered
- `flutter test` surfaced 4 failing tests in `app/test/components/route_planner/settings_tab_test.dart` (route-planner Settings tab, unrelated feature). Confirmed pre-existing via `git diff` against the commit immediately preceding this plan's first task — zero changes to any file under `app/lib/components/route_planner/` or `app/test/components/route_planner/` across all of 24-01/24-02. Out of this plan's `files_modified` scope; logged in `.planning/phases/24-settings-offline-maps-regions-ui/deferred-items.md` per the SCOPE BOUNDARY rule, not fixed.
- The plan's Task 2 `<human-check>` (on-device physical verification of download/pause/resume/delete/DEM-toggle/offline-resilience) is deferred to end-of-phase per this project's `human_verify_mode: end-of-phase` config — not performed by this automated execution pass.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 24 (Settings — Offline Maps/Regions UI) is functionally complete: `flutter analyze` is clean on all 3 plan-scoped files, `flutter test` introduces zero new failures, and every SETUI-01..06 requirement's automated verification criteria pass.
- End-of-phase on-device human verification (the six-point checklist in 24-02-PLAN.md's Task 2 `<human-check>`) is still outstanding before the phase can be considered fully signed off — this is expected under `human_verify_mode: end-of-phase`, not a blocker for advancing planning to Phase 25.
- Phase 25 (Map Rendering — Region-Based Viewport Pipeline) can proceed; nothing in this plan touches map rendering, and no new blockers were introduced.

---
*Phase: 24-settings-offline-maps-regions-ui*
*Completed: 2026-07-22*

## Self-Check: PASSED

All 6 created/modified files confirmed present on disk; all 4 task/fix commit hashes (`53fe40c2`, `90b658ab`, `d0a98970`, `7c957471`) confirmed present in `git log`.
