---
phase: 26-trail-download-guard
verified: 2026-07-24T12:09:59Z
status: gaps_found
score: 7/8 must-haves verified
overrides_applied: 0
gaps:
  - truth: "A region downloaded via the guard's missing-coverage sheet is correctly recognized as covered on a subsequent trail-download coverage check within the same app session (the guard must not re-fire for a region the user just fixed)"
    status: failed
    reason: "trail_download_state_provider.dart's download() never calls ref.invalidate(regionListNotifierProvider) after the guard's own downloadVector/downloadDem calls complete, violating the explicit invalidation contract documented in region_provider.dart's RegionListNotifier doc comment ('whose callers must ref.invalidate(regionListNotifierProvider) after each mutation'). The only other caller of downloadVector/downloadDem, settings_offline_regions_screen.dart, invalidates after every mutation (lines 661, 675); the guard's download() does not, anywhere in the file. Confirmed both independently (grep for 'invalidate(regionListNotifierProvider)' in the file returns zero matches) and by the code review's CR-02 finding. Because RegionEntity.status reads a cached ObjectBox ToOne.target, any RegionListNotifier instance still alive when the guard's region download finishes keeps reporting that region as not-yet-covered, so a later download() call for the same or another trail overlapping that region can needlessly re-show the missing-coverage sheet for a region the user just downloaded through this exact flow — directly undermining the phase goal's 'makes it easy to fix when it isn't' and the spirit of GUARD-04's 'never re-fires' guarantee."
    artifacts:
      - path: "app/lib/provider/trail/trail_download_state_provider.dart"
        issue: "No ref.invalidate(regionListNotifierProvider) call anywhere in the file after Future.wait(regionFutures) settles"
    missing:
      - "Add ref.invalidate(regionListNotifierProvider); after await Future.wait(regionFutures); (inside the existing isolating try/catch's finally, or immediately after it), matching settings_offline_regions_screen.dart's established pattern"
---

# Phase 26: Trail Download Guard Verification Report

**Phase Goal:** Before a trail downloads, the app makes sure its area is actually covered by a downloaded region, and makes it easy to fix when it isn't.
**Verified:** 2026-07-24T12:09:59Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | GUARD-01 — Tapping download on a fully-covered trail (all overlapping regions downloaded/updateAvailable) starts the download immediately, unchanged from today | ✓ VERIFIED | `trail_download_state_provider.dart:36-74` — `overlapping.isNotEmpty && missing.isEmpty` falls straight through to the pre-existing download body with no sheet, no extra toast. `trail_coverage_util.dart`'s `missingCoverageRegions` is test-guarded (11/11 `flutter test` pass) |
| 2 | GUARD-02 — When coverage is missing, a dialog names the specific missing region(s) + size, with a direct in-dialog per-region action — never a silent block or generic message | ✓ VERIFIED | `missing_coverage_sheet.dart` renders one bordered card per missing region with `region.name`, `formatBytes(vectorSize)`/`formatBytes(demSize)`, and per-region Vector/DEM checkboxes. Implemented as a checkbox-per-region + single Download button rather than a literal per-region "Download region" button — this is a deliberate, human-approved design refinement documented in `26-CONTEXT.md` (D-05–D-08) and `26-DISCUSSION-LOG.md`, made during phase planning, not an unauthorized deviation. Intent (name + size + per-region control, no silent block) is preserved |
| 3 | GUARD-03 — A trail spanning multiple regions lists every missing region with individual + combined size, lets the user download any subset, never forces full coverage | ✓ VERIFIED | Sheet loops over all `missingRegions`; combined-size `Text` recomputes on every checkbox toggle (`combinedBytes`/`selectedCount` in `build()`); `FilledButton onPressed` is never gated on selection (no `isNotEmpty` guard found); 0-checked Download still proceeds (trail-only) |
| 4 | GUARD-04 — `updateAvailable` regions satisfy coverage identically to `downloaded` (the missing-region filter itself) | ✓ VERIFIED | `missingCoverageRegions` excludes both `RegionStatus.downloaded` and `RegionStatus.updateAvailable`; dedicated test `updateAvailable region is EXCLUDED from missingCoverageRegions (GUARD-04)` passes |
| 5 | A region downloaded via the guard is correctly reflected as covered on a later coverage check in the same session (guard never re-fires for a region the user just fixed) | ✗ FAILED | `trail_download_state_provider.dart` never calls `ref.invalidate(regionListNotifierProvider)` after `downloadVector`/`downloadDem` complete (zero matches for the invalidation call in the file), unlike `settings_offline_regions_screen.dart`'s established pattern for the same mutation methods. See gap detail above (CR-02) |
| 6 | D-04 — A trail bbox overlapping NO catalog region shows a non-blocking warning and the trail still downloads | ✓ VERIFIED | `overlapping.isEmpty` branch adds an info toast ("Part of this trail isn't covered by any offered region.") then falls through unchanged, no sheet |
| 7 | D-08/D-09 — One Download tap starts the trail + every selected Vector/DEM package in parallel, fire-and-forget; dismissing the sheet aborts everything (nothing starts) | ✓ VERIFIED | `downloadVector`/`downloadDem` futures collected into `regionFutures` without awaiting before the trail download starts; dismissed sheet (`selection == null`) removes `trail.id` from state and `return`s before any provider/service read |
| 8 | D-10 — One unified id-42 notification aggregates trail + selected packages when 1+ package selected; 0-region path is unchanged | ✓ VERIFIED (minor gap) | `updateAggregate()` calls `showAggregateProgress('Downloading offline content', ...)`, gated on `hasSelectedPackages`; 0-region path retains unchanged `showProgress(trail.name, ...)`. Minor: `onGeneratingChanged` always calls `showGenerating(trail.name)` regardless of `hasSelectedPackages` (WR-01), briefly reverting the notification content during the tile-generation phase when packages are selected — cosmetic, not functional |

**Score:** 7/8 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/trail_coverage_util.dart` | Pure `bboxesOverlap`/`overlappingRegions`/`missingCoverageRegions` | ✓ VERIFIED | All 3 functions present; no Riverpod/ObjectBox/network dependency (grep confirms); `flutter analyze` clean |
| `app/test/util/trail_coverage_util_test.dart` | Table-driven tests | ✓ VERIFIED (info gap) | 11/11 `flutter test` pass covering overlap, degenerate bbox, fully-covered, no-region-gap, updateAvailable-exclusion. No explicit case for `downloading`/`error` statuses (IN-03, info-level, self-noted in SUMMARY) |
| `app/lib/components/trail/missing_coverage_sheet.dart` | `MissingCoverageSelection` + `showMissingCoverageSheet` + content widget | ✓ VERIFIED | Matches plan contract; `flutter analyze` clean; imported and used by the guard |
| `app/lib/services/download_notification_service.dart` | `showAggregateProgress` alongside unchanged `showProgress` | ✓ VERIFIED | Sibling method added; `showProgress(String trailName, int done, int total)` signature untouched; both consumed correctly by the guard |
| `app/lib/provider/trail/trail_download_state_provider.dart` | Coverage guard + parallel-download orchestration + unified notification | ⚠️ VERIFIED WITH BLOCKER | Guard branching, sheet trigger, parallel downloads, and aggregate notification are all present and functionally wired for a single coverage check. Missing `ref.invalidate(regionListNotifierProvider)` after region downloads (CR-02, gap above). Additionally carries a pre-existing (not phase-26-introduced) robustness bug: `showProgress(trail.name, 0, 0)` for the 0-region path executes *outside* the `try/finally` that clears `DownloadingTrailIds.state` (CR-01) — confirmed present in the pre-phase-26 version of this file via `git show c0e5d6eb`, so it is not a regression caused by this phase, but this phase's guard/sheet/region-download-start code now also executes in that same unprotected window ahead of the `try`, widening the exposure |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `trail_coverage_util_test.dart` | `trail_coverage_util.dart` | import + direct calls | WIRED | 11/11 tests pass |
| `missing_coverage_sheet.dart` | `byte_format_util.dart` | `formatBytes()` | WIRED | Used for every size string in the sheet |
| `missing_coverage_sheet.dart` | `tile_repository_provider.dart` | `ref.watch(tileRepositoryStatusProvider)` | WIRED | Line 107; drives live in-flight/error row state via `resolveVectorTileStatus`/`resolveDemTileStatus` |
| `trail_download_state_provider.dart` | `region_provider.dart` | `ref.read(regionListNotifierProvider)` | ⚠️ PARTIAL | Read is wired and correctly local-only (no `refreshCatalog`), but the file never invalidates the same provider after the guard's own region mutations — violates the documented contract (see gap) |
| `trail_download_state_provider.dart` | `missing_coverage_sheet.dart` | `showMissingCoverageSheet` via `navigatorKey.currentContext` | WIRED | Confirmed; `ctx == null` falls through trail-only rather than stranding the user |
| `trail_download_state_provider.dart` | `tile_repository_provider.dart` | `downloadVector`/`downloadDem` | WIRED | Called per selected region, fire-and-forget, futures collected for later isolated `Future.wait` |
| `trail_download_state_provider.dart` | `download_notification_service.dart` | `showAggregateProgress` | WIRED | Called from `updateAggregate()`, gated on `hasSelectedPackages`, driven by both `onProgress` and a `ref.container.listen` subscription (closed in the terminal path) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `missing_coverage_sheet.dart` | `region.vectorSize`/`region.demSize` | `RegionEntity` (ObjectBox-persisted catalog fields) | Yes | ✓ FLOWING |
| `missing_coverage_sheet.dart` | `missingRegions` param | `trail_coverage_util.missingCoverageRegions(trail, regions)` computed from the real `regionListNotifierProvider` ObjectBox snapshot | Yes | ✓ FLOWING |
| `trail_download_state_provider.dart` | `updateAggregate()`'s `packageStates` | `ref.read(tileRepositoryStatusProvider)` live download-progress map | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Coverage util correctness (`bboxesOverlap`/`overlappingRegions`/`missingCoverageRegions`) | `cd app && flutter test test/util/trail_coverage_util_test.dart` | `00:00 +11: All tests passed!` | ✓ PASS |
| Static analysis of all phase-26 modified/created files | `cd app && flutter analyze lib/util/trail_coverage_util.dart lib/components/trail/missing_coverage_sheet.dart lib/services/download_notification_service.dart lib/provider/trail/trail_download_state_provider.dart` | `No issues found! (ran in 5.5s)` | ✓ PASS |
| On-device sheet/guard end-to-end interaction (fully-covered/no-sheet, missing-coverage sheet appearance, dismiss-aborts, multi-region parallel download, unified notification) | n/a — requires a physical device/emulator | not run in this environment | ? SKIP — routed to Human Verification below, per this phase's own `human_verify_mode: end-of-phase` |

### Probe Execution

SKIPPED — no `scripts/*/tests/probe-*.sh` probes exist or are declared for this Flutter mobile phase; the plans' own `<verify>` blocks specify `flutter test`/`flutter analyze` (run above) plus deferred on-device human checks.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| GUARD-01 | 26-01, 26-03 | bbox coverage check vs downloaded/updateAvailable before proceeding | ✓ SATISFIED | `trail_coverage_util.dart` + guard branch in `download()` |
| GUARD-02 | 26-02, 26-03 | Dialog names missing region(s) + size, with in-dialog per-region action | ✓ SATISFIED | `missing_coverage_sheet.dart` per-region cards + checkboxes (design refined from literal "button" to checkbox-selection per documented CONTEXT.md decision) |
| GUARD-03 | 26-02, 26-03 | Multi-region: individual + combined size, subset download, never forces full coverage | ✓ SATISFIED | Combined-size summary + always-enabled Download button |
| GUARD-04 | 26-01, 26-03 | `updateAvailable` satisfies coverage identically to `downloaded` | ⚠️ SATISFIED WITH ADJACENT REGRESSION | `missingCoverageRegions` filter itself is correct and tested; but the guard's own missing `regionListNotifierProvider` invalidation (CR-02, gap above) undermines the broader "never re-fires for a region already fixed" guarantee in a closely related scenario |

No orphaned requirements — all four IDs mapped to Phase 26 in `.planning/REQUIREMENTS.md` (lines 133-136) are claimed by the plans' `requirements:` frontmatter (26-01: GUARD-01/04; 26-02: GUARD-02/03; 26-03: GUARD-01/02/03/04).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `trail_download_state_provider.dart` | whole guard+download body (95-198) | Missing `ref.invalidate(regionListNotifierProvider)` after the guard's own region-download mutations, contradicting the documented contract at `region_provider.dart:161-163` | 🛑 Blocker | Guard can re-show a region as missing after the user just downloaded it through this exact flow (CR-02) |
| `trail_download_state_provider.dart` | 146-166 | `showProgress(trail.name, 0, 0)` (0-region path) executes outside the `try/finally` that clears `DownloadingTrailIds.state` | ⚠️ Warning (pre-existing, exposure widened by this phase) | An exception there permanently strands the trail id in "downloading" (provider is `keepAlive: true`) until app restart (CR-01) — confirmed present before phase 26 via `git show c0e5d6eb`, but this phase added the sheet-await, region-download-start, and notification-setup code that now also runs in the same unprotected window |
| `trail_download_state_provider.dart` | 155-156 | `onGeneratingChanged` always calls single-source `showGenerating(trail.name)`, ignoring `hasSelectedPackages` | ⚠️ Warning | Aggregate notification content briefly reverts to plain trail-name copy during the tile-generation phase (WR-01) |
| `trail_download_state_provider.dart` | 127-133, 156, 163 | Fire-and-forget notification calls, no error handling | ⚠️ Warning | Plugin exceptions become unhandled async errors, indistinguishable from a real download failure (WR-02) |
| `download_notification_service.dart` | 6 | Single shared notification id (42) for all trails | ⚠️ Warning | Concurrent downloads of two different trails would interleave/clobber one notification (WR-03) — pre-existing single-trail assumption, not newly introduced by this phase |
| `missing_coverage_sheet.dart` | 133, 202, 302 | Inconsistent null-fallback default for `_vectorChecked` (`?? false` vs `?? true`) | ℹ️ Info | Currently harmless (map always pre-populated for every listed region) but a latent inconsistency (WR-04) |
| `missing_coverage_sheet.dart` | 144-186, 283-328 | Hardcoded English strings mixed with `l10n.*` calls in the same widget | ℹ️ Info | i18n inconsistency vs. the `track_save_options_sheet.dart` precedent this sheet is modeled on (IN-01) |
| `trail_coverage_util_test.dart` | n/a | No explicit `downloading`/`error` status test case for `missingCoverageRegions` | ℹ️ Info | Filter logic is a simple binary exclusion (only `downloaded`/`updateAvailable`), so risk is low, but coverage is incomplete (IN-03) |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` debt markers found in any of the 4 phase-26 modified/created source files.

### Human Verification Required

The following checks were explicitly deferred to end-of-phase on-device UAT by the plans themselves (`human_verify_mode: end-of-phase`) and could not be run in this text-only environment. They do not change the `gaps_found` status (CR-02 is independently confirmed via static code inspection) but should be completed before considering the phase UX-complete.

### 1. Fully-covered / updateAvailable trail starts immediately

**Test:** Tap download on a trail whose overlapping regions are all `downloaded`, and separately on one where an overlapping region is `updateAvailable`.
**Expected:** Download starts immediately in both cases, no sheet appears.
**Why human:** Requires a device/emulator with a populated region catalog and real download state; not derivable from static analysis alone.

### 2. Missing-coverage sheet appearance and dismiss-abort

**Test:** Tap download on a trail overlapping a `notDownloaded` region; observe the sheet, then swipe it down to dismiss.
**Expected:** Sheet lists the missing region(s) with name/size and Vector (checked)/DEM (unchecked, only if `demUrl != null`) checkboxes; dismissing starts nothing (no trail download, no region download, no notification).
**Why human:** Visual layout, checkbox defaults, and gesture-driven dismiss behavior require on-device interaction.

### 3. No-region-gap warning

**Test:** Tap download on a trail entirely outside every configured region.
**Expected:** An info/warning toast appears ("Part of this trail isn't covered by any offered region.") and the trail download still proceeds.
**Why human:** Toast rendering and timing are visual/runtime concerns.

### 4. Multi-region parallel download + unified notification

**Test:** On a multi-region trail, check Vector for two regions and DEM for one, tap Download once.
**Expected:** The trail and all three packages download in parallel; a single system notification (id 42) shows "Downloading offline content" with one combined, advancing progress bar (not one notification per download); Settings → Offline Maps shows the same regions downloading.
**Why human:** Requires observing real OS notification behavior and concurrent download progress over time.

### 5. Guard does not re-fire after a just-completed region download (regression check for CR-02)

**Test:** Trigger the missing-coverage sheet for a trail, check a region's Vector box, tap Download, wait for that region's download to finish, then immediately re-tap download on the same (or another overlapping) trail without navigating away from the current screen.
**Expected:** The guard should recognize the region as now `downloaded` and either start immediately (if fully covered) or omit that region from the sheet.
**Why human:** Confirms in a live session whether the missing `ref.invalidate(regionListNotifierProvider)` (CR-02) actually manifests as a user-visible re-fire, since the practical impact depends on whether `RegionListNotifierProvider` has a live watcher (e.g., Settings screen still mounted) at the time — static analysis confirms the missing invalidation call exists as a code-level contract violation, but on-device reproduction gives the ground-truth user impact.

### Gaps Summary

One blocking gap: `DownloadingTrailIds.download()` triggers region downloads (`downloadVector`/`downloadDem`) via the guard's missing-coverage sheet but never calls `ref.invalidate(regionListNotifierProvider)` afterward, contradicting the explicit invalidation contract already established and followed everywhere else in the codebase (`region_provider.dart`'s doc comment, obeyed by `settings_offline_regions_screen.dart` on every mutation). This is CR-02 from the phase's own code review, independently reproduced here by direct source inspection (zero occurrences of the invalidation call in the file). Practical impact: because `RegionEntity.status` reads a cached ObjectBox `ToOne.target`, a `RegionListNotifier` instance that stays alive after the guard's region download completes (e.g., because Settings/Offline Regions is mounted elsewhere) will keep reporting that region as not-yet-covered, so the guard can needlessly re-show the missing-coverage sheet for a region the user just fixed through this exact flow — directly undermining the phase goal's second clause, "makes it easy to fix when it isn't," and the spirit (though not the literal `updateAvailable` wording) of GUARD-04's "never re-fires" guarantee.

A second, non-blocking finding (CR-01) is also present: the 0-region path's `showProgress(trail.name, 0, 0)` call executes outside the `try/finally` that clears `DownloadingTrailIds.state`, so a plugin exception there permanently strands a trail in the "downloading" set for the whole app session (the provider is `keepAlive: true`). This bug predates Phase 26 (confirmed via `git show c0e5d6eb`, the pre-phase version of the file has the identical unprotected `await notificationService.showProgress(trail.name, 0, 0);` before the `try`), so it is not a regression this phase introduced — but this phase added meaningfully more code (coverage guard, sheet await, region-download starts, notification-aggregation setup) that now also executes ahead of the same unprotected `try`, widening the window in which an unrelated exception could trigger this pre-existing stranding bug. It does not block this phase's own goal achievement but is flagged per the code review and should be fixed alongside CR-02.

All four ROADMAP success criteria (GUARD-01 through GUARD-04) are otherwise faithfully implemented and match their literal or (for GUARD-02/03, per an explicit human-approved CONTEXT.md design refinement from a per-region button to a checkbox-plus-single-Download-button sheet) intentionally-adapted wording. Both existing trail-download call sites (`trail_detail_screen.dart`, `trail_dropdown.dart`) inherit the guard automatically through the shared `DownloadingTrailIds.download()` entry point. `flutter test`/`flutter analyze` are clean.

---

*Verified: 2026-07-24T12:09:59Z*
*Verifier: Claude (gsd-verifier)*
