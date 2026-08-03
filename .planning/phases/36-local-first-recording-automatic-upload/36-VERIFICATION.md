---
phase: 36-local-first-recording-automatic-upload
verified: 2026-08-03T21:50:00Z
status: human_needed
score: 11/11 requirements satisfied at the code level (6/6 roadmap success criteria hold; all
  previously-open blockers and the round-2 UAT badge gap are closed; one WARNING-level l10n
  coverage gap is a pre-existing, already-tracked deferral, not new debt)
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "11/11 requirements satisfied at the code level; 5 human_verification items
    outstanding (round-2 UAT, all pending)"
  gaps_closed:
    - "All 5 of 36-VERIFICATION.md's prior human_verification items -- 36-UAT.md's round 2 was
      actually run on a physical device (commit cde38b33, 'complete round 2 UAT - 5 passed, 1
      minor gap') and all 5 tests, including the required UX judgment call in Test 4, PASSED."
    - "The one gap round 2 UAT surfaced -- an unsynced trail's detail screen showed a generic
      'Offline' badge instead of its upload state -- is closed by plan 36-21: TrailPanel's badge
      is now keyed on trail.syncState via a gated SyncStatusChip render, with the pre-existing
      isLocal-keyed Offline pill narrowed to exclude unsynced trails. Verified independently in
      this pass (see below), not trusted from 36-21-SUMMARY.md."
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "On device, open the trail that reproduced round 2 UAT test 1's badge gap: an unsynced
      trail from the own-trails list. Its detail screen must read Waiting to upload (or
      Uploading… mid-drain), never Offline. Force a failure (airplane mode until it parks) and
      confirm the detail screen reads Upload failed · Tap to retry and that tapping it starts a
      retry. Then open an ordinary downloaded trail's detail screen and confirm it still reads
      Offline and looks exactly as it did before."
    expected: "Unsynced trail's detail screen shows the sync-state badge (Waiting to upload /
      Uploading… / Upload failed · Tap to retry) with a working retry tap; a downloaded trail's
      detail screen is visually unchanged (still reads Offline)."
    why_human: "This is the exact <human-check> block 36-21-PLAN.md deferred to end-of-phase
      (per the workflow's human_verify_mode=end-of-phase convention). 36-21-SUMMARY.md's own
      'Human-check (deferred to device pass)' section states plainly: 'Not performed in this
      session.' A widget test (trail_panel_sync_badge_test.dart) mounts the real TrailPanel and
      independently confirms the fix's logic in this pass, but the fix itself has never been
      seen rendered on a physical device -- the round-2 UAT pass that would have covered this
      predates the fix by definition (it is the pass that found the bug)."
---

# Phase 36: Local-First Recording & Automatic Upload Verification Report

**Phase Goal:** A recording saves instantly with no connection, stays in the hiker's own-trails
list, and uploads itself once the phone is back online.

**Verified:** 2026-08-03T21:50:00Z
**Status:** human_needed
**Re-verification:** Yes — this pass verifies plan 36-21 (the detail-screen sync badge gap
closure, the last of 21 plans in the phase) against the prior `36-VERIFICATION.md` pass, which
had already confirmed all CRITICAL/blocker findings closed and all WARNING findings closed or
deliberately deferred.

## What Changed Since the Last Verification Pass

The prior `36-VERIFICATION.md` (2026-08-03T21:15:00Z) concluded `human_needed` on code-level
truths that were fully verified (11/11 requirements, 6/6 roadmap success criteria), gated only on
five outstanding device-verification items mirroring `36-UAT.md` round 2's then-`pending` tests.

Since that pass:

1. **Round 2 UAT was actually run on a physical device** (`36-UAT.md`, commit `cde38b33`): all 5
   tests passed, including the required UX judgment call in Test 4 (the CR-03 narrowed
   offline-edit behavior was explicitly accepted by the user as shippable). This closes all five
   of the prior pass's `human_verification` items.
2. **Round 2 surfaced one minor, new gap**: an unsynced trail's *detail screen* (as opposed to
   the list surfaces) rendered a generic "Offline" badge instead of its sync state, because
   `TrailPanel`'s pre-Phase-36 badge was keyed on `trail.isLocal` (cache provenance) rather than
   `trail.syncState`. `SyncStatusChip` — the four-state indicator this phase built — had only
   ever been wired into the two list surfaces (`trail_list_item.dart`, `trail_card.dart`), never
   into `TrailPanel`.
3. **Plan 36-21 was written and executed to close that gap.** This pass independently verifies
   36-21's claims against current source, not from `36-21-SUMMARY.md`'s narration.

## Goal Achievement

### Plan 36-21 — Independent Verification

| Claim | Verdict | Evidence |
|---|---|---|
| `TrailPanel`'s Offline pill is narrowed to exclude unsynced trails | ✓ VERIFIED | `trail_panel.dart:205`: `if (trail.isLocal && !isUnsyncedState(trail.syncState))`, read directly from the current file, with a comment documenting the D-10 partition reasoning. |
| A `SyncStatusChip` renders below the title for unsynced trails only | ✓ VERIFIED | `trail_panel.dart:296-302`: `if (isUnsyncedState(trail.syncState)) ... Align(... SyncStatusChip(trail: trail))`, gated (not relying on the chip's own self-suppression), placed below the title per the plan's overflow-avoidance rationale. |
| `isUnsyncedState` correctly spans all three non-synced states (pending/uploading/failed) | ✓ VERIFIED | `trail_sync_state.dart:16`: `bool isUnsyncedState(TrailSyncState state) => state != TrailSyncState.synced;` — a single predicate, not an enumerated subset, so the guard is correct for `uploading` even though (per 36-21-REVIEW's WR-03) no panel-level test pins that specific state. |
| A widget test mounts the real `TrailPanel` and pins all 5 badge cases (pending/in-flight/failed/downloaded-control/remote-control) | ✓ VERIFIED — independently run, not trusted from SUMMARY | `flutter test test/components/trail/trail_panel_sync_badge_test.dart` run directly in this pass: **5/5 pass** (`Case A` through `Case E`, matching the plan's required case names). File is 284 lines, mounts the real widget tree (no source-text grepping), confirmed by reading the file directly. |
| Commits exist as claimed | ✓ VERIFIED | `git log` on `trail_panel.dart` shows `bdfde398` ("fix(36-21): key the detail-screen badge on sync state, not cache provenance") directly on top of `9419f872` ("test(36-21): RED — mount real TrailPanel, pin all four badge cases"), both present in history. |
| Full test suite and analyzer are clean after this change | ✓ VERIFIED — independently re-run | `flutter test`: **946 passed, 1 skipped, 0 failures**. `flutter analyze --no-pub`: **0 errors, 36 pre-existing info-level lints** (same baseline as the prior pass; all in vendored/deprecated-icon files unrelated to this change). |
| No new debt markers in the touched files | ✓ VERIFIED | `grep -n "TODO\|FIXME\|XXX\|TBD\|HACK\|PLACEHOLDER"` on `trail_panel.dart` and the new test file: no matches. |

### Delta Code Review (36-21-REVIEW.md) — Findings Assessed Against the Phase Goal

`36-21-REVIEW.md` found **0 blockers**, 4 warnings, 5 info. None block the phase goal:

| ID | Finding | Phase-goal impact | Disposition |
|---|---|---|---|
| WR-01 | New `sync_pending`/`sync_uploading`/`sync_failed` strings replace a translated `l18n.offline` string on the detail screen, regressing 13 non-English locales to English-only on this surface | Non-blocking — a UX/i18n coverage gap, not a functional defect. **Already tracked**: `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md` (created earlier, from plan 36-19) already lists `sync_pending`/`sync_uploading`/`sync_failed` as Priority-4 items in its 22-key backlog — this is not new, undocumented debt introduced by 36-21; it is an instance of an already-accepted, already-tracked deferral. | Accepted deferral, consistent with the phase's established pattern for translation work (verified: the todo file's Priority 4 list, items 8-10, names exactly these three keys). |
| WR-02 | The Offline-pill guard and the chip guard are asymmetric in shape (`isLocal && !unsynced` vs. `unsynced` alone) — correctness today rests on an unstated but currently-true invariant (`syncState != synced ⇒ isLocal == true`) | Non-blocking — a robustness/defensive-coding suggestion. Confirmed the invariant holds today (`TrailEntity.toModel()` hardcodes `isLocal: true` for every cached row; `Trail.syncState` defaults to `synced` and is JSON-excluded, so no server-parsed trail can carry a non-synced state). | Warning, not a gap. |
| WR-03 | No panel-level test exercises a *persisted* `TrailSyncState.uploading` row (Case B only reaches "Uploading…" via the in-flight set, not the persisted-state branch) — the guard could theoretically be narrowed to `pending \|\| failed` and all 5 tests would still pass | Non-blocking for the CURRENT implementation — independently confirmed `isUnsyncedState` is `!= synced`, a single predicate covering all three non-synced states, not an enumerated subset; the actual shipped code is correct for `uploading` today. This is a test-coverage gap (a future regression could go unpinned), not a present functional defect. | Warning, not a gap. |
| WR-04 | No narrow-viewport (360px) test proves the documented overflow-avoidance rationale for chip placement | Non-blocking — a test-coverage gap for a design rationale, not an observed defect. | Warning, not a gap. |

No BLOCKER or must-have-failing finding exists in the delta review. The one item with genuine
user-facing UX consequence (WR-01) is a pre-existing, already-tracked, deliberately deferred
translation-coverage gap — not new undisclosed debt.

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | Capturing with no connection saves instantly into the own-trails list, no offline save-failure ever shown, unsynced visibly distinct from synced AND downloaded | ✓ VERIFIED | Unchanged from the prior pass; confirmed again this pass on the detail-screen surface specifically — with 36-21, "visibly distinct" now holds structurally on all three surfaces (list item, card, detail screen), not just the two list surfaces. |
| 2 | Survives app restart, stays tied to the capturing account; a different account never sees/uploads it; logout never deletes it | ✓ VERIFIED | Unchanged; not touched by 36-21 (36-21 is UI-only, no schema/sync-logic change, confirmed via `git diff --stat` scope: `trail_panel.dart` and the new test file only). |
| 3 | Open/review/edit an unsynced trail's title, description, category, photos while offline | ⚠️ VERIFIED with the same narrowed edge case as the prior pass (accepted by the user in round-2 UAT Test 4's judgment call) | `36-UAT.md` round 2, Test 4: `result: pass`, `judgment: The narrowed offline-edit behaviour in this sub-state is ACCEPTED by the user as shippable.` Confirmed read directly from the UAT file. |
| 4 | Once foregrounded with connection, uploads on its own with inline per-item progress, manual retry on failure/stall — visible on the trail itself | ✓ VERIFIED — now also true on the detail screen | Prior pass confirmed the list surfaces; this pass confirms the detail screen via 36-21's independently-verified fix. SYNC-02's "visible on the trail itself" now holds on every surface a hiker can view an unsynced trail from. |
| 5 | An interrupted upload never produces a duplicate trail on retry; once uploaded the trail becomes ordinary in place, keeping its identity | ✓ VERIFIED (code-level; confirmed on device in round-2 UAT Test 5, `result: pass`) | `36-UAT.md` round 2 Test 5 explicitly covers this against a live device+server session and passed. |
| 6 | With no connection the own-trails list still renders, shows every not-yet-uploaded trail plus authored-and-downloaded trails, states it's offline-only | ✓ VERIFIED | Unchanged; confirmed on device in round-2 UAT Test 2, `result: pass`. |

**Score:** 6/6 roadmap success criteria hold, now confirmed both at the code level and (for 5 of
6, via round-2 UAT) on a physical device against a live server. Criterion 3 still carries the
same deliberate, narrow, user-accepted edge case from the CR-03 fix.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `app/lib/components/trail/trail_panel.dart` | Sync-state-keyed badge on the detail screen, isLocal Offline badge narrowed to synced trails | ✓ VERIFIED | Read directly; both edits present and correctly gated (see table above). |
| `app/test/components/trail/trail_panel_sync_badge_test.dart` | Behavioral widget test mounting the real `TrailPanel` for all four+one badge cases | ✓ VERIFIED | 284 lines (exceeds the 120-line must_have minimum); mounts real widget tree; 5/5 pass, independently re-run. |
| `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md` | Tracks the l10n coverage gap WR-01 restates | ✓ VERIFIED | Exists, already names `sync_pending`/`sync_uploading`/`sync_failed` at Priority 4 (items 8-10) — pre-dates 36-21, so WR-01 is not undisclosed new debt. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `trail_panel.dart` | `SyncStatusChip` | `isUnsyncedState`-gated render below the title | ✓ WIRED | `:296-302`; pattern `SyncStatusChip(trail: trail)` present and gated as specified in the plan's `must_haves.key_links`. |
| `trail_panel.dart`'s Offline-pill guard | `trail.syncState` | guard now also requires `!isUnsyncedState` | ✓ WIRED | `:205`; pattern `isLocal && !isUnsyncedState` present verbatim. |
| `SyncStatusChip`'s failed-state tap | `TrailSync.retry` | `onTap` in `_Chip` | ✓ WIRED | Confirmed via the widget test's Case C, which taps the chip's `InkWell` and asserts `TrailSync.retry` was called exactly once with the local id — independently re-run, passed. |

### Requirements Coverage

| Requirement | Status | Evidence |
|---|---|---|
| REC-01 | ✓ SATISFIED | Unchanged; local-first branches confirmed network-free (untouched by 36-21). |
| REC-02 | ✓ SATISFIED | Unchanged; `/trail/local/:localId` addressing intact. |
| REC-03 | ✓ SATISFIED — now also true on the detail screen | 36-21 closes the last surface where an unsynced trail was indistinguishable from a downloaded one. |
| REC-04 | ✓ SATISFIED | Unchanged; not touched by this plan. |
| REC-05 | ⚠️ SATISFIED WITH THE SAME NARROWED EDGE CASE, user-accepted per round-2 UAT Test 4 | See Observable Truth 3. |
| REC-06 | ✓ SATISFIED | Unchanged; confirmed on device (round-2 UAT Test 2). |
| SYNC-01 | ✓ SATISFIED | Unchanged; confirmed on device (round-2 UAT Test 5). |
| SYNC-02 | ✓ SATISFIED — now also true on the detail screen | 36-21 is precisely this requirement's closure on the last unwired surface. |
| SYNC-03 | ✓ SATISFIED — now also true on the detail screen | Retry tap-through confirmed reaching `TrailSync.retry` from the detail screen via the widget test. |
| SYNC-04 | ✓ SATISFIED (code-level + device-confirmed) | Round-2 UAT Test 5 passed on device. |
| SYNC-05 | ✓ SATISFIED (code-level + device-confirmed) | Round-2 UAT Test 5 passed on device. |

No orphaned requirements — all 11 IDs assigned to Phase 36 in `.planning/REQUIREMENTS.md` are
`[x]` (marked complete) and are traceable to at least one plan and one piece of verified evidence
above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `app/lib/i18n/*.arb` (13 non-English locales) | — | `sync_pending`/`sync_uploading`/`sync_failed` remain English-only, now surfaced on the detail screen too (WR-01) | ⚠️ Warning, deliberately deferred (pre-existing tracked debt, not new) | Already itemized in `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md` at Priority 4. Non-blocking; the app's documented English-fallback behavior means this does not break the surface, only its polish for 13 locales. |
| No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers | — | Scanned `trail_panel.dart` and the new test file | — | Clean. |

### Uncommitted Working-Tree State — Flagged, Not Fixed

The working tree carries uncommitted modifications to files outside this phase's executed scope,
per this run's stated context (a concurrent session): `app/lib/components/base/wanderer_sort_chip_group.dart`,
`app/lib/components/category/category_icon.dart`, `app/lib/components/list/list_list_item.dart`,
`app/lib/components/route_planner/route_anchor_list_tab.dart`, `app/lib/components/route_planner/settings_tab.dart`,
`app/lib/components/trail/trail_list_item.dart`, `app/lib/routes/library_screen.dart`,
`app/lib/routes/list_screen.dart`, `app/lib/routes/profile_screen.dart`,
`app/lib/routes/trail_create_screen.dart`, and `.planning/phases/36-local-first-recording-automatic-upload/36-UAT.md`
(only its frontmatter `status:` field differs: committed `complete`, working tree `diagnosed` —
no body-content difference).

`trail_create_screen.dart` is a core file for this phase's save-routing logic, so its uncommitted
diff was inspected directly: the changes are additive and orthogonal (unsaved-changes-dialog
dirty-tracking via a new `_syncUnsavedChanges()` helper and `_categoryDefaulted`/`_privacyDefaulted`
latches after a save). `resolveNetworkSaveTarget`, `resolveLocalSaveModeForRow`,
`applyNetworkEditToLocalRow`, and `_saveViaNetwork` — the phase 36 blocker-closure logic — are all
still present and call-site-unchanged in the current (uncommitted-edits-included) file. `flutter
test` was run against the actual working tree (including these uncommitted edits) and passed
946/1 skip/0 failures, confirming no regression. This is flagged for visibility, not treated as a
phase 36 gap, per this run's explicit instruction not to "fix" work outside this phase's scope.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Detail-screen badge widget test | `flutter test test/components/trail/trail_panel_sync_badge_test.dart` (independently re-run) | 5/5 pass | ✓ PASS |
| Full suite | `flutter test` (independently re-run) | 946 passed, 1 skipped, 0 failures | ✓ PASS |
| Analyzer | `flutter analyze --no-pub` (independently re-run) | 0 errors, 36 pre-existing info lints | ✓ PASS |
| Commits exist as claimed | `git log --oneline` on `trail_panel.dart` | `bdfde398`, `9419f872` present | ✓ PASS |
| No debt markers in touched files | `grep` on `trail_panel.dart` + new test file | none | ✓ PASS |
| Live device confirmation of the 36-21 fix itself | — | — | ? SKIP — see human_verification (this is the one item this pass could not close) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this project layout (Flutter/Dart). SKIPPED — no
conventional or PLAN-declared probes found.

### Human Verification Required

One item, carried in the frontmatter `human_verification`: the `<human-check>` block
`36-21-PLAN.md` deferred to end-of-phase, which `36-21-SUMMARY.md` explicitly records as "Not
performed in this session." This is the only remaining gap between "code-level and widget-test
verified" and "confirmed working on a physical device" for this phase. Everything else this
phase's prior verification pass flagged as needing a device has since been run on-device via
`36-UAT.md` round 2 (5/5 passed, including the one required judgment call).

### Gaps Summary

**No BLOCKER-level gaps remain.** All 21 plans in the phase are complete; the three prior
CRITICAL blockers (CR-01/CR-02/CR-03) remain independently confirmed closed from the last pass;
round-2 UAT ran on a physical device and passed all 5 tests, including the CR-03 UX judgment
call; the one gap that pass surfaced (the detail screen's generic Offline badge) is closed by
plan 36-21, independently re-verified in this pass by reading the current source, re-running the
new widget test (5/5 pass), re-running the full suite (946/1 skip/0 failures) and the analyzer (0
errors), and confirming the claimed commits exist in history.

The phase's one remaining risk is narrow and specific: 36-21's fix has been proven correct by
source reading and a real widget test, but — exactly like the rest of this phase's
device-dependent claims — has never been rendered on an actual phone. That is a `human_needed`
gate, not a code gap, and it is the only item left in that category for the entire 21-plan phase.

The one open WARNING from the delta review (WR-01, translation coverage for the three new
sync-status strings) is not a new gap: it is an instance of an already-tracked, already-accepted
deferral (`.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md`,
created before 36-21 ran) that already names these exact three keys at Priority 4.

---

_Verified: 2026-08-03T21:50:00Z_
_Verifier: Claude (gsd-verifier)_
