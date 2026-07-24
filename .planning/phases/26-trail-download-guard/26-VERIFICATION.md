---
phase: 26-trail-download-guard
verified: 2026-07-24T15:10:00Z
status: human_needed
score: 18/18 must-haves verified (statically); 2 require on-device re-confirmation
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 13/13 (automated) with 7 items awaiting on-device UAT
  gaps_closed:
    - "26-UAT.md Test 4 (unified id-42 aggregate progress bar resets downward 3 times and finishes below 100%) — root-caused to updateAggregate()'s `?? 0.0` raw read of tileRepositoryStatusProvider's ephemeral vectorProgress/demProgress, which tile_repository_provider.dart clears to null on completion. Fixed in 26-05 with vectorLatched/demLatched monotonic-max latch maps + whenComplete-driven 1.0 forcing + trailSucceeded-gated deferred showSuccess."
    - "26-UAT.md Test 8 (DEM package downloads successfully but shows 'not downloaded' in Settings -> Offline Regions) — root-caused to startVectorDownload's late lastDownloadedVersion box.put(region) re-serializing a stale RegionEntity snapshot captured before startDemDownload's concurrent early link, clobbering demPackage.targetId back to unset. Fixed in 26-05 with a freshRegion = _regionById(id) re-fetch inside every region-row write transaction in both _getOrCreatePackage and startVectorDownload's late write."
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "26-UAT.md Test 4 re-run: unified progress bar advances monotonically to 100%"
    expected: "On a multi-region trail, check Vector for two regions and DEM for one, tap Download once. The id-42 notification shows one combined progress bar that never jumps downward as each package completes, reaches 100%, and only then finalizes to success."
    why_human: "Requires observing real OS notification content/percentage over time during a live concurrent download; the fix is statically confirmed present and correctly scoped (monotonic latch, whenComplete forcing, deferred gated showSuccess) but its live behavior can only be ground-truthed on-device, per this phase's own human_verify_mode: end-of-phase and the 26-05-PLAN's own <human-check> block."
  - test: "26-UAT.md Test 8 re-run: concurrent Vector+DEM download shows DEM as downloaded"
    expected: "Trigger the missing-coverage sheet for a trail with a missing region, check both Vector and DEM, tap Download. After both finish, Settings -> Offline Regions shows the DEM package as 'downloaded' (matching Vector), with no orphaned/severed relation."
    why_human: "Requires a live ObjectBox read/write race between two concurrently-running downloads on a real device; the fresh-row read-modify-write fix is statically confirmed present and correctly scoped, but its live effect on the actual persisted relation can only be ground-truthed on-device, per this phase's own human_verify_mode: end-of-phase and the 26-05-PLAN's own <human-check> block."
---

# Phase 26: Trail Download Guard Verification Report

**Phase Goal:** Before a trail downloads, the app makes sure its area is actually covered by a downloaded region, and makes it easy to fix when it isn't.
**Verified:** 2026-07-24T15:10:00Z
**Status:** human_needed
**Re-verification:** Yes — after 26-05 gap-closure plan execution (closing 2 UAT-found issues on top of the already-passed 26-04 gap-closure round)

## Goal Achievement

### Re-Verification Summary

The prior verification (2026-07-24T13:05:00Z) found 13/13 must-haves statically verified but 7 items deferred to on-device UAT. The user then ran that UAT (`26-UAT.md`): 6 of 8 tests passed (including all 7 previously-deferred items, since the UAT session covered them plus one additional regression check), but 2 NEW issues surfaced that were not previously known:

1. **Test 4 (major):** the unified id-42 aggregate progress-bar notification resets downward 3 times during a multi-region download and finishes below 100%, even though the trail itself downloads successfully.
2. **Test 8 (major):** a region's DEM package downloads successfully (file + DB row both correct) but Settings → Offline Regions shows it as "not downloaded" — a concurrent-write race, newly exposed because Phase 26's missing-coverage sheet is the first call site in the codebase to start a region's Vector and DEM downloads concurrently.

Both were diagnosed with confirmed root causes (`.planning/debug/26-04-progress-bar-resets.md`, `.planning/debug/26-04-dem-status-not-reflected.md`) and closed by plan `26-05`. **Direct source re-inspection (not SUMMARY narrative) of both modified files confirms both fixes actually landed as described:**

**Gap 1 fix (`app/lib/provider/trail/trail_download_state_provider.dart`):**
- `vectorLatched`/`demLatched` monotonic-max latch maps exist (`grep -c` → 5 each), declared as two separate maps keyed by `region.id` (avoiding a same-region-both-selected key collision), initialized to `0.0`.
- `updateAggregate()`'s per-package summation reads `packageStates[region.id]?.vectorProgress`/`.demProgress` but only ever RAISES the latch (`if (live != null && live > latched[id]!) latched[id] = live;`) before summing `latched[id]!` — confirmed by direct read, not just grep. This means `tile_repository_provider.dart`'s documented clear-to-null-on-completion (confirmed present and unchanged: `_clearVectorProgress`/`_clearDemProgress` grep count 8, unchanged) can no longer snap a finished package's contribution back to 0.
- Each region future is now `tileRepoNotifier.downloadVector(region.id).whenComplete(() { vectorLatched[region.id] = 1.0; updateAggregate(); })` (and the DEM mirror) — `whenComplete` count is 4 (2 vector-population + 2 dem-population closures across the two `regionFutures.addAll` list-comprehension entries), confirming every selected package's own future completion (success OR failure) forces its latch to a full 1.0, guaranteeing the aggregate can reach 100%.
- `showSuccess` is now gated: `grep -c 'showSuccess'` → 2 — one at `if (regionFutures.isEmpty) { await notificationService.showSuccess(trail.name); }` (trail-only path, unchanged timing) and a new deferred call `if (regionFutures.isNotEmpty && trailSucceeded) { await ref.read(downloadNotificationServiceProvider).showSuccess(trail.name).catchError((_) {}); }` positioned AFTER `aggregateSub?.close()`, itself AFTER the region-futures `Future.wait` + `regionListNotifierProvider` invalidation. `trailSucceeded` (grep count 5: declaration, assignment, and the two gate reads) is set immediately after `downloadTrail()` resolves, alongside forcing `lastTrailFraction = 1.0`.
- **26-04 invariants preserved:** `remove(trail.id)` count is still exactly 1 (CR-01 unchanged), `invalidate(regionListNotifierProvider)` count is still exactly 1 in its original `finally` position (CR-02 unchanged), `onGeneratingChanged` still branches on `hasSelectedPackages` calling `updateAggregate()` (WR-01 unchanged), `.catchError((_) {})` hardening is intact on the `showAggregateProgress` call inside `updateAggregate()`.

**Gap 2 fix (`app/lib/services/tile_repository_manager.dart`):**
- `_getOrCreatePackage` signature changed to `_getOrCreatePackage(RegionEntity region, {required bool dem})` (confirmed by direct read of the declaration) — no call site still passes a raw `ToOne` (`grep -c '_getOrCreatePackage(region, region.'` → 0). All 4 call sites (2 vector, 2 DEM) use the new named-arg form (`dem: false` count 3, `dem: true` count 3 — 1 pre-existing `_requestPathFor(id, dem: ...)` call plus 2 converted `_getOrCreatePackage` call sites each, matching the plan's own acceptance-criteria arithmetic exactly).
- Inside `_getOrCreatePackage`, when creating a new package, the write transaction re-fetches the current row (`final freshRegion = _regionById(region.id) ?? region;`) and sets ONLY the caller's own relation (`(dem ? freshRegion.demPackage : freshRegion.vectorPackage).target = created;`) before `put(freshRegion)` — confirmed by direct read.
- `startVectorDownload`'s late `lastDownloadedVersion` write (the confirmed clobber path) no longer uses the stale entry-time `region` snapshot: `grep -c 'region.lastDownloadedVersion = region.version'` → 0 (the old stale-snapshot line is gone), replaced by `_store.runInTransaction(TxMode.write, () { final freshRegion = _regionById(id); if (freshRegion != null) { freshRegion.lastDownloadedVersion = freshRegion.version; _store.box<RegionEntity>().put(freshRegion); } });` — `freshRegion` grep count is 8 (well above the plan's `>= 2` gate), confirming both the `_getOrCreatePackage` link-put and the late vector write both now re-fetch before writing.
- `startDemDownload` is confirmed unchanged apart from its `_getOrCreatePackage(region, dem: true)` call-site signature update — still has NO `lastDownloadedVersion` write, preserving the documented "DEM has no staleness concept" independence contract (D-06/D-07). No cross-reference exists between `startVectorDownload` and `startDemDownload` — the two downloads still run fully concurrently with no new lock/await/serialization.
- `deleteRegion`/`deleteDemPackage` (out of scope per the plan) are confirmed untouched.

Both 26-05 commits (`1a0e1553`, `455d3f79`) are present in `git log` for these exact files, immediately after the 26-04 commits (`5d35ad84`, `12a920ee`), confirming the SUMMARY's claims are backed by real, inspectable commits.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | GUARD-01 — Tapping download on a fully-covered trail starts immediately | ✓ VERIFIED | `overlapping.isNotEmpty && missing.isEmpty` falls straight through, no sheet; unchanged; confirmed passed on-device (26-UAT.md Test 1) |
| 2 | GUARD-02 — Missing coverage shows a dialog naming region(s) + size, in-dialog per-region action | ✓ VERIFIED | `missing_coverage_sheet.dart` at HEAD unchanged since 26-04; confirmed passed on-device (26-UAT.md Test 2) |
| 3 | GUARD-03 — Multi-region trail lists every missing region, subset download, never forces full coverage | ✓ VERIFIED | Unchanged core behavior; combined-size summary recomputes on toggle |
| 4 | GUARD-04 — `updateAvailable` regions satisfy coverage identically to `downloaded` | ✓ VERIFIED | `missingCoverageRegions` filter unchanged; 11/11 dedicated tests pass |
| 5 | A region downloaded via the guard is correctly reflected as covered on a later coverage check in the same session (CR-02) | ✓ VERIFIED | `invalidate(regionListNotifierProvider)` still exactly 1 occurrence, unchanged position; confirmed passed on-device (26-UAT.md Test 5) |
| 6 | D-04 — A trail bbox overlapping no catalog region shows a non-blocking warning, trail still downloads | ✓ VERIFIED | Unchanged; confirmed passed on-device (26-UAT.md Test 3) |
| 7 | D-08/D-09 — One Download tap starts trail + every selected package in parallel; dismiss aborts everything | ✓ VERIFIED | Unchanged abort semantics; confirmed passed on-device (26-UAT.md Test 2) |
| 8 | D-10 — One unified id-42 notification aggregates trail + selected packages; 0-region path unchanged | ✓ VERIFIED (source) — **see truth #14/#15 below for the two 26-05-specific fixes to this same notification** | `onGeneratingChanged` still branches on `hasSelectedPackages`; confirmed passed on-device for notification-content-stability (26-UAT.md Test 6) |
| 9 | CR-01 — An exception between "downloading" and completion always clears `trail.id`; button never permanently stranded | ✓ VERIFIED | Exactly one `remove(trail.id)` in the outer `finally`; confirmed passed on-device (26-UAT.md Test 7) |
| 10 | `trail.id` clears immediately after the trail download settles, not delayed until background region packages finish | ✓ VERIFIED | Outer `try/finally` closes right after the inner `downloadTrail` try/catch; region-futures wait/latch/deferred-success all positioned textually and functionally after it |
| 11 | WR-01 — Unified notification content preserved through tile-generation phase | ✓ VERIFIED | `onGeneratingChanged` calls `updateAggregate()` when `hasSelectedPackages`; confirmed passed on-device (26-UAT.md Test 6) |
| 12 | WR-02 — A notification-plugin exception in a fire-and-forget call is swallowed locally | ✓ VERIFIED | `.catchError((_) {})` present on `showAggregateProgress`, `showGenerating` (false branch), `showProgress` (else branch), and now also the deferred `showSuccess` call |
| 13 | No regression to fully-covered / no-region-gap / dismiss-abort / 0-region download paths | ✓ VERIFIED | `flutter analyze` clean on both 26-05 files; full-app `flutter analyze` shows only the same 37 pre-existing, unrelated issues; `flutter test test/util/trail_coverage_util_test.dart` 11/11 pass |
| 14 | **[26-05 gap 1]** During a multi-region parallel download, the unified id-42 notification's progress percentage never jumps downward when a selected package completes — each package's contribution latches monotonically 0→1 and holds at 1.0 even after `tile_repository_provider.dart` clears its ephemeral field to null | ✓ VERIFIED (static) — **live confirmation deferred, see Human Verification #1** | `vectorLatched`/`demLatched` maps confirmed by direct read to only ever be raised by live progress, never lowered; each package's own future forces its latch to 1.0 via `whenComplete` regardless of the ephemeral field's subsequent null-clear |
| 15 | **[26-05 gap 1 secondary]** The unified notification advances to 100% and finalizes to success only after BOTH the trail download and every selected package have settled | ✓ VERIFIED (static) — **live confirmation deferred, see Human Verification #1** | `showSuccess` now gated on `regionFutures.isEmpty` (immediate, trail-only path) vs. deferred `regionFutures.isNotEmpty && trailSucceeded` call positioned after `Future.wait(regionFutures)` + `aggregateSub?.close()` |
| 16 | **[26-05 gap 2]** After the sheet downloads a region's DEM package concurrently with Vector, Settings → Offline Regions shows the DEM package as downloaded — `region.demPackage.target` survives, no longer clobbered by the vector-side region-row write | ✓ VERIFIED (static) — **live confirmation deferred, see Human Verification #2** | `startVectorDownload`'s late region-row write now re-fetches (`freshRegion = _regionById(id)`) before writing only `lastDownloadedVersion`; the stale-snapshot write (`region.lastDownloadedVersion = region.version`) is confirmed gone (grep count 0) |
| 17 | **[26-05 gap 2 constraint]** Vector and DEM downloads for the same region still run fully concurrently — neither is serialized or blocked on the other | ✓ VERIFIED | No `await`, lock, mutex, or cross-reference added between `startVectorDownload`/`startDemDownload`; confirmed by direct read — only the tiny synchronous DB-write sections re-fetch |
| 18 | **[26-05 gap 1 constraint]** `tile_repository_provider.dart`'s clear-on-completion contract is unchanged — the per-region Settings "downloading" signal still clears `vectorProgress`/`demProgress` to null on completion | ✓ VERIFIED | File not in `26-05`'s `files_modified`; `_clearVectorProgress`/`_clearDemProgress` grep count 8, unchanged; confirmed by direct read of `tile_repository_provider.dart` |

**Score:** 18/18 truths statically verified (13 carried forward unchanged/regression-free + 5 new from `26-05`'s must_haves). 2 of the 18 (#14/#15's parent symptom and #16) still require on-device confirmation that the fix produces the correct user-visible outcome in a live concurrent download, per this phase's own `human_verify_mode: end-of-phase` and `26-05-PLAN.md`'s own `<human-check>` blocks — this is exactly the same escalation pattern the prior verification round used for CR-02/CR-01/WR-01 before their live UAT confirmation.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/trail_coverage_util.dart` | Pure bbox coverage helpers | ✓ VERIFIED | Unchanged; `flutter analyze` clean |
| `app/test/util/trail_coverage_util_test.dart` | Table-driven tests | ✓ VERIFIED | 11/11 `flutter test` pass |
| `app/lib/components/trail/missing_coverage_sheet.dart` | Sheet UI + selection model | ✓ VERIFIED (committed state) | Unchanged since 26-04 at HEAD; **note:** an uncommitted local working-tree diff exists (cosmetic text-style change, `bodyLarge`→`bodyMedium`, and formatting reflow) — not part of any 26-0X plan's `files_modified`, not committed, and outside phase 26 scope; does not affect the verified (committed) phase-26 behavior |
| `app/lib/services/download_notification_service.dart` | `showAggregateProgress`/`showSuccess`/etc. | ✓ VERIFIED | Unchanged, not in 26-05's `files_modified` |
| `app/lib/provider/trail/trail_download_state_provider.dart` | Coverage guard + parallel download + unified notification + crash-safe cleanup + monotonic latch + gated success | ✓ VERIFIED | All CR-01/CR-02/WR-01/WR-02 (26-04) + latch/gated-success (26-05) fixes confirmed present by direct source read; `flutter analyze` clean; commits `5d35ad84`/`12a920ee`/`1a0e1553` all present |
| `app/lib/services/tile_repository_manager.dart` | Region download engine + fresh-row read-modify-write | ✓ VERIFIED | `_getOrCreatePackage`/`startVectorDownload` fresh-row fix confirmed present by direct source read; `flutter analyze` clean; commit `455d3f79` present; `flutter test test/services/tile_repository_manager_test.dart` 16/16 pass |
| `app/lib/provider/region/tile_repository_provider.dart` | Per-region ephemeral progress + clear-on-completion (read-only reference, NOT modified) | ✓ VERIFIED UNCHANGED | Confirmed absent from `26-05`'s `files_modified`; `_clearVectorProgress`/`_clearDemProgress` grep count 8, unchanged from prior verification |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `trail_download_state_provider.dart` (`updateAggregate`) | `vectorLatched`/`demLatched` maps | monotonic-max latch, raised by live progress, forced to 1.0 by `whenComplete` | ✓ WIRED | Confirmed by direct read; replaces the raw `?? 0.0` ephemeral read |
| `trail_download_state_provider.dart` | region futures | `whenComplete` per-package latch-to-1.0 callback | ✓ WIRED | `whenComplete` count 4, confirmed on both vector and DEM population loops |
| `trail_download_state_provider.dart` | `downloadNotificationServiceProvider` (deferred success) | `showSuccess` gated on `regionFutures.isNotEmpty && trailSucceeded`, positioned after `Future.wait(regionFutures)` | ✓ WIRED | Confirmed by direct read; correctly positioned after `aggregateSub?.close()` |
| `tile_repository_manager.dart` (`_getOrCreatePackage`) | `freshRegion = _regionById(region.id)` | fresh-row re-fetch inside `runInTransaction(TxMode.write)` before `put()` | ✓ WIRED | Confirmed by direct read; `freshRegion` count 8 |
| `tile_repository_manager.dart` (`startVectorDownload` late write) | `freshRegion = _regionById(id)` | fresh-row re-fetch before `lastDownloadedVersion` write | ✓ WIRED | Confirmed by direct read; stale-snapshot line (`region.lastDownloadedVersion = region.version`) grep count 0 |
| `trail_download_state_provider.dart` | `region_provider.dart` | `ref.invalidate(regionListNotifierProvider)` (26-04, CR-02) | ✓ WIRED (unchanged) | 1 occurrence, same position as prior verification |
| `trail_download_state_provider.dart` | `DownloadingTrailIds.state` | single outer `try/finally` (26-04, CR-01) | ✓ WIRED (unchanged) | Exactly one `remove(trail.id)` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `trail_download_state_provider.dart` | `vectorLatched[id]`/`demLatched[id]` | `ref.read(tileRepositoryStatusProvider)` live progress, monotonically latched | Yes | ✓ FLOWING — raised by real live progress, never fabricated |
| `trail_download_state_provider.dart` | `trailSucceeded` | set immediately after `await trailDownloadService.downloadTrail(...)` resolves | Yes | ✓ FLOWING — real completion signal, not a static default |
| `tile_repository_manager.dart` | `freshRegion` | `_regionById(id)` — a live ObjectBox query re-executed at write time | Yes | ✓ FLOWING — re-reads the actual current row, not a cached/stale in-memory snapshot |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Coverage util correctness | `cd app && flutter test test/util/trail_coverage_util_test.dart` | `+11: All tests passed!` | ✓ PASS |
| Tile repository manager pure-helper suite (gap 2 regression signal) | `cd app && flutter test test/services/tile_repository_manager_test.dart` | `+16: All tests passed!` (folded into the combined run below) | ✓ PASS |
| Static analysis of both 26-05-modified files | `cd app && flutter analyze lib/provider/trail/trail_download_state_provider.dart lib/services/tile_repository_manager.dart` | `No issues found! (ran in 5.4s)` | ✓ PASS |
| Full-app `flutter analyze` (ripple/regression check) | `cd app && flutter analyze` | 37 issues, all pre-existing (deprecated FontAwesome renames in `icon_util.dart`, dead code in `map_screen.dart`, unrelated `BuildContext`-async-gap infos) — zero in any phase-26 file | ✓ PASS (no new issues) |
| Full app-wide `flutter test` (broad regression check) | `cd app && flutter test` | 4 pre-existing failures, all in `test/components/route_planner/settings_tab_test.dart` (unrelated feature area — icon-widget-finder mismatch, last touched by an unrelated `quick-260720` commit); zero failures related to phase 26 files | ✓ PASS (no new failures; pre-existing, unrelated) |
| Gap 1 fix present and correctly scoped | `grep -c 'vectorLatched\|demLatched\|whenComplete\|trailSucceeded' lib/provider/trail/trail_download_state_provider.dart` → 5/5/4/5; direct read confirms monotonic-max semantics | Confirmed by source read | ✓ PASS |
| Gap 2 fix present and correctly scoped | `grep -c 'freshRegion'` → 8; `grep -c 'region.lastDownloadedVersion = region.version'` → 0; `_getOrCreatePackage` signature confirmed `{required bool dem}` | Confirmed by source read | ✓ PASS |
| 26-04 invariants unchanged (no regression) | `grep -c 'remove(trail.id)'` → 1; `grep -c 'invalidate(regionListNotifierProvider)'` → 1 | Confirmed by source read | ✓ PASS |
| `tile_repository_provider.dart` untouched (gap 1 constraint) | `grep -c '_clearVectorProgress\|_clearDemProgress' lib/provider/region/tile_repository_provider.dart` → 8, file absent from `26-05`'s `files_modified` | Confirmed by source read | ✓ PASS |
| Git commits for 26-05 exist | `git log --oneline -- app/lib/provider/trail/trail_download_state_provider.dart app/lib/services/tile_repository_manager.dart` | `455d3f79 fix(26-05): ...`, `1a0e1553 fix(26-05): ...`, both present immediately after `12a920ee`/`5d35ad84 fix(26-04): ...` | ✓ PASS |
| On-device confirmation of gap 1/gap 2 live behavior | n/a — requires a physical device/emulator with a live concurrent download | Not run in this environment; the underlying fix's static correctness is confirmed, but live behavior (real OS notification percentage over time; a real ObjectBox concurrent-write race outcome) cannot be observed via grep/static analysis | ? SKIP — routed to Human Verification below |

### Probe Execution

SKIPPED — no `scripts/*/tests/probe-*.sh` probes exist or are declared for this Flutter mobile phase; the plans' own `<verify>` blocks specify `flutter test`/`flutter analyze` (run above) plus deferred on-device human checks (`26-05-PLAN.md`'s own `<human-check>` blocks for Task 1 and Task 2).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| GUARD-01 | 26-01, 26-03, 26-04 | bbox coverage check vs downloaded/updateAvailable; robust never-stranded button | ✓ SATISFIED | Unchanged; confirmed regression-free |
| GUARD-02 | 26-02, 26-03, 26-04, **26-05** | Dialog names missing region(s) + size, in-dialog per-region action | ✓ SATISFIED | Unchanged core dialog behavior; 26-05 restores the promise that a selected DEM package actually registers as downloaded (static confirmation; live confirmation pending — Human Verification #2) |
| GUARD-03 | 26-02, 26-03, 26-04, **26-05** | Multi-region: individual + combined size, subset download, never forces full coverage; unified notification stable/accurate | ✓ SATISFIED | Core behavior unchanged; 26-05 restores the "one combined, advancing progress bar" promise (D-10) at the source-code level (static confirmation; live confirmation pending — Human Verification #1) |
| GUARD-04 | 26-01, 26-03, 26-04 | `updateAvailable` satisfies coverage identically to `downloaded`; guard never re-fires for a region already fixed | ✓ SATISFIED | Unchanged; confirmed on-device (26-UAT.md Test 5) |

All four IDs mapped to Phase 26 in `.planning/REQUIREMENTS.md` (lines 133-136) are marked "Complete". `26-05-PLAN.md`'s `requirements:` frontmatter declares `[GUARD-02, GUARD-03]` — both consistent with the two gaps it closes (a selected DEM package registering as downloaded is a GUARD-02/03 promise; the unified progress bar is GUARD-03's D-10). No orphaned requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `download_notification_service.dart` | 6 | Single shared notification id (42) for all trails | ⚠️ Warning (WR-03, unchanged, explicitly out of scope for both 26-04 and 26-05) | Pre-existing, documented, out of scope |
| `missing_coverage_sheet.dart` | 133, 202, 302 | Inconsistent null-fallback default for `_vectorChecked` | ℹ️ Info (WR-04, unchanged, explicitly out of scope) | Latent inconsistency only |
| `missing_coverage_sheet.dart` | 144-186, 283-328 | Hardcoded English strings mixed with `l10n.*` calls | ℹ️ Info (IN-01, unchanged, explicitly out of scope) | i18n inconsistency |
| `trail_coverage_util_test.dart` | n/a | No explicit `downloading`/`error` status test case | ℹ️ Info (IN-03, unchanged, explicitly out of scope) | Low risk, simple binary filter |
| `trail_download_state_provider.dart` (`whenComplete` latch) | ~178-186 | Each package's latch is forced to 1.0 in `whenComplete` regardless of whether the package's own download succeeded or errored | ℹ️ Info — matches `26-05-PLAN.md`'s explicit design intent ("each package's own future forces its latch to a full 1.0 on completion," Task 1 step 3), not an executor deviation | The aggregate notification will show 100%/success even if an individual region package silently failed; the failure still surfaces separately via the region's own Settings/Regions status (untouched contract). Worth a human eye during the Test 4 re-run if a package is deliberately made to fail, but not a phase-26 blocker since it is the plan's own documented tradeoff |
| Working tree | `missing_coverage_sheet.dart`, `trail_download_service.dart` | Uncommitted local modifications present (cosmetic text-style change + formatting reflow) unrelated to any 26-0X plan's `files_modified` | ℹ️ Info | Not part of phase 26 scope; the committed HEAD state of these files (which is what phase 26's plans/summaries describe) is unaffected — noted for hygiene, not a phase gap |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` debt markers found in either of the 2 phase-26-05 modified files (re-confirmed by grep on current HEAD).

### Human Verification Required

Both items below are the direct `<human-check>` blocks declared in `26-05-PLAN.md` itself (`human_verify_mode: end-of-phase`), harvested per the standard end-of-phase deferral pattern this whole phase has consistently used. They correspond exactly to 26-UAT.md's Test 4 and Test 8, now that the underlying fixes are statically confirmed present and correctly scoped.

### 1. Unified progress bar advances monotonically to 100% (26-UAT.md Test 4 re-run)

**Test:** On a multi-region trail, check Vector for two regions and DEM for one, tap Download once.
**Expected:** The single id-42 "Downloading offline content" notification shows ONE combined, monotonically advancing progress bar that does NOT reset/jump downward as each package completes, reaches 100%, and only then finalizes to a success state. (Falsification cross-check per the debug session: a single-package selection with no DEM should show zero downward resets at all.)
**Why human:** Requires observing real OS notification content/percentage transitions over time during a live, concurrent multi-download session — not derivable from static analysis alone, even though the monotonic-latch fix is confirmed present and correctly wired in source.

### 2. DEM package shows as downloaded in Settings after concurrent download (26-UAT.md Test 8 re-run)

**Test:** Trigger the missing-coverage sheet for a trail with a missing region, check the DEM checkbox (leave Vector checked too), tap Download.
**Expected:** After both finish, Settings → Offline Regions shows that region's DEM package as "downloaded" (matching Vector), and the region's totals remain consistent — no orphaned/severed DEM relation.
**Why human:** Requires a live ObjectBox concurrent-write race between two real, independently-timed downloads on-device — the fresh-row read-modify-write fix is confirmed present and correctly scoped in source, but its actual effect on the persisted relation under real timing can only be ground-truthed on-device.

### Gaps Summary

No blocking gaps remain at the source-code level. Both UAT-found issues (Test 4's downward-resetting/sub-100% progress bar, Test 8's DEM-relation clobbering race) are independently confirmed closed by direct re-inspection of the current `trail_download_state_provider.dart` and `tile_repository_manager.dart` — not by trusting the `26-05-SUMMARY.md`'s narrative. Both fixes are correctly scoped (the latch survives, but does not suppress, `tile_repository_provider.dart`'s intentional clear-on-completion; the fresh-row re-fetch preserves full concurrency between Vector and DEM downloads, adding no lock/serialization), and both underlying commits (`1a0e1553`, `455d3f79`) exist in git history for these exact files. `flutter analyze` is clean on both modified files and shows zero new issues app-wide (the same 37 pre-existing, unrelated issues remain). `flutter test test/services/tile_repository_manager_test.dart` (16/16) and `flutter test test/util/trail_coverage_util_test.dart` (11/11) both pass; the only app-wide test failures (4, in `test/components/route_planner/settings_tab_test.dart`) are pre-existing and entirely unrelated to phase 26.

Status is `human_needed` rather than `passed` because the two fixes this round specifically targets are, by their own nature (a live notification-percentage sequence over time; a live concurrent-database-write race), only observable end-to-end on a real device — exactly the same category of truth (visual/timing/OS-integration) that made 7 items require human verification in the prior round, and consistent with this phase's own declared `human_verify_mode: end-of-phase` and `26-05-PLAN.md`'s own `<human-check>` blocks for both tasks. This is not a code-level deficiency finding — it is the same escalation pattern already used successfully once in this phase (the 26-04 round's CR-02/CR-01/WR-01 items, which the user's own on-device UAT subsequently confirmed all passed).

All 5 previously-passed on-device UAT items (fully-covered start, sheet appearance/dismiss, no-region-gap warning, CR-02 live re-fire, WR-01 notification stability, CR-01 button-stranding — 6 items, all confirmed pass in `26-UAT.md`) show no regression under static analysis: their supporting code paths (`overlappingRegions`/`missingCoverageRegions` branch, `missing_coverage_sheet.dart`, the toast branch, the single `invalidate(regionListNotifierProvider)` occurrence, `onGeneratingChanged`'s `hasSelectedPackages` branch, and the single `remove(trail.id)` occurrence) are all textually unchanged or unaffected by the 26-05 diff.

WR-03 (shared notification id 42), WR-04 (inconsistent checkbox null-fallback default), IN-01 (mixed hardcoded/localized strings), and IN-03 (missing `downloading`/`error` status test cases) remain as documented, explicitly out-of-scope residual findings — not silent omissions, and not blocking this phase's goal achievement.

---

*Verified: 2026-07-24T15:10:00Z*
*Verifier: Claude (gsd-verifier)*
