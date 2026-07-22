---
phase: 24-settings-offline-maps-regions-ui
verified: 2026-07-22T14:20:56Z
status: human_needed
score: 6/6 must-haves verified (source + unit-test level); on-device re-confirmation of UAT tests 2, 3, 4, 5 outstanding
overrides_applied: 0
re_verification:
  previous_status: human_needed (initial VERIFICATION.md, 2026-07-22, source-level only) -> gaps_found (24-UAT.md, on-device testing surfaced 2 blockers) -> gap-closure plan 24-03 executed
  previous_score: "24-UAT.md: 2 passed, 2 issues (blocker), 2 blocked"
  gaps_closed:
    - "UAT test 2 root cause (freeDiskSpaceBytes throws/swallows 'Specified path does not exist' on a never-downloaded region's not-yet-created regions/<id>/ directory, hasEnoughSpace(null,...) fails closed, download marked error) — fixed at the code level by resolveFreeDiskSpaceBytes's device-wide fallback in disk_space_util.dart, confirmed against the actual installed disk_space_2@1.0.12 plugin source (getFreeDiskSpaceForPath explicitly throws Exception('Specified path does not exist') via an existsSync() guard, matching the fallback's catch branch exactly)."
    - "UAT test 3 (same root cause, DEM call site) — fixed by the same shared-wrapper change; tile_repository_manager.dart confirmed unmodified (git diff --stat empty for that file across the 24-03 commits), so both call sites now benefit identically."
  gaps_remaining:
    - "On-device re-confirmation of UAT tests 2 and 3 on a physical Android device against a never-before-downloaded region has NOT been performed — code-level fix verified, physical-device behavior not independently observable by this verifier."
    - "UAT tests 4 and 5 (previously blocked_by: prior-phase, blocked by tests 2/3's failure) remain unexecuted and need to be re-run now that a region can complete its first download."
  regressions: []
gaps: []
human_verification:
  - test: "Re-run 24-UAT.md test 2: on a physical Android device, pick a region that has NEVER been downloaded before, tap 'Download vector', watch it transition to `downloading` (progress bar advances, not `error`). Then pause it, resume it, and delete it via the confirm dialog."
    expected: "Package transitions to downloading (not error); pause/resume/delete all function as originally specified; the device log line 'freeDiskSpaceBytes gives: Specified path does not exist' no longer appears."
    why_human: "This is the exact real-filesystem scenario the 24-03 fix targets (StatFs/NSFileManager path-does-not-exist behavior only reproduces on a real device, not necessarily the emulator per the plan's own note); requires physical hardware, live download timing, and disk I/O this verifier cannot execute."
  - test: "Re-run 24-UAT.md test 3: toggle DEM on for that same never-downloaded region."
    expected: "DEM download starts (spinner, in-flight feedback), not error."
    why_human: "Same real-device-only path-does-not-exist scenario as test 2, at the DEM call site."
  - test: "Re-run 24-UAT.md test 4 (previously blocked): watch the disk-usage total as packages are added/removed, including while a download is paused mid-transfer."
    expected: "Total updates after each mutation and includes partial (.part) bytes for a paused/in-progress download."
    why_human: "Live FutureBuilder recomputation timing and real on-disk byte accounting; was blocked by tests 2/3 failing (no region could ever finish a first download) and can now be attempted."
  - test: "Re-run 24-UAT.md test 5 (previously blocked): with airplane mode on and at least one previously-downloaded region, open the screen."
    expected: "The previously-downloaded region still appears and is usable; the screen does not blank or show a full-screen error just because the catalog refresh failed offline."
    why_human: "Requires a real offline network condition and a region that has actually completed a download, which was blocked by tests 2/3's failure."
---

# Phase 24: Settings — Offline Maps/Regions UI Verification Report (Re-verification after gap-closure 24-03)

**Phase Goal:** Settings — Offline Maps/Regions UI: Flat searchable region list with download/pause/resume/delete, DEM toggle, and total disk usage.
**Verified:** 2026-07-22T14:20:56Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plan 24-03 (device-wide disk-space fallback), following 24-UAT.md's two diagnosed on-device blockers.

## Context

The initial (2026-07-22) 24-VERIFICATION.md source-level pass returned `human_needed` (6/6 truths verified at the source level, but on-device confirmation of the roadmap's device-dependent behaviors was explicitly outstanding). Physical-device UAT was then run (`24-UAT.md`): tests 2 and 3 both failed with the identical blocker — `freeDiskSpaceBytes gives: "Specified path does not exist"` on a never-before-downloaded region — which cascaded to block tests 4 and 5 as well. A debug session (`.planning/debug/region-download-diskspace.md`) diagnosed the root cause, and gap-closure plan `24-03-PLAN.md` was executed to fix it. This report re-verifies whether that fix is actually present, substantive, wired, and correct in the codebase — not whether SUMMARY.md says it is.

## Goal Achievement

### Observable Truths (24-03 gap-closure scope)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | On a region whose `regions/<id>/` directory does not yet exist, `freeDiskSpaceBytes` returns a real device-wide free-space value instead of `null` (SETUI-03) | VERIFIED (code-level) | `app/lib/util/disk_space_util.dart:95-101`: `freeDiskSpaceBytes` delegates to `resolveFreeDiskSpaceBytes` (lines 54-81), whose catch block (line 65) retries `deviceQuery()` on any path-query exception before returning `null`. Confirmed against the actual installed `disk_space_2@1.0.12` source (`~/.pub-cache/hosted/pub.dev/disk_space_2-1.0.12/lib/disk_space_2.dart:36-42`): `getFreeDiskSpaceForPath` explicitly does `if (!Directory(path).existsSync()) throw Exception("Specified path does not exist")` — the exact string from the UAT bug report — so the fallback's catch branch is reliably exercised for this real scenario, not a hypothetical one. |
| 2 | Starting a vector download for a never-downloaded region transitions the package to `downloading`, not `error`, on a real device (SETUI-03) | UNVERIFIABLE BY THIS VERIFIER (routed to human) | `tile_repository_manager.dart:116-129` (`startVectorDownload`) is confirmed unmodified (`git diff --stat` empty for that file across all 24-03 commits) and still calls `await freeDiskSpaceBytes(regionStorageDir(root, id))` before `Directory(...).createSync(...)` at line 129 — it now benefits from the fallback without any reordering. The code-level chain is sound, but "transitions to downloading on a real device" is an on-device runtime observation this verifier cannot produce. |
| 3 | Toggling DEM on for a never-downloaded region starts the DEM download (spinner, not error) on a real device (SETUI-04) | UNVERIFIABLE BY THIS VERIFIER (routed to human) | Same fix, same reasoning, at `startDemDownload` (`tile_repository_manager.dart:201-214`) — call site unmodified, now backed by the same fallback wrapper. On-device confirmation outstanding. |
| 4 | When BOTH the path-specific and device-wide queries fail, `hasEnoughSpace` still fails closed and refuses the download — TILE-03's genuine low-disk protection is preserved | VERIFIED | `resolveFreeDiskSpaceBytes` returns `null` when both `pathQuery` and the retried `deviceQuery` throw (lines 71-77), exercised by unit test `'path query throws AND device query throws -> returns null (fail closed, TILE-03 preserved)'` (`disk_space_util_test.dart:118-130`) — **ran and passed** (see Spot-Checks below). `hasEnoughSpace`'s own fail-closed contract (`freeBytes == null -> false`, line 119) is byte-for-byte unchanged and its 5 pre-existing unit tests still pass. |

**Score:** 2/4 gap-closure truths fully code-verified; 2/4 require on-device confirmation this verifier cannot perform (correctly routed to human verification, not claimed as passed).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/disk_space_util.dart` | `resolveFreeDiskSpaceBytes` injectable orchestrator implementing path→device-wide fallback | VERIFIED | Declared line 55 (`Future<int?> resolveFreeDiskSpaceBytes({...})`), `@visibleForTesting`, `freeDiskSpaceBytes` (line 95) delegates to it as a thin wrapper wiring real `DiskSpace.getFreeDiskSpaceForPath`/`DiskSpace.getFreeDiskSpace`. Not a stub — full fallback/retry logic present and matches plan spec exactly. |
| `app/test/util/disk_space_util_test.dart` | Unit tests covering the 6 fallback cases with injected fakes, no platform channel | VERIFIED | `resolveFreeDiskSpaceBytes` group (lines 85-171) has 6 tests exactly matching the plan's spec (path succeeds/device skipped, path throws→device fallback, both throw→null, forPath null→device direct, forPath null+device throws→null, null mebibytes→null bytes). All pre-existing `hasEnoughSpace` tests (5) retained unmodified. |

Both artifacts pass all 3 levels (exist, substantive, wired) per acceptance-criteria greps run directly against the working tree:
- `grep -n 'Future<int?> resolveFreeDiskSpaceBytes' lib/util/disk_space_util.dart` → line 55, present.
- `grep -n 'resolveFreeDiskSpaceBytes(' lib/util/disk_space_util.dart` → lines 55 (decl) and 96 (wrapper delegation) — wired.
- `grep -n 'deviceQuery(' lib/util/disk_space_util.dart` → lines 64 (primary no-path path) and 72 (fallback retry) — both invocation sites present.
- `grep -c 'resolveFreeDiskSpaceBytes' test/util/disk_space_util_test.dart` → 9 (exceeds the plan's `>= 6` threshold).

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `disk_space_util.dart:freeDiskSpaceBytes` | `resolveFreeDiskSpaceBytes` | thin wrapper wiring real plugin queries (`DiskSpace.getFreeDiskSpaceForPath`, `() => DiskSpace.getFreeDiskSpace`) | WIRED | Lines 95-101; both real plugin calls correctly passed as `PathSpaceQuery`/`DeviceSpaceQuery` closures. |
| `resolveFreeDiskSpaceBytes` catch block | `deviceQuery` | device-wide retry on path-specific query failure before returning `null` | WIRED | Line 72, inside the `catch (e)` at line 65, nested `try`/`catch` correctly returns `null` only if the retry also throws (line 76). |
| `tile_repository_manager.dart:startVectorDownload`/`startDemDownload` | `freeDiskSpaceBytes` | `await freeDiskSpaceBytes(regionStorageDir(root, id))` | WIRED, UNCHANGED | Lines 116 and 201; confirmed identical to pre-fix version (`git log` shows no plan-03 commit touching this file; last change was 24-01's `7b725b2b`, unrelated). Both call sites transparently inherit the fallback with zero code changes at the call site, exactly as the plan required. |

### Data-Flow / Behavioral Trace

| Check | Method | Result | Status |
|-------|--------|--------|--------|
| `disk_space_2@1.0.12`'s actual `getFreeDiskSpaceForPath` throws (not returns null) on a nonexistent path | Read installed package source directly (`~/.pub-cache/hosted/pub.dev/disk_space_2-1.0.12/lib/disk_space_2.dart:36-42`) | `throw Exception("Specified path does not exist")` via an `existsSync()` guard — matches the UAT log line verbatim | CONFIRMED: the fix's catch-and-retry branch is the branch that actually fires for this real bug, not a hypothetical/untested path |
| `flutter test test/util/disk_space_util_test.dart` | Executed directly by this verifier | `00:00 +11: All tests passed!` (5 `hasEnoughSpace` + 6 `resolveFreeDiskSpaceBytes`) | PASS |
| `flutter analyze lib/util/disk_space_util.dart test/util/disk_space_util_test.dart` | Executed directly by this verifier | `No issues found!` | PASS |
| `flutter analyze lib/routes/settings_offline_regions_screen.dart` | Executed directly by this verifier (regression check — file has uncommitted, phase-24-unrelated cosmetic layout/styling edits in the working tree) | `No issues found!` | PASS (no regression; core widgets/functions `_buildDiskUsageSummary`, `_combinedProgress`, `WandererSearchBar`, DEM `Switch` all still present) |
| `flutter test test/util/region_disk_usage_util_test.dart test/util/byte_format_util_test.dart` | Executed directly by this verifier (regression check on Phase 24's other unit-tested artifacts) | `00:00 +11: All tests passed!` | PASS, no regression |
| `git diff --stat` across 24-03 commits (`b9e07055`, `8be39c5a`) for `tile_repository_manager.dart` | Executed directly by this verifier | Empty — file untouched | Confirms plan's "explicitly NOT changed" commitment held |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `app/lib/util/disk_space_util.dart` | 51-79 | WR-01 (from independent `24-REVIEW.md`, re-confirmed present in current source): fallback only triggers on a *thrown* path-query exception, not a quiet `null` return | Warning (non-blocking) | Low real-world exposure — the actual installed `disk_space_2@1.0.12` plugin always throws (never returns null) for the "path doesn't exist" case that caused the UAT blocker (confirmed by direct source read above), so this warning does not affect the fix's correctness for the diagnosed bug. Left open by design; not part of the 24-03 plan's must-haves. |
| `app/lib/util/disk_space_util.dart` | 65, 73 | WR-02 (re-confirmed): caught exceptions are silently discarded with no `debugPrint`/log | Warning (non-blocking) | A future *different* failure mode (e.g. both queries genuinely failing on a production device) would be invisible/undiagnosable; doesn't affect the diagnosed bug's fix. |

No debt markers (`TODO`/`FIXME`/`TBD`/`XXX`/`HACK`/`PLACEHOLDER`) found in `disk_space_util.dart`, `disk_space_util_test.dart`, or `settings_offline_regions_screen.dart`.

**Note on unrelated working-tree state:** `app/lib/routes/settings_offline_regions_screen.dart` carries uncommitted cosmetic changes (disk-usage summary moved below the search bar, bordered/shaded containers added) not attributable to any Phase 24 plan's `files_modified` list and not mentioned in any Phase 24 SUMMARY. These appear to be unrelated, in-progress UI-polish work. They do not affect any Phase 24 truth (all referenced functions/widgets remain present, `flutter analyze` clean), so they are noted here for transparency but are out of scope for this phase's gap-closure re-verification and not treated as a gap.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SETUI-01 | 24-01, 24-02 | Flat, searchable region list | SATISFIED | Unchanged since initial verification; re-confirmed present (`WandererSearchBar`, flat `ListView`). |
| SETUI-02 | 24-01, 24-02 | Name, status, size breakdown before download | SATISFIED | Unchanged since initial verification. |
| SETUI-03 | 24-02, 24-03 (gap-closure) | Download/pause/resume/delete actions available per region | SATISFIED (code-level); on-device re-confirmation of UAT tests 2/4 outstanding | Root-cause fix (`resolveFreeDiskSpaceBytes`) implemented and unit-tested; call sites unchanged and now benefit from it. |
| SETUI-04 | 24-01, 24-02, 24-03 (gap-closure) | Per-region DEM toggle | SATISFIED (code-level); on-device re-confirmation of UAT test 3 outstanding | Same fix applied identically to the DEM call site. |
| SETUI-05 | 24-01, 24-02 | Total disk usage summary | SATISFIED | Unchanged since initial verification; on-device re-confirmation of UAT test 4 (live updates incl. `.part` bytes during a paused download) still outstanding, now unblocked by the 24-03 fix and re-runnable. |
| SETUI-06 | 24-02 | `updateAvailable` badge | SATISFIED | Unchanged since initial verification. |

No orphaned requirements — `REQUIREMENTS.md` maps SETUI-01..06 exclusively to Phase 24 (lines 40-45, 115-120), all marked `[x]` Complete, and all six appear across the `requirements:` frontmatter of `24-01-PLAN.md`, `24-02-PLAN.md`, and `24-03-PLAN.md`.

### Human Verification Required

See frontmatter `human_verification` list. Four items, all re-runs of `24-UAT.md`'s previously-failed/blocked tests (2, 3, 4, 5) against a physical device, now that the diagnosed root cause has a code-level fix. This verifier confirmed the fix is real, substantive, correctly wired to both call sites, matches the actual installed plugin's throw behavior for the exact bug reported, and is covered by passing unit tests — but per this project's `human_verify_mode: end-of-phase` and the explicit instruction for this re-verification, on-device confirmation is not something this verifier can perform or claim on the executor's behalf.

### Gaps Summary

No code-level gaps found. The 24-UAT.md-diagnosed blocker (tests 2 and 3: every first-ever region download refused with "Specified path does not exist") has a correct, targeted, unit-tested fix in `disk_space_util.dart`'s `resolveFreeDiskSpaceBytes`, verified against the actual installed `disk_space_2@1.0.12` plugin source rather than assumed behavior. `tile_repository_manager.dart` is confirmed unmodified, so the fix's zero-call-site-change design held. All automated checks (`flutter test`, `flutter analyze`) pass with no regressions in adjacent Phase 24 artifacts. Status is `human_needed`, not `passed`, because the phase's roadmap success criteria describe device-dependent, user-observable behavior (a download actually starting, a spinner actually appearing, a disk-usage total actually updating live) that only a physical-device UAT re-run can confirm — this verifier cannot execute that re-run itself, and SUMMARY.md's own text explicitly defers it ("Outstanding before Phase 24 can be considered fully signed off: the end-of-phase on-device human-check pass").

---

_Verified: 2026-07-22T14:20:56Z_
_Verifier: Claude (gsd-verifier)_
