---
phase: quick-260719-n8g
verified: 2026-07-19T18:00:00Z
status: gaps_found
score: 4/5 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Pausing a recording actually pauses GPS breadcrumb capture (not just the displayed timer/stats)"
    status: failed
    reason: "NavigationStatsNotifier.onPosition correctly early-returns while isPaused/isStationary, but the separate Navigation notifier that owns `breadcrumb` (the data persisted to disk and later exported as the saved trail's GPX via buildGpxFromPoints) has no pause/stationary gating at all. The single position-stream listener in NavigationScreen.initState calls both notifiers unconditionally on every GPS fix, so breadcrumb points keep accumulating while the UI shows a frozen stopwatch/pause icon. A user who pauses at a rest stop, gets in a car, etc. will get those segments silently baked into the saved trail — directly undermining the pause button's implied contract for a route recorder."
    artifacts:
      - path: "app/lib/routes/navigation_screen.dart"
        issue: "Lines 343-374: position-stream listener calls navigationProvider(...).notifier.onPosition(pos) unconditionally, with no isPaused/isStationary check before the call (unlike the adjacent navigationStatsProvider(...).notifier.onPosition(pos) call, which internally gates on that state)."
      - path: "app/lib/provider/navigation_provider.dart"
        issue: "Navigation.onPosition (line ~114) unconditionally appends to breadcrumb with zero awareness of pause/stationary state — no such flag is threaded into this provider at all."
    missing:
      - "Gate the breadcrumb-affecting onPosition call in NavigationScreen's position-stream listener on the same isPaused || isStationary condition NavigationStatsNotifier.onPosition already uses (read via navigationStatsProvider before deciding whether to feed the fix into navigationProvider), OR thread isPaused/isStationary into Navigation.onPosition itself so the provider that owns breadcrumb/GPX-export semantics is self-consistent with the stats provider."
---

# Quick Task 260719-n8g: Implement the missing route recorder — Verification Report

**Task Goal:** Implement the missing route recorder. Most of the pieces are already in place as route recording is essentially navigating but without an original trail. Reuse as many components/screens as possible. After a user finishes recording a trail they can save them via the trail_create_screen as usual.

**Verified:** 2026-07-19T18:00:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Tapping "Record trail" card requests location permission then opens a live GPS recording session (map + stats, no maneuver banner) | ✓ VERIFIED | `trail_source_select_screen.dart:37-75` `_openRecorder` mirrors the `navigation_launch_util.dart` permission gate (service-enabled → checkPermission → requestPermission → iOS re-prompt) then `context.push('/record')`. Router builds `NavigationScreen(id: '', response: const NavigateResponse(maneuvers: [], shape: []), isRecording: true, ...)` (`router_provider.dart:276-292`). `_buildActiveBannerContent` returns `SizedBox.shrink()` when `maneuvers.isEmpty` (`navigation_screen.dart:1243-1245`), so no maneuver banner renders. |
| 2 | Bottom button row is [pause, stop, elevation] in recording mode; pause toggle freezes/resumes the timer and stats | ✓ VERIFIED | `_buildButtonRow` branches on `widget.isRecording` (`navigation_screen.dart:1645-1671`): left `_buildPauseFab` (calls `togglePause()` + `_persistNow()`), center red `FloatingActionButton` (`FontAwesomeIcons.stop`, `colorScheme.error`, `heroTag: 'rec_stop'`) calling `_confirmExit`, right `_buildElevationFab`. `NavigationStatsNotifier.togglePause`/`onPosition` (`navigation_stats_provider.dart:175-177, 234, 248`) correctly freezes elapsed/distance/elevation while `isPaused || isStationary`. |
| 3 | Red Stop button opens 3-option dialog (Cancel / Exit without saving / Save); Save hands recorded breadcrumb to trail_create_screen via buildDraftTrail | ✓ VERIFIED | `_confirmExit` (`navigation_screen.dart:1137-1179`) branches dialog content on `widget.isRecording` (`stop_recording_confirm` vs `stop_navigation_confirm`), keeps the same 3-choice `.then` switch. `_saveRecordedTrack` (`:667-709`) builds GPX from `navState.breadcrumb`, calls `buildDraftTrail(ref, gpx, category: originalTrail?.categoryId)`, then `context.pushReplacement('/trail/create/edit', extra: trail)`. |
| 4 | Recording session persists as `ActiveSessionType.rec` row (`trailId` null); app-kill relaunch prompts resume | ✓ VERIFIED | `_persistNow` (`navigation_screen.dart:599-604`): `sessionType: widget.isRecording ? ActiveSessionType.rec : ActiveSessionType.nav`, `trailId: widget.isRecording ? null : widget.id`. `main.dart`'s `_maybeResume` (`:177-180`) branches to `_maybeResumeRecording` (`:244-272`) for `ActiveSessionType.rec` rows, shows `resume_recording_prompt` dialog, re-pushes `/record` with the row as `extra` on accept. |
| 5 (derived) | Pausing a recording actually pauses GPS breadcrumb capture, not just the displayed stats | ✗ FAILED | See gap above — `NavigationStatsNotifier.onPosition` gates on `isPaused`/`isStationary`, but `Navigation.onPosition` (owner of `breadcrumb`, the data exported to the saved GPX) has no such gate and the position-stream listener in `NavigationScreen.initState` (`:343-374`) calls it unconditionally. Confirmed independently in code (not just SUMMARY/REVIEW claims): `navigation_provider.dart`'s `onPosition` unconditionally does `breadcrumb: [...state.breadcrumb, Wpt(...)]` with no pause parameter anywhere in that file. |

**Score:** 4/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `app/lib/routes/navigation_screen.dart` | `isRecording` flag + branched button row, exit-dialog content, `_persistNow` session type, elevation-page guard | ✓ VERIFIED | All four branch points present and match plan spec (constructor `:51-71`, button row `:1640-1702`, `_confirmExit` `:1137-1179`, `_persistNow` `:584-632`, elevation guard `:1526-1532`). |
| `app/lib/provider/router_provider.dart` | Top-level `/record` GoRoute building `NavigationScreen` in recording mode | ✓ VERIFIED | `:276-292`, reads `state.extra as ActiveNavigationEntity?` resume seed, builds with `isRecording: true`, `id: ''`, empty `NavigateResponse`. |
| `app/lib/routes/trail_source_select_screen.dart` | Record card wired to permission gate + `context.push('/record')`; dead `_comingSoon` removed | ✓ VERIFIED | `_openRecorder` present (`:37-75`), card `onTap` wired (`:182-184`). `grep` confirms zero remaining references to `_comingSoon`/`comingSoon` anywhere in `lib/`. |
| `app/lib/main.dart` | `ActiveSessionType.rec` branch in `_maybeResume` | ✓ VERIFIED | `:177-180` branch + `_maybeResumeRecording` helper (`:244-272`). |
| `app/lib/i18n/app_en.arb` | `stop_recording`, `stop_recording_confirm`, `resume_recording_prompt` keys | ✓ VERIFIED | Present in `app_en.arb` and confirmed present (3/3 hits) in all 14 `app_*.arb` locale files. `app_localizations.dart` regenerated with matching getters. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `trail_source_select_screen.dart` | `/record` route | `context.push('/record')` after permission gate | ✓ WIRED | `grep` hit at `:74`, inside `_openRecorder` after the permission checks. |
| `router_provider.dart` | `NavigationScreen` recording mode | GoRoute builder, `isRecording: true` | ✓ WIRED | `grep` hit at `:288`. |
| `navigation_screen.dart` | `ActiveNavigationEntity` persistence | `_persistNow` session-type branch | ✓ WIRED | `grep` hit at `:602`, ternary sets `ActiveSessionType.rec`. |
| `main.dart` | `/record` resume | `_maybeResume` rec branch push | ✓ WIRED | `grep` hit at `:267`, `push('/record', extra: row)`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---|---|---|---|---|
| `navigation_screen.dart` breadcrumb rendering (`_breadcrumbGeoJson`) | `navState.breadcrumb` | `navigationProvider(...)`, fed by real `TraceletPositionSource` GPS stream via `onPosition` | Yes — real GPS fixes | ✓ FLOWING (but see gap: flows even while paused, which is a correctness bug, not a disconnection) |
| `navigation_screen.dart` stats sheet | `stats` (time/distance/elevation/speed) | `navigationStatsProvider(...)`, correctly gated on pause/stationary | Yes | ✓ FLOWING |
| `_saveRecordedTrack` → `trail_create_screen` handoff | `trail` (stub Trail) | `buildDraftTrail(ref, gpx, ...)` built from real breadcrumb via `/trail/convert` round-trip | Yes | ✓ FLOWING |

### Behavioral Spot-Checks / Automated Verification (run live by verifier, not taken from SUMMARY)

| Behavior | Command | Result | Status |
|---|---|---|---|
| `flutter gen-l10n` produces zero untranslated-message warnings for the 3 new keys | `cd app && flutter gen-l10n` (+ temporary `untranslated-messages-file` probe, reverted after) | Overall project has 26–105 untranslated messages per locale (pre-existing debt — confirmed only `trail_source_record`, an older key, appears among them); none of `stop_recording`, `stop_recording_confirm`, `resume_recording_prompt` appear in the untranslated list for any locale | ✓ PASS (for the 3 keys in scope) |
| Provider test suite passes, including new recording-mode case | `cd app && flutter test test/provider/navigation_provider_test.dart` | `+11: All tests passed!` including `NavigationNotifier recording mode (empty response) empty NavigateResponse resolves to an empty state without throwing, then a fed position still appends to the breadcrumb` | ✓ PASS |
| `flutter analyze` clean on touched files (no `unused_element`) | `cd app && flutter analyze lib/routes/navigation_screen.dart lib/i18n/app_localizations.dart lib/routes/trail_source_select_screen.dart lib/provider/router_provider.dart lib/main.dart` | 2 `info`-level `use_build_context_synchronously` hints at `navigation_screen.dart:693,703` — confirmed via `git blame` these lines belong to commit `c521149e` (a prior, unrelated phase), not this task's commits (`20316c47`, `c41b757d`). No `unused_element`, no new issues introduced by this task. | ✓ PASS (no regressions from this task) |
| `_comingSoon` fully removed, no dead references | `grep -rn "_comingSoon\|comingSoon" app/lib/` | No hits | ✓ PASS |

### Code Review Findings (260719-n8g-REVIEW.md) — not resolved by any subsequent commit

A prior code-review pass (`260719-n8g-REVIEW.md`, `status: issues_found`, 1 critical / 6 warnings / 1 info) flagged a critical correctness bug. `git log` confirms no commit after the review (`4075d625` is a docs-only commit) addresses it — it is carried forward into this verification as the FAILED truth above (Truth #5).

Additional unresolved warnings (not blocking the phase goal, but worth tracking):
- WR-01: `_openRecorder` missing `mounted` guards after each `await` (sibling flows `_openPlanner`/`_importGpx` have them).
- WR-02: `_openRecorder` has no re-entrancy/loading guard — rapid double-tap could push `/record` twice concurrently.
- WR-03: Card descriptions on `trail_source_select_screen.dart` bypass i18n (hardcoded English) while titles use it.
- WR-04: `_persistNow` collapses null elevation to `0.0`, losing the null/missing distinction on resume.
- WR-05: `_saveRecordedTrack` clears `active_nav` / sets `pendingImportedTrail` before confirming `context.mounted`.
- WR-06: `_saveRecordedTrack`'s catch block swallows errors silently (no `debugPrint`, unlike every other catch in the file).
- IN-01: Commented-out debug line in `main.dart:37`.

These are pre-existing/adjacent quality issues; they do not independently fail a must-have truth from the plan and are reported as warnings, not blockers.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| RECORD-MODE | 260719-n8g-PLAN.md | Reuse NavigationScreen as empty-response recording mode | ✓ SATISFIED (with caveat) | `isRecording` flag threaded through 4 branch points; core UI/flow works, but breadcrumb-pause gap (Truth #5) means the "recording mode" doesn't fully honor pause semantics. |
| RECORD-ENTRY | 260719-n8g-PLAN.md | Wire "Record trail" card + /record route | ✓ SATISFIED | Entry card + route + resume-seed handling all verified. |
| RECORD-RESUME | 260719-n8g-PLAN.md | Resume ActiveSessionType.rec after app kill | ✓ SATISFIED | `_maybeResumeRecording` verified end-to-end. |

(This is a quick task; these requirement IDs are local to the PLAN.md frontmatter and are not tracked in `.planning/REQUIREMENTS.md` — no orphaned-requirement check applies.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `app/lib/routes/navigation_screen.dart` | 343-374 | Missing pause/stationary gate on breadcrumb-affecting `onPosition` call | 🛑 Blocker | Recorded track silently includes "paused" segments — undermines pause button's contract for a route recorder (Truth #5). |
| `app/lib/routes/trail_source_select_screen.dart` | 37-75 | Missing `mounted` guards after `await` in `_openRecorder` | ⚠️ Warning | Potential use-after-dispose if user navigates away mid permission-flow. |
| `app/lib/routes/trail_source_select_screen.dart` | 177-185 | No re-entrancy guard on Record card tap | ⚠️ Warning | Rapid double-tap could concurrently push `/record` twice. |
| `app/lib/main.dart` | 37 | Commented-out debug line | ℹ️ Info | Leftover local-debug artifact, harmless but should be removed or gated. |

No `TODO`/`FIXME`/`XXX`/`TBD` debt markers found in any file modified by this task.

### Human Verification Required

The plan's own verification section states widget tests are impractical for `NavigationScreen` (native MapLibre/tracelet/sensor dependencies) and specifies manual on-device verification. These are still outstanding regardless of the automated checks above:

#### 1. End-to-end recording flow

**Test:** Trail source → tap "Record trail" → grant permission → observe recording session opens (map centers on first fix, no maneuver banner, bottom row [pause, stop, elevation]).
**Expected:** Session opens cleanly with the described layout.
**Why human:** Requires a live device with GPS/MapLibre rendering — cannot be verified via static analysis or unit tests.

#### 2. Pause/resume visual + GPS-gating behavior

**Test:** Walk a few meters, confirm stats/timer advance; tap pause, confirm timer freezes; tap again, confirm resumes. Additionally: after fixing the CR-01 gap, confirm the saved GPX excludes points captured while paused.
**Expected:** Timer/stats freeze and resume correctly; saved track excludes paused-segment points once the gap is fixed.
**Why human:** Requires live GPS motion and a saved-track GPX inspection — not verifiable statically.

#### 3. Stop dialog and save handoff

**Test:** Tap red Stop button; confirm dialog reads "Stop recording?" with Cancel / Exit without saving / Save; tap Save; confirm handoff to trail_create_screen with the recorded track prefilled.
**Expected:** Dialog and handoff work as described.
**Why human:** UI flow correctness and visual confirmation of the prefilled track require manual interaction.

#### 4. Resume-after-kill flow

**Test:** Start a recording, swipe-kill the app, relaunch; confirm "Resume recording?" prompt appears; accept and confirm breadcrumb + stats continue; on a separate run, decline and confirm no prompt reappears on next launch.
**Expected:** Resume dialog appears once per killed session; accept continues the session; decline clears it.
**Why human:** Requires actual process kill/relaunch on a device/emulator — not testable via static code inspection.

### Gaps Summary

The recorder's core wiring — entry point, `/record` route, reused `NavigationScreen` in recording mode, 3-option stop dialog with save handoff, and `ActiveSessionType.rec` resume-after-kill — is all genuinely implemented and correctly wired (verified directly in code, not from SUMMARY narrative alone; all `flutter gen-l10n`/`flutter test`/`flutter analyze` checks were re-run live by this verifier and pass for the scope of this task).

However, a real functional defect survives from the phase's own code review (`260719-n8g-REVIEW.md`, CR-01, unresolved) and was independently re-confirmed here by reading `navigation_screen.dart` and `navigation_provider.dart`: **the pause button freezes the displayed stats but does not stop GPS points from being appended to the breadcrumb that gets persisted and eventually exported as the saved trail's GPX.** This means a user who pauses recording (e.g., at a rest stop, getting into a car) will unknowingly get that segment baked into their saved route — a meaningful correctness gap for a "route recorder" feature, not just a cosmetic issue. This is the reason for `status: gaps_found` rather than `passed`.

**This looks unintentional** (not a deliberate scope decision) — the code review already flagged it as the single critical issue and it was never addressed. No override is suggested; recommend closing this gap via `/gsd-plan-phase --gaps` or an equivalent quick-task follow-up before treating recording pause as reliable.

---

_Verified: 2026-07-19T18:00:00Z_
_Verifier: Claude (gsd-verifier)_
