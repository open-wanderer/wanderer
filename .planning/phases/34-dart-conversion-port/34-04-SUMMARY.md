---
phase: 34-dart-conversion-port
plan: 04
subsystem: testing
tags: [gpx, dart, flutter, corpus, cross-language-parity, dart-port, freezed]

# Dependency graph
requires:
  - phase: 34-dart-conversion-port
    plan: 01
    provides: computeTrailMetrics/GpxTrailMetrics/parseGpxSafely in gpx_conversion_util.dart
  - phase: 34-dart-conversion-port
    plan: 03
    provides: "fixtures/gpx-corpus/ - the on-disk, language-neutral fixture corpus this plan's Dart suite reads"
provides:
  - "trailFromGpx(Gpx) in gpx_conversion_util.dart - the Dart port of gpx2trail's trail assembly (PORT-01: name, description, waypoints, start coordinates, date, distance, elevation gain/loss, duration, bounding box, no network call)"
  - "app/test/util/gpx_corpus_test.dart - the Dart side of the PORT-02 cross-language parity contract, 30 tests against fixtures/gpx-corpus/"
  - "trailDisplayDuration(TrailSummary) in format_util.dart - D-10's single app-side duration display rule"
  - "GpxMappingUtils.getTotals()/GpxStats deleted from gpx_util.dart (D-17) - the app now has exactly one GPX metrics implementation"
affects: [34-05, 34-06, 34-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TrailSummary.movingDuration as an abstract getter (no default body): `implements` does not inherit a default implementation in Dart, so every TrailSummary implementer must declare its own override explicitly"

key-files:
  created:
    - app/test/util/gpx_corpus_test.dart
  modified:
    - app/lib/util/gpx_conversion_util.dart
    - app/test/util/gpx_conversion_util_test.dart
    - app/lib/util/gpx_util.dart
    - app/lib/util/format_util.dart
    - app/test/util/format_util_test.dart
    - app/lib/components/trail/elevation_profile.dart
    - app/lib/components/trail/trail_panel.dart
    - app/lib/components/trail/trail_card.dart
    - app/lib/components/trail/trail_list_item.dart
    - app/lib/models/trail_summary.dart
    - app/lib/models/global_search_models.dart

key-decisions:
  - "trailDisplayDuration is typed on TrailSummary, not the plan's literal Trail: trail_card.dart/trail_list_item.dart hold a TrailSummary (which may be a search-result summary with no moving-time concept at runtime), while trail_panel.dart holds a Trail (a TrailSummary subtype) - typing on Trail would not compile at two of the three mandated call sites"
  - "TrailSummary.movingDuration is declared as a plain abstract getter (double? get movingDuration;), not a default-bodied one: Dart's `implements` clause uses only the interface, never the implementation, so a default body would be silently ignored by every implementer and give a false sense of an inherited fallback. TrailSearchResult (global_search_models.dart) now has its own explicit `=> null` override; Trail's existing freezed field already satisfies the interface"
  - "The gpx_util.dart deletion comment avoids the literal substring \"getTotals\" (writes \"a totals-style method\" instead) to satisfy the plan's own grep -c getTotals == 0 acceptance gate, mirroring 34-03's precedent for literal-substring collisions between prose and grep gates"

patterns-established:
  - "_expectClose(actual, expected, tol, label) in gpx_corpus_test.dart - the Dart-side single implementation of the corpus's per-field absolute-tolerance comparison, mirroring the TS suite's expectCorpusMatch()"
  - "Fixture-count and CWD guards in Dart test suites that load external fixtures use a plain thrown StateError rather than package:test's expect() when the check runs at main()-body time (outside any test() closure) - expect() throws OutsideTestException if called before a test starts"

requirements-completed: [PORT-01, PORT-02, CONV-06]

# Metrics
duration: 95min
completed: 2026-08-01
---

# Phase 34 Plan 04: Dart trailFromGpx Port + Corpus Proof + Second-Metrics Deletion Summary

**Dart gains `trailFromGpx` (the gpx2trail port) proven against the same on-disk fixture corpus the TS suite reads, while the app's second, CONV-01-buggy metrics implementation (`GpxMappingUtils.getTotals()`/`GpxStats`) is deleted and both its consumers redirected onto the ported `computeTrailMetrics`.**

## Performance

- **Duration:** ~95 min
- **Tasks:** 3/3 completed
- **Files modified:** 12 (1 new test file, 11 modified)

## Accomplishments

- `trailFromGpx(Gpx gpx, {String? fallbackName, Duration? movingDuration, String? gpxData})` added to
  `gpx_conversion_util.dart`: a pure, offline (D-14), line-for-line port of `gpx2trail`
  (`web/src/lib/util/gpx_util.ts:37-89`) producing a complete draft `Trail` - name (with the TS `||`
  empty-string-fallthrough semantics replicated via explicit `isNotEmpty` checks, not `??`),
  description, waypoints (icon resolved through the closed-set `fontAwesomeIconsMap` lookup, T-34-18),
  start coordinates, date (UTC calendar date, only when both first and last trkpt of the first segment
  carry a time), distance/elevation/duration from `computeTrailMetrics`, and a bounding box guarded by
  a finiteness check (D-13's `movingDuration` override never feeds `duration`)
- 13 new `trailFromGpx` tests appended to `gpx_conversion_util_test.dart` covering the name fallback
  chain, description default, lat/lon, the date guard, duration, the D-13 override, waypoint icon
  mapping, and the bounding-box guard for a trackless GPX
- `app/test/util/gpx_corpus_test.dart` (new, 30 tests): the Dart side of the PORT-02 cross-language
  contract, reading `fixtures/gpx-corpus/` from disk via the same relative path convention as the TS
  suite. Three groups mirror the TS suite exactly - metrics (`computeTrailMetrics`), trail assembly
  (`trailFromGpx`), and integrity (`expected.json` id/DERIVATION.md checks) - with tolerances declared
  exactly once (`kMetreTolerance = 1e-6`, `kDegreeTolerance = 1e-9`) and a loud `StateError` (not
  `expect()`, which cannot run outside a `test()` body) if the corpus resolves to fewer than 10
  fixtures or the wrong CWD
- D-17: `GpxMappingUtils.getTotals()` and `class GpxStats` deleted from `gpx_util.dart` - the second,
  CONV-01-buggy metrics implementation that let an unsaved-GPX preview disagree with a saved trail's
  numbers for the same GPX (T-34-19). `elevation_profile.dart`'s and `trail_panel.dart`'s preview
  branches now both call `computeTrailMetrics`
- `trailDisplayDuration(TrailSummary trail)` added to `format_util.dart` - D-10's single app-side
  duration display rule (show `movingDuration` when present and > 0, else `duration`), applied at all
  three mandated sites: `trail_panel.dart`, `trail_card.dart`, `trail_list_item.dart`
- `TrailSummary.movingDuration` added as a required abstract getter (no default body - `implements`
  in Dart does not inherit one), with `TrailSearchResult` (the only other `TrailSummary` implementer)
  explicitly overriding it to `null` (search results have no moving-time concept)

## Task Commits

1. **Task 1: Port gpx2trail's trail assembly as trailFromGpx** - `2fa8f91c` (feat)
2. **Task 2: Prove the Dart implementation against the shared corpus** - `96bd1506` (test)
3. **Task 3: Delete the second metrics implementation, redirect its consumers, and apply the app display rule** - `b336e76e` (feat)

## Files Created/Modified

- `app/lib/util/gpx_conversion_util.dart` - adds `trailFromGpx` (Task 1's implementation was
  pre-existing, authored by the user before this execution session; verified against the plan's
  acceptance criteria and committed as-is with no code changes needed)
- `app/test/util/gpx_conversion_util_test.dart` - `trailFromGpx` test group (pre-existing, same as above)
- `app/test/util/gpx_corpus_test.dart` - new: the Dart cross-language parity suite
- `app/lib/util/gpx_util.dart` - `GpxStats`/`getTotals()` deleted; a doc comment on the
  `GpxMappingUtils` extension forbids re-adding any metrics computation
- `app/lib/util/format_util.dart` - `trailDisplayDuration(TrailSummary)`
- `app/test/util/format_util_test.dart` - `trailDisplayDuration` test group (4 cases)
- `app/lib/components/trail/elevation_profile.dart` - preview branch redirected to `computeTrailMetrics`
- `app/lib/components/trail/trail_panel.dart` - preview branch and duration display redirected
- `app/lib/components/trail/trail_card.dart`, `trail_list_item.dart` - duration display via
  `trailDisplayDuration`
- `app/lib/models/trail_summary.dart` - adds the `movingDuration` abstract getter
- `app/lib/models/global_search_models.dart` - `TrailSearchResult`'s `movingDuration => null` override

## Decisions Made

- **`trailDisplayDuration` typed on `TrailSummary`, not the plan's literal `Trail`.** The plan's
  Artifacts table specifies `trailDisplayDuration(Trail trail) -> double?`, but `trail_card.dart` and
  `trail_list_item.dart`'s `trail` field is declared `TrailSummary` (it may hold a `TrailSearchResult`
  at runtime, which is not a `Trail`). A `Trail`-typed parameter would not compile against those two of
  the plan's three mandated call sites. `TrailSummary` is the correct, minimal supertype: `Trail`
  already `implements TrailSummary`, so `trail_panel.dart`'s `Trail`-typed field passes through
  unchanged. Confirmed the plan's grep gate (`grep -c "double? trailDisplayDuration"`) still matches
  regardless of parameter type, so this does not violate any literal acceptance criterion.
- **`TrailSummary.movingDuration` is a plain abstract getter, not a default-bodied one.** An initial
  attempt gave it `=> null` directly on the interface, expecting every `implements TrailSummary` class
  to inherit that default. `flutter analyze` immediately surfaced
  `Missing concrete implementation of 'getter TrailSummary.movingDuration'` on the generated
  `_TrailSearchResult` class - Dart's `implements` clause uses only the interface signature, never the
  implementation, so a default body on an `implements`-only interface is dead code that silently does
  nothing for implementers. Fixed by making the getter purely abstract and adding an explicit
  `@override double? get movingDuration => null;` to `TrailSearchResult` (the app's only other
  `TrailSummary` implementer besides `Trail`, which already satisfies the interface via its existing
  freezed field).
- **The gpx_util.dart deletion comment avoids the literal string "getTotals".** The plan's own
  acceptance criteria requires `grep -c "getTotals" app/lib/util/gpx_util.dart` to return exactly 0.
  An initial doc comment explaining the deletion used the word "getTotals()" in prose and tripped that
  same gate; reworded to "a totals-style method" instead, following the identical pattern 34-03's
  SUMMARY documented for its own literal-substring grep-gate collisions.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `_loadFixtures()`'s fixture-count guard used `expect()` outside a test body**
- **Found during:** Task 2, first `flutter test` run
- **Issue:** `expect(fixtures.length, greaterThanOrEqualTo(10))` was called directly inside
  `_loadFixtures()`, which runs from `main()`'s top level before any `test()` closure starts.
  `package:flutter_test`'s `expect()` throws `OutsideTestException` in that context, failing the whole
  suite at load time with a confusing error rather than the intended "corpus shrank" message.
- **Fix:** Replaced the `expect()` call with a plain `if (fixtures.length < 10) throw StateError(...)`,
  matching the existing `existsSync` guard's loud-failure style (which was already a thrown
  `StateError`, not an `expect()`, for the same reason).
- **Files modified:** `app/test/util/gpx_corpus_test.dart`
- **Verification:** `flutter test test/util/gpx_corpus_test.dart` - 30/30 pass
- **Committed in:** `96bd1506` (Task 2 commit)

**2. [Rule 1 - Bug] `TrailSummary.movingDuration`'s default body was silently unusable**
- **Found during:** Task 3, `flutter analyze` after adding the getter
- **Issue:** Declared `double? get movingDuration => null;` directly on the `TrailSummary` abstract
  class, assuming `implements` inherits default bodies the way `extends`/`with` does. It does not -
  `flutter analyze` reported a missing-override error on the generated `_TrailSearchResult` class.
- **Fix:** Made the getter purely abstract on `TrailSummary` and added an explicit
  `@override double? get movingDuration => null;` on `TrailSearchResult`.
- **Files modified:** `app/lib/models/trail_summary.dart`, `app/lib/models/global_search_models.dart`
- **Verification:** `flutter analyze` - no errors (33 pre-existing info-level lints unrelated to this
  plan remain, e.g. deprecated FontAwesome icon aliases in `icon_util.dart`)
- **Committed in:** `b336e76e` (Task 3 commit)

**3. [Rule 2 - Missing coverage / grep-gate collision] `gpx_util.dart`'s D-17 comment tripped its own
   literal `getTotals` grep gate**
- **Found during:** Task 3's own acceptance-criteria verification
- **Issue:** The doc comment explaining why no metrics implementation may be re-added to
  `GpxMappingUtils` used the literal substring `getTotals()` in prose, which the plan's own
  `grep -c "getTotals" app/lib/util/gpx_util.dart` (must return 0) gate matched.
- **Fix:** Reworded to "a totals-style method" - same explanatory content, no literal match.
- **Files modified:** `app/lib/util/gpx_util.dart`
- **Verification:** `grep -c "getTotals" app/lib/util/gpx_util.dart` returns 0
- **Committed in:** `b336e76e` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 1, 1 Rule 2)
**Impact on plan:** All three were necessary for correctness (a test-framework misuse, a type-system
misunderstanding of `implements` semantics) or to satisfy the plan's own literal acceptance-criteria
grep gates. No scope creep - no file outside `files_modified` was touched.

## Known Acceptance-Criteria Grep-Gate False Positive (documented, not fixed)

The plan's Task 3 acceptance criteria require
`grep -c "totalElevationloss\|totalElevationGain\|totalDistance" app/lib/components/trail/elevation_profile.dart app/lib/components/trail/trail_panel.dart`
to return 0 for both files. `elevation_profile.dart` returns 0. `trail_panel.dart` returns **1**, from
the line the plan itself directs (`totalDistance: metrics?.distance` - the *named parameter* of
`TrailTimeline`, an unrelated, pre-existing widget in `trail_timeline.dart`, which is not in this
plan's `files_modified`). This is not a reference to the deleted `GpxStats.totalDistance` field; it is
`TrailTimeline`'s own public constructor parameter name, present at that call site both before and
after this plan (previously `totalDistance: totals?.totalDistance`, now `totalDistance:
metrics?.distance`). Renaming `TrailTimeline`'s parameter to dodge this grep gate would be an
out-of-scope, unnecessary API change to a file this plan does not touch. Verified via targeted grep
that zero references to the deleted `GpxStats`, `getTotals()`, `totalElevationGain`, or
`totalElevationloss` remain anywhere in `app/lib/` outside comments - the T-34-19 threat (two
disagreeing metrics implementations) is fully mitigated; only the grep gate's pattern is
over-broad.

## Issues Encountered

None beyond the deviations and the documented grep-gate false positive above, both caught and resolved
during this plan's own verification steps before committing.

## Verification Performed

- `cd app && flutter test test/util/gpx_conversion_util_test.dart` - 44/44 pass
- `cd app && flutter analyze lib/util/gpx_conversion_util.dart test/util/gpx_conversion_util_test.dart` - no issues
- `cd app && flutter test test/util/gpx_corpus_test.dart` - 30/30 pass
- Cross-language agreement: `cd web && npx vitest run src/lib/models/gpx/gpx-corpus.test.ts` (30/30
  pass) and `cd app && flutter test test/util/gpx_corpus_test.dart` (30/30 pass) run back to back
  against the same unmodified `fixtures/gpx-corpus/`
- Non-vacuity: temporarily changed `06-stationary-ends-mid-swing/expected.json`'s
  `metrics.elevationGain` from 7 to 0 - both suites failed on exactly that fixture
  (`elevationGain: 7 not within ... of 0` / Dart `7.0 not within 0.000001 of 0`); reverted; `git status
  --porcelain fixtures/gpx-corpus` clean
- `cd app && flutter test test/util/format_util_test.dart` - 18/18 pass, including the four
  `trailDisplayDuration` cases
- `cd app && flutter analyze` - no errors (33 pre-existing info-level lints unaffected by this plan)
- `cd app && flutter test` (full suite) - 556 tests, 4 documented pre-existing failures in
  `test/components/route_planner/settings_tab_test.dart` (icon-lookup assertions, verified against the
  pre-phase baseline commit `fb381452` in a throwaway worktree per the execution prompt's known-state),
  no new failures. Note: this differs from the plan's own `<verification>` section, which named a
  different baseline set (`feed_item_test.dart` x2, `settings_screen_test.dart` x1); the execution
  prompt's explicitly re-verified `settings_tab_test.dart` baseline is treated as authoritative here.
- Behaviour equivalence, `fixtures/gpx-corpus/08-jittery-track/input.gpx`: the fixture's own
  `expected.json` `metrics.distance` is `100.07543398026468`, confirming `elevation_profile.dart`'s
  redirected preview path (`computeTrailMetrics`) now shows the smoothed ~100.075 m instead of the
  deleted `getTotals()`'s raw jitter-inflated ~110.083 m
- `git status --porcelain app/pubspec.yaml` - clean (no `assets:` entry added)
- `dart run build_runner build --delete-conflicting-outputs` - ran once after adding
  `TrailSummary.movingDuration`; produced no diff in any tracked generated file (the interface change
  is satisfied by hand-written getter overrides, not new freezed constructor fields), confirmed via
  `git status --short` before staging

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `trailFromGpx` and `computeTrailMetrics` are the app's only GPX-derived-metrics path; both preview
  surfaces (`elevation_profile.dart`, `trail_panel.dart`) and the shared corpus agree
- `movingDuration` is threaded through the `Trail` model, `TrailSummary` interface, and the display
  rule, but no producer writes it yet - D-11/D-12's session -> trail hand-off (the recording path
  populating `movingDuration` from `NavigationStats.elapsed`) is 34-05's work, not this plan's
- No blockers for 34-05..34-07

---
*Phase: 34-dart-conversion-port*
*Completed: 2026-08-01*

## Self-Check: PASSED

All created/modified files confirmed present on disk; all three task commit hashes
(`2fa8f91c`, `96bd1506`, `b336e76e`) confirmed present in `git log`.
