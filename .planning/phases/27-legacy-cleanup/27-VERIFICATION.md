---
phase: 27-legacy-cleanup
verified: 2026-07-24T18:10:00Z
status: passed
score: 13/13 must-haves verified
overrides_applied: 0
---

# Phase 27: Legacy Cleanup Verification Report

**Phase Goal:** The old trail-scoped tile system is gone and any files it left behind are cleaned up, so the region system is the only tile path left and its disk-usage figure is trustworthy. (Success criterion #2 — the one-time orphaned-file sweep — is explicitly descoped per 27-CONTEXT.md D-05, a locked user decision. Only success criteria #1 and #3 apply to this phase.)
**Verified:** 2026-07-24T18:10:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `trail_download_service.dart`'s tile-download methods (`_downloadMapTiles`, `_fetchCellList`, `_pollUntilReady`) and all generating-state wiring are deleted (ROADMAP SC#1) | ✓ VERIFIED | `grep -n "_downloadMapTiles\|_fetchCellList\|_pollUntilReady\|isGenerating\|handleGeneratingChanged\|tileResult\|onCellPointsDelta\|_pollInterval\|_pollTimeout" app/lib/services/trail_download_service.dart` → zero lines (independently re-run) |
| 2 | `downloadTrail()` has no `onGeneratingChanged` parameter (D-02) | ✓ VERIFIED | Signature at `trail_download_service.dart:34-38` takes only `trail`, `cancelToken`, `onProgress` |
| 3 | `TrailEntity.pmTiles`/`demPmTiles` and `Trail.pmTiles`/`demPmTiles` fields are deleted, generated code regenerated (D-06, ROADMAP SC#1) | ✓ VERIFIED | `grep -n "pmTiles\|demPmTiles" app/lib/entities/trail_entity.dart app/lib/models/trail.dart app/lib/models/trail.freezed.dart app/lib/models/trail.g.dart app/lib/objectbox-model.json app/lib/objectbox.g.dart` → zero lines |
| 4 | `map_cell.dart` and its `MapCellInfoList`/`MapCellInfo`/`MapCellStatusResponse`/`MapCellStatus` symbols no longer exist anywhere in `app/lib` or `app/test` | ✓ VERIFIED | Files confirmed deleted on disk; `grep -rn "map_cell\|MapCellInfo" app/lib app/test --include="*.dart"` → zero lines |
| 5 | `DownloadNotificationService` has no `showGenerating()` method and no code path calls it (D-07) | ✓ VERIFIED | `grep -rn "showGenerating" app/lib --include="*.dart"` → zero lines; `showProgress`/`showAggregateProgress`/`showSuccess`/`showError`/`dismiss`/`_ensureInitialized` all still present |
| 6 | Project-wide zero remaining references to the legacy trail-scoped tile system (ROADMAP SC#1: "app builds and runs with zero remaining references") | ✓ VERIFIED | `grep -rn -E "pmTiles|demPmTiles|showGenerating|onGeneratingChanged|MapCellInfo|map_cell|_downloadMapTiles|_fetchCellList|_pollUntilReady" app/lib app/test` → zero matches (exit 1) |
| 7 | Trail download still persists photos, waypoint photos, and the Valhalla nav-cache with working `onProgress` reporting (D-04) | ✓ VERIFIED | `_downloadTracked`, `_downloadPhotos`, `_downloadWaypointPhotos` still present; Valhalla `/valhalla/navigate` best-effort write present; `totalPoints` now computed up front (`photoTotal * _pointsPerUnit`, line 79) fixing the would-be dead `report()` regression |
| 8 | Phase 26 guard invariants (single `remove(trail.id)`, single `invalidate(regionListNotifierProvider)`, `vectorLatched`/`demLatched` latch, `trailSucceeded`-gated `showSuccess`) untouched (D-03) | ✓ VERIFIED | All five patterns confirmed present and unchanged in `trail_download_state_provider.dart` |
| 9 | The provider's `downloadTrail()` call site passes only `onProgress`, no `onGeneratingChanged` | ✓ VERIFIED | `trail_download_state_provider.dart:206-215` shows `downloadTrail(trail, onProgress: (done, total) {...})` only |
| 10 | Project analyzes clean (no new errors/warnings introduced by this phase) | ✓ VERIFIED | `flutter analyze` re-run independently: 37 info/warning-level issues, all in files untouched by this phase (`map_screen.dart`, `icon_util.dart`, `navigation_stats_provider.dart`, `settings_categories_screen.dart`, `settings_subcategories_screen.dart`) — matches the SUMMARY's claimed pre-existing baseline exactly, zero errors |
| 11 | Full test suite passes with no regression introduced by this phase | ✓ VERIFIED | `flutter test` re-run independently: 4 failures, all in `test/components/route_planner/settings_tab_test.dart` (icon-lookup/state-selection assertions unrelated to tile/download code) — confirmed via `git log` that this test file was last touched by an unrelated `feat(quick-260717-t7q)` commit predating Phase 27, and it contains zero references to `pmTiles`/`demPmTiles`/`map_cell`/`showGenerating`/`onGeneratingChanged`/`trail_download` |
| 12 | A hiker can still download a trail and use it fully offline (basemap + navigation) purely through the region system, no functional regression (ROADMAP SC#3) | ✓ VERIFIED | Region system files (`tile_repository_manager.dart`, rendering pipeline) are not touched by any Phase 27 commit; the 26-04/26-05 automated regression coverage exercising exactly this download/notification code path passes (see #8, #11); this is the acceptance evidence the plan itself specifies |
| 13 | ROADMAP success criterion #2 (orphaned-file sweep) correctly marked descoped, not silently missing | ✓ VERIFIED (by design) | ROADMAP.md Phase 27 section (line 568) explicitly annotates SC#2 as `[DESCOPED for Phase 27 per CONTEXT.md D-05 ...]` — see Documentation Notes below for a related staleness issue that does NOT affect this truth |

**Score:** 13/13 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/services/trail_download_service.dart` | `downloadTrail()` reduced to photo + waypoint-photo + nav-cache download, contains `_downloadTracked` | ✓ VERIFIED | Tile methods gone, `_downloadTracked` present, `totalPoints` fix in place |
| `app/lib/provider/trail/trail_download_state_provider.dart` | Region-download guard wired to `downloadTrail(onProgress:)` only | ✓ VERIFIED | `onProgress:` present, `onGeneratingChanged:` absent |
| `app/lib/services/download_notification_service.dart` | Notification service without the trail-tile generation spinner | ✓ VERIFIED | `showProgress` present, `showGenerating` absent |
| `app/lib/models/map_cell.dart` (+ `.freezed.dart`/`.g.dart`) | Deleted | ✓ VERIFIED | `test -e` confirms all three absent |
| `app/lib/entities/trail_entity.dart` | `TrailEntity` without tile fields or mapping refs | ✓ VERIFIED | Zero `pmTiles`/`demPmTiles` hits |
| `app/lib/models/trail.dart` | `Trail` freezed model without `pmTiles`/`demPmTiles` | ✓ VERIFIED | Zero hits |
| `app/lib/objectbox-model.json` | Regenerated ObjectBox schema without `pmTiles`/`demPmTiles` properties | ✓ VERIFIED | Zero hits; regenerated alongside `objectbox.g.dart`, `trail.freezed.dart`, `trail.g.dart` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `trail_download_state_provider.dart` | `trail_download_service.dart` | `downloadTrail(onProgress:)` call | ✓ WIRED | Call site confirmed at lines 206-215; only `onProgress` argument passed |
| `trail_entity.dart` | `trail.dart` | `TrailEntityMapping` bridge (edited together) | ✓ WIRED | Both files clean of `pmTiles`/`demPmTiles`; bridge methods compile (confirmed via `flutter analyze` — no undefined-field errors) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Zero remaining references to legacy tile system, whole project | `grep -rn -E "pmTiles\|demPmTiles\|showGenerating\|onGeneratingChanged\|MapCellInfo\|map_cell\|_downloadMapTiles\|_fetchCellList\|_pollUntilReady" app/lib app/test` | No matches (exit 1) | ✓ PASS |
| Legacy model files physically deleted | `test -e app/lib/models/map_cell.dart(.freezed/.g).dart` | All three: not found | ✓ PASS |
| Project compiles / analyzes | `cd app && flutter analyze` | 37 info/warning issues, 0 errors, all in files this phase did not touch | ✓ PASS (matches claimed pre-existing baseline) |
| Test suite | `cd app && flutter test` | 357 passed / 4 failed, all 4 failures in `settings_tab_test.dart` (unrelated file, unrelated commit history) | ✓ PASS (matches claimed pre-existing baseline, no regression) |
| Phase 26 guard invariants intact | `grep -n "invalidate(regionListNotifierProvider)\|remove(trail.id)\|vectorLatched\|demLatched\|trailSucceeded" trail_download_state_provider.dart` | All 5 patterns present | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| CLEAN-01 | 27-01, 27-02 | Trail-scoped tile download code removed outright — `trail_download_service.dart` methods, `TrailEntity.pmTiles`/`demPmTiles`, related UI | ✓ SATISFIED | Confirmed above across truths #1-9; REQUIREMENTS.md checklist line 74 correctly marks `[x]` |
| CLEAN-02 | (none — intentionally not claimed by any plan) | One-time on-device cleanup sweep for orphaned legacy tile files | Intentionally descoped (D-05) — not evaluated as a gap per phase scope and per explicit verification-task instruction | REQUIREMENTS.md checklist line 75 correctly shows `[ ]` (unimplemented, as intended); ROADMAP.md Phase 27 section explicitly annotates SC#2 as descoped |

**Orphaned requirements check:** REQUIREMENTS.md maps only CLEAN-01/CLEAN-02 to Phase 27; both accounted for above. No orphans.

### Anti-Patterns Found

None. No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers, no empty-implementation stubs, and no hardcoded-empty-data patterns in any file this phase modified (`trail_download_service.dart`, `trail_download_state_provider.dart`, `download_notification_service.dart`, `trail_entity.dart`, `trail.dart`).

### Documentation Notes (non-blocking)

Two pre-existing documentation staleness issues were found during verification. Neither affects the phase's code-level goal achievement (both are project-tracking artifacts, not the deliverable itself), so they are reported as informational findings, not gaps:

1. **REQUIREMENTS.md traceability table (lines 143-144) is stale.** It still reads `CLEAN-01 | Phase 27 | Pending (27-01 runtime portion done; 27-02 removes pmTiles/demPmTiles fields)` — written after 27-01 but never updated after 27-02 completed. Should now read `Complete`. `CLEAN-02 | Phase 27 | Pending` has no explanation of the D-05 descope decision, unlike the checklist section above it and ROADMAP.md's Phase 27 success-criteria list, both of which do explain it correctly.
2. **ROADMAP.md's milestone summary line (359)** reads `Phase 27: Legacy Cleanup - Trail-scoped tile code deleted outright, orphaned legacy files swept on first launch (completed 2026-07-24)` — this contradicts the same document's own Phase 27 section three sections down (line 568), which correctly marks the sweep as `[DESCOPED for Phase 27 per CONTEXT.md D-05 ...]`. The summary line overstates what shipped.

These were flagged by CONTEXT.md itself as deferred to "a future editing pass" (D-05 downstream note) and are not attributable to either 27-01 or 27-02's `files_modified` scope. Recommend a small housekeeping pass to reconcile REQUIREMENTS.md's traceability table and ROADMAP.md's summary line with the Phase 27 section's own (correct) annotation, but this does not block phase closure.

### Human Verification Required

None. All must-haves are verifiable via grep/analyze/test against the actual codebase, and region-system code (the path a hiker's offline download/use actually exercises) was not modified by this phase — the existing 26-04/26-05 automated regression suite (re-run and confirmed green above) is sufficient evidence for ROADMAP success criterion #3.

### Gaps Summary

No gaps. All applicable ROADMAP success criteria (#1 and #3; #2 explicitly descoped per locked decision D-05) are achieved. Both plans' must-haves are verified against the actual codebase, not just SUMMARY.md claims — every grep in this report was independently re-run rather than trusted from the summaries, and both `flutter analyze` and `flutter test` were independently re-executed with results cross-checked against file/commit history to confirm the claimed pre-existing baseline (37 analyzer infos/warnings, 4 unrelated `settings_tab_test.dart` failures) was not altered by this phase.

---

*Verified: 2026-07-24T18:10:00Z*
*Verifier: Claude (gsd-verifier)*
