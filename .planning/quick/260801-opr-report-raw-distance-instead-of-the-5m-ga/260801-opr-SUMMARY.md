---
phase: quick-260801-opr
plan: 01
subsystem: conversion
tags: [gpx, haversine, distance, dart-port, corpus-fixtures]

requires:
  - phase: 33-conversion-correctness
    provides: the corrected GPX->trail metrics computation (CONV-01..05) this quick task partially reverts
  - phase: 34-dart-conversion-port
    provides: the Dart port and cross-language corpus (PORT-01/02) this quick task keeps in parity
provides:
  - Raw haversine distance reported on both web and app, replacing the 5 m-gated smoothed total
  - Two regenerated corpus fixtures (04-switchback-scramble, 08-jittery-track) asserting raw values
  - A new real-sampling-density corpus fixture (12-dense-switchback) guarding against a silent gate re-introduction
  - CONV-05 marked superseded in REQUIREMENTS.md (smoothed-distance half only)
affects: [gpx-conversion, trail-import, route-planner, elevation-profile-parity]

tech-stack:
  added: []
  patterns:
    - "Corpus fixtures derive every expected value independently (fresh haversine transcription), never from the implementation under test"
    - "NOT REPORTED marker comments on unreported-but-retained accumulator fields"

key-files:
  created:
    - fixtures/gpx-corpus/12-dense-switchback/input.gpx
    - fixtures/gpx-corpus/12-dense-switchback/expected.json
    - fixtures/gpx-corpus/12-dense-switchback/DERIVATION.md
  modified:
    - web/src/lib/models/gpx/gpx.ts
    - web/src/lib/models/gpx/gpx-metrics-computation.ts
    - web/src/lib/models/gpx/gpx.test.ts
    - web/src/lib/models/gpx/gpx-corpus.test.ts
    - web/src/lib/models/gpx/gpx-metrics-computation.test.ts
    - app/lib/util/gpx_conversion_util.dart
    - app/test/util/gpx_conversion_util_test.dart
    - app/test/util/gpx_corpus_test.dart
    - fixtures/gpx-corpus/04-switchback-scramble/expected.json
    - fixtures/gpx-corpus/04-switchback-scramble/DERIVATION.md
    - fixtures/gpx-corpus/08-jittery-track/expected.json
    - fixtures/gpx-corpus/08-jittery-track/DERIVATION.md
    - fixtures/gpx-corpus/README.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "Reported distance reverted from the 5 m-gated smoothed accumulator (totalDistanceSmoothed) to the raw haversine sum (totalDistance), because FIT ground truth measured raw at +0.54% vs. the gate's -3.29% against a real device's own session.total_distance"
  - "totalDistanceSmoothed retained on both classes, marked NOT REPORTED, since its lastFilteredPointXY anchor sits in a class whose other anchors are elevation-critical and a future speed-aware filter could build on it"
  - "thresholdXY_m and both GpxMetricsComputation(5, 5) call sites left untouched -- elevation output is bit-identical before and after"
  - "CONV-05 marked superseded rather than edited in place or silently flipped, scoped to the smoothed-distance half only (the cumulativeDistance rebuild still stands)"
  - "ROADMAP.md left untouched despite the plan's Task 3 instruction to append a scope-note sentence there, per this run's explicit top-level constraint that quick tasks must not update ROADMAP.md"

patterns-established:
  - "NOT REPORTED marker comments (dated, with a pointer to the superseding quick task) for accumulator fields kept alive but no longer read by any consumer"

requirements-completed: [CONV-05-REVERT, PORT-02]

duration: 9min
completed: 2026-08-01
---

# Quick Task 260801-opr Summary

**Web and app now report the raw haversine distance accumulator instead of the 5 m-gated smoothed total, closing a 3.29%-vs-0.54% ground-truth gap and adding a real-sampling-density corpus regression guard.**

## Performance

- **Duration:** 9 min (18:01:08 -> 18:06:31, commit timestamps)
- **Started:** 2026-08-01T18:01:08+02:00
- **Completed:** 2026-08-01T18:06:31+02:00
- **Tasks:** 3
- **Files modified:** 17 (3 created, 14 modified)

## Accomplishments

- `gpx.ts` and `gpx_conversion_util.dart` now report `totalDistance` (raw) instead of
  `totalDistanceSmoothed` (5 m-gated), reverting the reporting half of CONV-05. `thresholdXY_m`
  and both `GpxMetricsComputation(5, 5)` call sites are untouched -- elevation is bit-identical.
- `totalDistanceSmoothed` survives on both classes with a dated NOT REPORTED marker comment; the
  now-false D-01 comment in `gpx.ts` was rewritten to state the new same-accumulator relationship.
- Two corpus fixtures regenerated to assert raw values instead of the gated ones
  (`04-switchback-scramble` 0 -> 4.403319095226456 m; `08-jittery-track` 100.075 -> 110.083 m),
  including rewritten `notes` and `DERIVATION.md` sections.
- The `gpx.test.ts` executable invariant that used to assert `reported < raw` was inverted (not
  deleted) to assert strict equality between the reported distance and `cumulativeDistance`'s
  last entry.
- A new `12-dense-switchback` corpus fixture (41 points, ~3.852 m mean hop -- real GPS watch
  sampling density) pins the raw total at ~154.084 m and records the superseded gate's ~77.292 m
  counterfactual; both language suites' discovery floors raised from 11 to 12 fixtures.
- CONV-05 marked superseded in `REQUIREMENTS.md` (smoothed-distance half only) with the FIT
  ground-truth reason and a pointer to this quick task; the traceability status row updated.
- Confirmed (Q-06) that no Go-side code under `db/` computes or overrides `trails.distance` on a
  client save -- `db/hooks/trails.go` has zero distance references; the only Go distance producer
  is the plugin-import ingest path (`db/plugins/importer/importer.go`, `gpxData.Length2D()`),
  which is a separate path from client saves and is itself already a raw consecutive-pair sum.

## Task Commits

1. **Task 1: Report raw, and regenerate everything that asserted the gated value** - `a84e7ab9` (fix)
2. **Task 2: Add the real-sampling-density corpus fixture and raise both discovery floors** - `b704c7bf` (test)
3. **Task 3: Supersede CONV-05, retitle the class-level tests, and confirm no Go-side distance** - `961888c4` (docs)

**Plan metadata:** pending (orchestrator's docs commit)

## Files Created/Modified

- `web/src/lib/models/gpx/gpx.ts` - reports `metrics.totalDistance`; rewrote the D-01 comment
- `web/src/lib/models/gpx/gpx-metrics-computation.ts` - NOT REPORTED marker on `totalDistanceSmoothed`
- `web/src/lib/models/gpx/gpx.test.ts` - retitled CONV-05 describe/tests, inverted the D-01 invariant
- `web/src/lib/models/gpx/gpx-corpus.test.ts` - discovery floor raised to 12
- `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` - retitled class-level tests, no numbers changed
- `app/lib/util/gpx_conversion_util.dart` - reports `metrics.totalDistance`; NOT REPORTED marker
- `app/test/util/gpx_conversion_util_test.dart` - CONV-04 distance assertion re-pointed to `closeTo(4.403319)`; CONV-05 test retitled
- `app/test/util/gpx_corpus_test.dart` - discovery floor raised to 12
- `fixtures/gpx-corpus/04-switchback-scramble/expected.json` + `DERIVATION.md` - regenerated for raw distance
- `fixtures/gpx-corpus/08-jittery-track/expected.json` + `DERIVATION.md` - regenerated for raw distance
- `fixtures/gpx-corpus/12-dense-switchback/input.gpx` + `expected.json` + `DERIVATION.md` - new fixture (created)
- `fixtures/gpx-corpus/README.md` - `distance` documented as raw; new fixture's coverage row added
- `.planning/REQUIREMENTS.md` - CONV-05 marked superseded (both the requirement line and traceability row)

## Decisions Made

- Reverted the reporting half of CONV-05 (raw over smoothed) on FIT ground-truth evidence: a real
  device's own `session.total_distance` (10912.01 m) sits +0.54% from raw and -3.29% from the
  smoothed/gated total -- 6x worse, and the gate chord-shortcuts switchback geometry at real
  sampling density rather than removing genuine noise.
- Kept `totalDistanceSmoothed` and its `lastFilteredPointXY` anchor alive, unreported, rather than
  deleting them -- they sit in a class whose other anchors are elevation-critical, and a future
  speed-aware filter could build on the smoothed accumulator.
- CONV-05 marked superseded rather than edited in place, scoped narrowly to the smoothed-distance
  half; its second clause (the `cumulativeDistance` rebuild) still stands and is unaffected.
- Per this run's explicit top-level constraint ("Do NOT update ROADMAP.md — quick tasks are
  separate from planned phases"), the plan's Task 3 instruction to append a supersession sentence
  to ROADMAP.md's Phase 33 scope note was **not** applied -- see Deviations below.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 4 override — orchestrator constraint takes precedence] ROADMAP.md left unmodified**
- **Found during:** Task 3
- **Issue:** The plan's Task 3 action explicitly instructed appending a supersession sentence to
  `.planning/ROADMAP.md`'s Phase 33 scope note. This run's top-level constraints explicitly state
  "Do NOT update ROADMAP.md (quick tasks are separate from planned phases)."
- **Resolution:** Made the edit initially (matching the plan), then reverted it via
  `git checkout -- .planning/ROADMAP.md` once the conflict was identified, since an explicit
  orchestrator-level directive takes precedence over a plan instruction. `.planning/REQUIREMENTS.md`
  — not restricted by the same constraint — still carries the full supersession record (reason,
  scope, pointer to this quick task), so the traceability chain (`covers: ["CONV-05"]` in the
  corpus fixtures -> REQUIREMENTS.md) remains intact without touching ROADMAP.md.
- **Files affected:** `.planning/ROADMAP.md` (reverted to its pre-task state, not committed)
- **Committed in:** N/A — deliberately not committed

**2. [Scope boundary — out of scope, logged not fixed] Pre-existing dart format churn in `gpx_corpus_test.dart`**
- **Found during:** Task 3's `dart format --set-exit-if-changed` verification pass
- **Issue:** `dart format` reports `test/util/gpx_corpus_test.dart` as needing reformatting
  (long `test('...', () { ... })` closures collapsed onto fewer physical lines than the formatter
  now prefers). Verified this predates this quick task's Task 2 edit (checked against the
  pre-task version of the file, which already fails `dart format --set-exit-if-changed`).
- **Resolution:** Not fixed — none of the lines involved were touched by Task 2's edit (which
  only changed the doc comment and the `< 11` -> `< 12` floor check), and reformatting the whole
  file would touch far more than this quick task's scope. Logged here per the scope-boundary rule
  rather than fixed.
- **Files affected:** `app/test/util/gpx_corpus_test.dart` (not modified for this)
- **Committed in:** N/A — deferred, out of scope

---

**Total deviations:** 2 (1 orchestrator-constraint override, 1 out-of-scope pre-existing issue logged)
**Impact on plan:** No change to test behavior or correctness. REQUIREMENTS.md still carries the
full supersession record even without the ROADMAP.md addition. The dart format issue is
pre-existing and unrelated to any line this quick task touched.

## Issues Encountered

None beyond the deviations above. All automated verification gates in the plan passed on first
run; no auto-fix attempts were needed on any task.

## Verification Results (actual output)

**Web GPX suites** (`cd web && npx vitest run src/lib/models/gpx/`):
```
Test Files  4 passed (4)
     Tests  84 passed (84)
```

**Dart GPX suites** (`cd app && flutter test test/util/gpx_conversion_util_test.dart test/util/gpx_corpus_test.dart`):
```
00:01 +136: All tests passed!
```
(136 total; no pre-existing unrelated failures in these two files. The three pre-existing failures
noted in the plan — `feed_item_test.dart` x2, `settings_screen_test.dart` x1 — live in different
files, were not run here, and are unrelated to this change.)

**Static grep gates:** `REPORT_SWAP_OK`, `RAW_REPORTED_OK`, `THRESHOLD_AND_FIELD_INTACT`,
`FIXTURE_PRESENT`, `POINT_COUNT_OK`, `FLOORS_RAISED`, `SUPERSEDED_RECORDED`,
`COVERS_STILL_RESOLVES`, `NO_GO_SAVE_HOOK_DISTANCE` — all printed as expected.

**`flutter analyze lib/util/gpx_conversion_util.dart`:** No issues found.

**`dart format` on files this task actually edited** (`gpx_conversion_util.dart`,
`gpx_conversion_util_test.dart`): 0 changed — clean.

**Forbidden-files guard:** `git status --porcelain` on all five user-owned unrelated files
(`track_save_options_sheet.dart`, `trail_source_select_screen.dart`,
`track_save_options_util.dart`, `gpx_util.ts`, `trail/convert/+server.ts`) shows all five still
` M` (modified, unstaged) after every commit — none were staged or touched.

**Corpus parity:** 12 fixtures discovered and pass in both languages (36 Vitest tests, 36 Dart
tests across metrics/trail-assembly/integrity describes). `git status --porcelain fixtures/`
confirms only `04-switchback-scramble/`, `08-jittery-track/`, `12-dense-switchback/`, and
`README.md` changed under `fixtures/gpx-corpus/` — the other eight fixtures are byte-identical.

## Manual Verification Still Needed (not run in this session)

Task 3's `<human-check>` block requires physically importing a real FIT file
(`~/Downloads/19440058502_ACTIVITY.fit`, confirmed present on disk) into a running web dev server
and the Flutter app, and visually confirming:

1. Web: the trail summary header and the elevation-profile tooltip both read **10.97 km** (device
   truth 10912.01 m; previously the summary read 10.55 km).
2. App: the create screen, then the saved trail's stat chip and chart header, all read **10.97 km**.
3. Elevation stays exactly **344 m up / 351 m down** on both platforms, before and after.
4. A previously-saved trail keeps its stored distance until re-saved (no backfill).

This requires a running dev server/app instance and manual visual confirmation and was not
performed as part of this automated execution session — the code-level changes and all automated
test suites confirm the underlying computation is correct (raw accumulator, bit-identical
elevation), but the end-to-end device-truth comparison against the real FIT file is a manual step
for the user to perform.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The reporting-side revert of CONV-05 is complete and test-covered on both platforms; the
  corpus now has a real-sampling-density guard (`12-dense-switchback`) making a silent
  re-introduction of the 5 m gate impossible to miss.
- REQUIREMENTS.md's CONV-05 entry accurately reflects the superseded state; no other requirement
  or phase's completion status was touched.
- Outstanding: the manual end-to-end FIT-file verification described above, and the pre-existing
  `gpx_corpus_test.dart` dart-format churn (out of scope, logged, not blocking).

---
*Phase: quick-260801-opr*
*Completed: 2026-08-01*

## Self-Check: PASSED

All 17 files listed in `key-files` (created + modified) verified present on disk via direct
`[ -f ... ]` checks. All three task commit hashes (`a84e7ab9`, `b704c7bf`, `961888c4`) verified
present in `git log --oneline --all`. No missing items.
