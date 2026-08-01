---
phase: quick-260801-opr
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/lib/models/gpx/gpx.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.ts
  - web/src/lib/models/gpx/gpx.test.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.test.ts
  - web/src/lib/models/gpx/gpx-corpus.test.ts
  - app/lib/util/gpx_conversion_util.dart
  - app/test/util/gpx_conversion_util_test.dart
  - app/test/util/gpx_corpus_test.dart
  - fixtures/gpx-corpus/README.md
  - fixtures/gpx-corpus/04-switchback-scramble/expected.json
  - fixtures/gpx-corpus/04-switchback-scramble/DERIVATION.md
  - fixtures/gpx-corpus/08-jittery-track/expected.json
  - fixtures/gpx-corpus/08-jittery-track/DERIVATION.md
  - fixtures/gpx-corpus/12-dense-switchback/input.gpx
  - fixtures/gpx-corpus/12-dense-switchback/expected.json
  - fixtures/gpx-corpus/12-dense-switchback/DERIVATION.md
  - .planning/REQUIREMENTS.md
  - .planning/ROADMAP.md
autonomous: true
requirements: [CONV-05-REVERT, PORT-02]

must_haves:
  truths:
    - "Importing 19440058502_ACTIVITY.fit on web reports ~10971 m (10.97 km) in the trail summary, the same figure the elevation profile already showed, against the device's own session.total_distance of 10912.01 m"
    - "The same file in the app reports 10.97 km on the create screen and on the saved trail's stat chip and chart header"
    - "Elevation gain/loss is unchanged on both platforms (344 m up / 351 m down on the reference file) — thresholdXY_m and GpxMetricsComputation(5, 5) are untouched"
    - "GPX.getTotals()'s reported distance is exactly the last cumulativeDistance entry — the same accumulator, not two decoupled ones"
    - "A corpus fixture at real GPS sampling density (~3.85 m mean hop over a switchback) reports its full ~154 m raw length, so re-introducing the 5 m distance gate fails both language suites by ~77 m"
    - "totalDistanceSmoothed still exists on both classes, carries a not-reported marker, and has no remaining reporting consumer"
    - "CONV-05 is recorded as superseded in REQUIREMENTS.md with the FIT ground-truth reason and a pointer to this quick task, not silently flipped"
    - "No Go-side code computes or overrides trails.distance on a client save"
  artifacts:
    - path: "fixtures/gpx-corpus/12-dense-switchback/input.gpx"
      provides: "41-point switchback at ~3.85 m mean hop — real watch sampling density, every hop under the 5 m gate"
      contains: "Dense Switchback"
      min_lines: 45
    - path: "fixtures/gpx-corpus/12-dense-switchback/expected.json"
      provides: "Raw-distance expectations (~154.084 m) for the density fixture, both metrics and trail blocks"
      contains: "12-dense-switchback"
      min_lines: 25
    - path: "fixtures/gpx-corpus/12-dense-switchback/DERIVATION.md"
      provides: "First-principles derivation plus the ~77.292 m counterfactual the reverted gate would report"
      contains: "77.29"
      min_lines: 40
  key_links:
    - from: "web/src/lib/models/gpx/gpx.ts"
      to: "GpxMetricsComputation.totalDistance"
      via: "getTotals() reports the raw accumulator"
      pattern: "totalDistance = metrics\\.totalDistance;"
    - from: "app/lib/util/gpx_conversion_util.dart"
      to: "GpxMetricsComputation.totalDistance"
      via: "computeTrailMetrics reports the raw accumulator"
      pattern: "distance: metrics\\.totalDistance,"
    - from: "web/src/lib/models/gpx/gpx-corpus.test.ts"
      to: "fixtures/gpx-corpus/12-dense-switchback"
      via: "auto-discovery floor raised so a silently-shrunk corpus fails"
      pattern: "toBeGreaterThanOrEqual\\(12\\)"
    - from: "app/test/util/gpx_corpus_test.dart"
      to: "fixtures/gpx-corpus/12-dense-switchback"
      via: "auto-discovery floor raised so a silently-shrunk corpus fails"
      pattern: "fixtures\\.length < 12"
---

<objective>
Report the raw haversine accumulator as a trail's distance instead of the 5 m-gated smoothed
accumulator, reverting the reporting half of CONV-05 (`b7631bef`, Phase 33).

A trail imported from `19440058502_ACTIVITY.fit` shows two lengths on one page — summary
**10.55 km**, elevation profile **10.97 km** — identically on web and app. Two accumulators walk
the same points: raw sums every hop (10971.38 m), smoothed holds an anchor and adds the chord only
once displacement clears 5 m (10553.46 m). Only smoothed is reported; every chart re-walks the
geometry and gets raw.

Ground truth decides it. Decoded from the FIT binary, `session.total_distance`, the single
`lap.total_distance` and the last record's cumulative distance all agree: **10912.01 m**. Raw is
**+0.54%**; the 5 m gate is **−3.29%** — 6x worse. The gate is chord-shortcutting, not noise
removal: 1027 sub-5 m hops totalling 2703 m occur while moving at >= 0.5 m/s, because switchbacks
at a 4 s sample interval put most hops under 5 m at hiking pace. Phase 33 already refuted itself —
CONV-04 removed this exact horizontal threshold from elevation for switchbacks and scrambles while
leaving the identical flaw in distance.

**This is a decided direction, not an open question.** Do not propose a speed-aware filter, do not
re-litigate raw-vs-smoothed, do not backfill existing trails.

Work items (referenced as Q-01..Q-06 in task actions), from the approved plan at
`/Users/christianbeutel/.claude/plans/3-the-summary-at-lively-scone.md`:

- **Q-01** Report raw — two lines (`gpx.ts:148`, `gpx_conversion_util.dart:435`), plus the
  now-false D-01 comment at `gpx.ts:157-160` and not-reported markers on both
  `totalDistanceSmoothed` declarations.
- **Q-02** Regenerate the only two corpus fixtures that change: `04-switchback-scramble`
  (0.000 -> 4.403 m) and `08-jittery-track` (100.075 -> 110.083 m), including both `notes` fields,
  which currently assert the gated value as correct.
- **Q-03** Add a real-sampling-density corpus fixture — the class of track every existing fixture
  misses, and the reason this shipped.
- **Q-04** Re-point the tests that encode the old behaviour; invert (never delete) the executable
  invariant; retitle where wording implies smoothed is what gets reported.
- **Q-05** Mark CONV-05 superseded in `.planning/REQUIREMENTS.md` with a one-line reason and a
  pointer to the FIT measurement — not silently flipped, because corpus fixtures still carry
  `covers: ["CONV-05"]`.
- **Q-06** Verify `db/` neither computes nor overrides `distance` on save (Phase 34 D-08: clients
  compute their own metrics). Investigation item — report the finding.

**Hard constraints — a violation is a failed plan:**

1. **Do not change `thresholdXY_m` or the `GpxMetricsComputation(5, 5)` constructor arguments.**
   `gpx-metrics-computation.ts:209` uses `thresholdXY_m` as the horizontal-stillness check inside
   the elevation noise filter (`returnDistance < this.thresholdXY_m`); the Dart port mirrors it.
   Elevation output must be bit-identical before and after.
2. **Do not delete `totalDistanceSmoothed`** or its `lastFilteredPointXY` anchor. It becomes an
   unreported field — removing it would also strip an anchor from a class whose other anchors are
   elevation-critical, and it is what a future speed-aware filter would build on.
3. **Do not touch any chart/elevation-profile code.** `app/lib/components/trail/elevation_profile.dart`
   and `web/src/lib/vendor/maplibre-elevation-profile/` already use raw and converge for free. The
   comment block at `elevation_profile.dart:667-686` records that swapping that accumulator was
   tried and reverted (it destroys gradient colouring). Leave it alone.
4. **Do not stage or commit these five files** — they are unrelated uncommitted work already in the
   tree: `app/lib/components/navigation/track_save_options_sheet.dart`,
   `app/lib/routes/trail_source_select_screen.dart`, `app/lib/util/track_save_options_util.dart`,
   `web/src/lib/util/gpx_util.ts`, `web/src/routes/api/v1/trail/convert/+server.ts`.
   Commit only explicit paths. Never `git add -A`, never `git add .`.

`web/src/lib/util/gpx_util.ts` needs no edit anyway: line 76 is `trail.distance = totals.distance`,
so `gpx2trail()` inherits the fix through `getTotals()`.

Output: raw-reported distance on both platforms, two regenerated fixtures, one new density fixture
wired into both parity suites, re-pointed tests, and a superseded CONV-05.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

@web/src/lib/models/gpx/gpx.ts
@web/src/lib/models/gpx/gpx-metrics-computation.ts
@app/lib/util/gpx_conversion_util.dart
@fixtures/gpx-corpus/README.md
@fixtures/gpx-corpus/04-switchback-scramble/expected.json
@fixtures/gpx-corpus/04-switchback-scramble/DERIVATION.md
@fixtures/gpx-corpus/08-jittery-track/expected.json
@fixtures/gpx-corpus/08-jittery-track/DERIVATION.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Report raw, and regenerate everything that asserted the gated value</name>
  <files>web/src/lib/models/gpx/gpx.ts, web/src/lib/models/gpx/gpx-metrics-computation.ts, web/src/lib/models/gpx/gpx.test.ts, app/lib/util/gpx_conversion_util.dart, app/test/util/gpx_conversion_util_test.dart, fixtures/gpx-corpus/04-switchback-scramble/expected.json, fixtures/gpx-corpus/04-switchback-scramble/DERIVATION.md, fixtures/gpx-corpus/08-jittery-track/expected.json, fixtures/gpx-corpus/08-jittery-track/DERIVATION.md</files>
  <action>
The behaviour change and every assertion it invalidates land in one commit — splitting them leaves
the suites red at a commit boundary.

**Q-01, the two report lines.** In `web/src/lib/models/gpx/gpx.ts:148`, change the right-hand side
of the `totalDistance = ...` assignment from `metrics.totalDistanceSmoothed` to
`metrics.totalDistance`. In `app/lib/util/gpx_conversion_util.dart:435`, change the
`distance:` argument of the returned `GpxTrailMetrics` from `metrics.totalDistanceSmoothed` to
`metrics.totalDistance`. Leave the surrounding `finalElevationGain`/`finalElevationLoss` lines and
the `final*`-rather-than-`total*Smoothed` comment above them untouched — that comment is about
elevation and stays true.

**Q-01, the now-false D-01 comment** at `gpx.ts:157-160`. It currently claims `cumulativeDistance`
is "deliberately not equal to the reported `distance` above, which is the smoothed total." After
this change they are the same accumulator by construction: `addAndFilter` pushes
`this.totalDistance` onto `cumulativeDistance` immediately after the `totalDistance +=` on every
call, so the last entry is that exact double. Rewrite the comment to keep the two facts that are
still load-bearing — index-aligned with `flatten()`, one entry per point, first entry 0, consumed
by the trail-edit crop slider for position interpolation — and replace the decoupling claim with
the new relationship: same accumulator as the reported `distance`, whose last entry therefore
equals that total by construction.

**Q-01, not-reported markers.** Above `totalDistanceSmoothed = 0;` at
`gpx-metrics-computation.ts:55` and above `double totalDistanceSmoothed = 0;` at
`gpx_conversion_util.dart:151`, add a short comment (matching each file's existing comment voice)
recording: NOT REPORTED as of 2026-08-01 — no consumer reads this for a trail's distance;
`getTotals()`/`computeTrailMetrics` report the raw `totalDistance`. State why it survives: its
`lastFilteredPointXY` anchor sits in a class whose other anchors are elevation-critical, and a
future speed-aware filter would build on it. Point at
`.planning/quick/260801-opr-report-raw-distance-instead-of-the-5m-ga/`. Do not touch
`thresholdXY_m`, the `GpxMetricsComputation(5, 5)` call sites, or any elevation field.

**Q-04, web tests.** In `gpx.test.ts`, the describe at line 117 ("GPX.getTotals — CONV-05 smoothed
distance") is now backwards. Retitle the describe to say the reported distance is the raw
accumulator and note CONV-05 is superseded. Its first test (`:118-138`) asserts
`toBeCloseTo(100.075, 0)` on the 16-point jitter fixture — re-point that number to the raw sum,
`110.083`, and update the test name and the explanatory comment at `:119-124` (which currently
frames 100.075 as "real forward travel" and 110.083 as merely what `cumulativeDistance` holds).
Its second test (`:140-160`) is an executable invariant asserting reported `<` raw — **invert it,
do not delete it**: assert the reported distance *equals* the last `cumulativeDistance` entry.
Because both come from the same accumulation in the same order, use strict equality
(`toBe(rawTotal)`), not a closeness matcher, and retitle to say the two are the same accumulator.
Also reword the comment at `:209-211` inside the `:207` describe ("CONV-05 does not regress the
planner route") — its assertion of 444.78 stays correct (every hop there clears 5 m so raw and
gated coincide), but the wording implies smoothed is what gets reported.

**Q-04, Dart tests.** In `app/test/util/gpx_conversion_util_test.dart`, line 467 is a report-level
assertion that **will break**: `expect(metrics.distance, 0.0)` inside the CONV-04 scramble test at
`:452-469`. Under raw the same 12-point stretch measures ~4.4033 m. Re-point it to
`closeTo(4.403319, 1e-6)` and rewrite the comment at `:466` — the independence of the elevation and
distance thresholds is still demonstrated (88 m of climb against 4.4 m of travel), but "distance
smoothing must stay gated" is no longer the point being made. The CONV-05 mirror at `:560-585`
asserts `totalDistanceSmoothed` (100.075) and `totalDistance` (110.083) against the **class**
directly — both stay valid and both numbers stay; only update the test name and comment so nothing
implies `totalDistanceSmoothed` is what gets reported. Lines 376 (134.59), 606 (444.78), 644 (0.0,
empty document) and the 881-915 parity block are expected to keep passing — verify, do not edit.

**Q-02, the two changed fixtures.** Only these two of the eleven change; the other nine are
byte-identical — verify by running the suites, do not edit them.

For `fixtures/gpx-corpus/04-switchback-scramble/`: set `metrics.distance` **and** `trail.distance`
to the raw sum of the eleven ~0.4003 m meridian hops. Derive it fresh with an independently
transcribed haversine (`R = 6371000`), per the corpus's derive-don't-execute rule; it must agree
with the `4.403319095226456` already recorded in that fixture's own DERIVATION.md (along a meridian
the hop sum and the direct first-to-last distance agree to within ~1e-12). Rewrite the `notes`
field: it currently states distance stays 0 and calls that proof the thresholds are gated
independently. The independence claim survives — 88 m of elevation against 4.4 m of horizontal
travel — but 0 must stop being presented as the desired outcome. In its DERIVATION.md, rewrite the
`### distance` section: keep the haversine one-liner, but replace the "`totalDistanceSmoothed`
stays exactly 0 throughout" conclusion with the raw sum as the reported value, and record 0 as the
counterfactual the superseded 5 m gate would have produced.

For `fixtures/gpx-corpus/08-jittery-track/`: set `metrics.distance` and `trail.distance` to
`110.08297737671097` — already derived in that fixture's own DERIVATION.md as `raw`. Rewrite the
`notes` field, which currently describes the smoothed value as "what `GPX.getTotals()` reports as
`distance` (gpx.ts:148)". In DERIVATION.md, rewrite the "Defect pinned" section (it currently
states CONV-05's now-superseded rule as the requirement) and the paragraph at `:56-61` that says
the corpus asserts the smoothed value; keep the node one-liner and both computed figures, and
present 100.075 as the counterfactual. Leave `covers: ["CONV-05"]` in place in both fixtures —
Task 3 marks CONV-05 superseded rather than deleting it precisely so these references stay valid.

Commit with an explicit path list naming exactly the nine files above. Never `git add -A` or
`git add .` — five unrelated files are uncommitted in this tree (see the objective's constraint 4).
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/web && npx vitest run src/lib/models/gpx/ 2>&1 | tail -20</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter test test/util/gpx_conversion_util_test.dart test/util/gpx_corpus_test.dart 2>&1 | tail -20</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && test "$(grep -c 'totalDistance = metrics\.totalDistanceSmoothed' web/src/lib/models/gpx/gpx.ts)" = 0 && test "$(grep -c 'distance: metrics\.totalDistanceSmoothed' app/lib/util/gpx_conversion_util.dart)" = 0 && echo REPORT_SWAP_OK</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && grep -q 'totalDistance = metrics\.totalDistance;' web/src/lib/models/gpx/gpx.ts && grep -q 'distance: metrics\.totalDistance,' app/lib/util/gpx_conversion_util.dart && echo RAW_REPORTED_OK</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && grep -q 'new GpxMetricsComputation(5, 5)' web/src/lib/models/gpx/gpx.ts && grep -q 'GpxMetricsComputation(5, 5)' app/lib/util/gpx_conversion_util.dart && grep -q 'totalDistanceSmoothed' web/src/lib/models/gpx/gpx-metrics-computation.ts && grep -q 'totalDistanceSmoothed' app/lib/util/gpx_conversion_util.dart && echo THRESHOLD_AND_FIELD_INTACT</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && git status --porcelain app/lib/components/navigation/track_save_options_sheet.dart app/lib/routes/trail_source_select_screen.dart app/lib/util/track_save_options_util.dart web/src/lib/util/gpx_util.ts web/src/routes/api/v1/trail/convert/+server.ts | grep -qv '^[MARD]' && echo FORBIDDEN_FILES_UNSTAGED</automated>
  </verify>
  <done>Both platforms report `totalDistance`; `thresholdXY_m`, `GpxMetricsComputation(5, 5)` and both `totalDistanceSmoothed` fields survive with not-reported markers; fixtures 04 and 08 assert 4.403319/110.083 in both `metrics.distance` and `trail.distance` with rewritten notes and DERIVATION sections; the gpx.test.ts invariant is inverted to a strict-equality assertion rather than deleted; Dart `:467` is re-pointed; both suites green; the five forbidden files remain unstaged.</done>
</task>

<task type="auto">
  <name>Task 2: Add the real-sampling-density corpus fixture and raise both discovery floors</name>
  <files>fixtures/gpx-corpus/12-dense-switchback/input.gpx, fixtures/gpx-corpus/12-dense-switchback/expected.json, fixtures/gpx-corpus/12-dense-switchback/DERIVATION.md, web/src/lib/models/gpx/gpx-corpus.test.ts, app/test/util/gpx_corpus_test.dart, fixtures/gpx-corpus/README.md</files>
  <action>
**Q-03.** The corpus's blindness is why the gate shipped: `10-realistic-track` has **zero** hops
under 5 m (min 39.39 m, mean 44.8 m) — ten times sparser than a real watch (4.27 m) — so no
existing fixture can see this defect class. This fixture is the regression guard that makes
re-introducing the gate impossible to do silently.

**Geometry.** Directory `fixtures/gpx-corpus/12-dense-switchback/`, GPX 1.1 document matching the
corpus convention exactly (`version="1.1" creator="wanderer-corpus"
xmlns="http://www.topografix.com/GPX/1/1"`, a `<metadata><name>Dense Switchback</name>`, one
`<trk>` with one `<trkseg>`). 41 `<trkpt>` elements, index `i = 0..40`:

- `lat` = `47 + (144 * i) / 1e7`, emitted as an exact decimal literal — `47` at i=0, `47.0000144`
  at i=1, `47.000576` at i=40.
- `lon` = `11` when `i` is even, `11.0000462` when `i` is odd — the switchback zig-zag.
- No `<ele>` and no `<time>` on any point. Elevation is deliberately out of scope here: fixture 04
  already pins threshold-independent elevation, and omitting `<ele>` avoids a second hand-trace of
  the defer-then-publish state machine. This fixture pins one thing — horizontal sampling density.

Generate `input.gpx` from an integer counter (`47 + 144*i/1e7`, `11 + (i % 2) * 462/1e7`) so no
value is hand-typed; follow fixture 08's element formatting (self-closed-style
`<trkpt lat="..." lon="..."></trkpt>`, trailing zeros dropped: write `47`, not `47.0`).

**Why it discriminates.** Every consecutive hop is ~3.852 m — under the 5 m gate, and at real watch
density. The raw accumulator therefore measures the full ~154.084 m, while the superseded gate
advances its anchor only every third point and chord-shortcuts to ~77.292 m: **the gate loses half
the length of a real switchback**. Record both figures in DERIVATION.md; the counterfactual is what
gives the guard teeth.

**Derivation (`derivation: "hand"`, so DERIVATION.md is mandatory).** Per the corpus's
derive-don't-execute rule, obtain every value independently — a freshly transcribed haversine
(`R = 6371000`) in a scratch `node -e` run for distance, plain arithmetic for the rest. Never call
`GPX.getTotals()`, `gpx2trail()`, `GpxMetricsComputation` or `computeTrailMetrics` to produce an
expected value. Cross-check: the raw sum must agree with 154.0841571 m and the gated counterfactual
with 77.2922075 m to within 1e-6; a disagreement means the point list drifted from the spec above.

**expected.json.** `id` `12-dense-switchback`, `derivation` `"hand"`, `covers` `["CONV-05"]` — the
same id fixture 08 carries, now pinning CONV-05's *superseded* form (reported distance is raw);
say so explicitly in `notes`, along with the mean hop spacing, the real-watch comparison, and the
77.292 m counterfactual. `metrics`: `distance` = the derived raw sum at full precision,
`elevationGain` 0, `elevationLoss` 0, `durationMs` 0, `pointCount` 41, `boundingBox`
`{ minLat: 47, maxLat: 47.000576, minLon: 11, maxLon: 11.0000462 }`, `centroid`
`{ lat: 47.000288, lon: 11.000022536585366 }`. The centroid latitude is exact — the mean of
`i = 0..40` is 20, so `47 + 20 * 0.0000144`; the longitude is `20 * 0.0000462 / 41` (20 odd indices
of 41 points), well inside the corpus's 1e-9 degree tolerance. Bounding-box and `trail.lat`/`lon`
are compared **exactly**, so write those as the identical decimal literals that appear in
`input.gpx`. `trail`: `name` `"Dense Switchback"`, `description` null, `lat` 47, `lon` 11, `date`
null, `distance` identical to `metrics.distance`, `elevationGain`/`elevationLoss` 0, `duration` 0,
the same four bbox values, `waypoints` `[]`.

**Wiring.** Both suites auto-discover fixture directories but assert a minimum count so a silently
shrinking corpus fails loudly. Raise both floors from 11 to 12: `gpx-corpus.test.ts:94`
(`expect(fixtures.length).toBeGreaterThanOrEqual(11)`) plus the doc comment at `:77-78`, and
`gpx_corpus_test.dart:72-75` (`if (fixtures.length < 11)` and its `StateError` message) plus the
doc comment at `:48-49`. No other wiring exists or is needed — discovery is by directory listing.

**README.** In `fixtures/gpx-corpus/README.md`: the `metrics` key table describes `distance` as
"smoothed total distance, metres" — change it to raw. Update the `08-jittery-track` row of the
fixture-to-defect coverage table (it currently reads "Reported distance is the smoothed
accumulator, not the raw jitter-inflated sum") and add a `12-dense-switchback` row describing the
density guard. Add the new directory to the layout tree.

Commit with an explicit path list naming exactly the six files above.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && test "$(ls -d fixtures/gpx-corpus/*/ | wc -l | tr -d ' ')" = 12 && test -f fixtures/gpx-corpus/12-dense-switchback/input.gpx && test -f fixtures/gpx-corpus/12-dense-switchback/expected.json && test -f fixtures/gpx-corpus/12-dense-switchback/DERIVATION.md && echo FIXTURE_PRESENT</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && test "$(grep -c '<trkpt' fixtures/gpx-corpus/12-dense-switchback/input.gpx)" = 41 && echo POINT_COUNT_OK</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && grep -q 'toBeGreaterThanOrEqual(12)' web/src/lib/models/gpx/gpx-corpus.test.ts && grep -q 'fixtures.length < 12' app/test/util/gpx_corpus_test.dart && echo FLOORS_RAISED</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/web && npx vitest run src/lib/models/gpx/gpx-corpus.test.ts 2>&1 | tail -15</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter test test/util/gpx_corpus_test.dart 2>&1 | tail -15</automated>
  </verify>
  <done>`12-dense-switchback` exists with all three files, 41 trkpts at ~3.852 m mean spacing, and expected values derived independently (raw ~154.084 m; DERIVATION.md records ~77.292 m as the gate's counterfactual); both suites discover 12 fixtures, both floors read 12, and both pass; README's metrics table says raw and its coverage table has the new row.</done>
</task>

<task type="auto">
  <name>Task 3: Supersede CONV-05, retitle the class-level tests, and confirm no Go-side distance</name>
  <files>.planning/REQUIREMENTS.md, .planning/ROADMAP.md, web/src/lib/models/gpx/gpx-metrics-computation.test.ts</files>
  <action>
**Q-05, requirement traceability.** `.planning/REQUIREMENTS.md:27` marks CONV-05 Complete and
`:102` carries its status row; corpus fixtures still declare `covers: ["CONV-05"]` and
`.planning/ROADMAP.md:341,361` reference it. Mark it **superseded**, never silently flip or delete
it. At `:27`, keep the `[x]` (it did ship) and append a supersession note: superseded 2026-08-01 by
`.planning/quick/260801-opr-report-raw-distance-instead-of-the-5m-ga/`, because the 5 m gate
chord-shortcuts switchbacks at real sampling density; FIT ground truth
(`19440058502_ACTIVITY.fit`, `session.total_distance` = 10912.01 m) measured raw at +0.54% against
the gate's −3.29%; distance is now the raw accumulator. **Scope the supersession to the smoothed
half only** — CONV-05's second clause (the dead, misaligned `cumulativeDistance` array) still
stands, that array was rebuilt index-aligned and its crop-slider consumer is live. Update the `:102`
status row from Complete to a superseded state naming this quick task.

In `.planning/ROADMAP.md`, leave lines 341 and 361 untouched — they are historical plan records and
rewriting them would falsify the log. Instead append one sentence to Phase 33's existing
`**Scope note:**` paragraph (at the end of the Phase 33 block) recording that CONV-05's
reported-distance half was superseded on 2026-08-01 and pointing at REQUIREMENTS.md.

**Q-04, class-level test titles.** `web/src/lib/models/gpx/gpx-metrics-computation.test.ts:103-113`
and `:197-214` exercise `GpxMetricsComputation` directly (`totalDistanceSmoothed` 0 and 100.075,
`totalDistance` 110.083). Every assertion stays valid and every number stays — the class is
unchanged. Retitle only where wording implies smoothed is what gets reported: the describe at
`:196` ("distance smoothing is unchanged") and the test at `:103` ("keeps totalDistanceSmoothed at
0 for the same stretch (distance smoothing must stay gated)") both read as statements about the
reported value. Reword them to say plainly that these assert the class's now-unreported smoothed
field, and add a one-line comment noting the reported distance is `totalDistance` since 2026-08-01.
Do not change any expected number in this file.

**Q-06, Go-side verification (investigation — report the finding, change nothing without cause).**
Confirm `db/` neither computes nor overrides `trails.distance` on a client save; if a Go-side
computation feeds that column on the save path, it must change too or it will overwrite the
corrected value. Prior reconnaissance found: `db/hooks/trails.go` has zero occurrences of
`distance`; the only Go distance producer is `db/plugins/importer/importer.go` (`:204`
`Distance: gpxData.Length2D()`, with a provider-metadata override at `:375-376`), which is the
plugin-import ingest path for third-party providers, not the client save path — and `Length2D()`
is itself a raw consecutive-pair sum, so it already agrees with the new reporting. Every other
`distance` hit under `db/` is a migration, a collection field definition, a Meilisearch attribute
list or a SQL view. Re-run the greps to confirm this still holds, then record the finding in the
SUMMARY: no Go change required, and why. If the greps contradict the above, stop and report rather
than expanding scope.

Commit with an explicit path list naming exactly the three files above.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && grep -i 'supersede' .planning/REQUIREMENTS.md | grep -q 'CONV-05' && grep -q '260801-opr' .planning/REQUIREMENTS.md && echo SUPERSEDED_RECORDED</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && grep -q 'CONV-05' fixtures/gpx-corpus/08-jittery-track/expected.json && grep -q 'CONV-05' .planning/REQUIREMENTS.md && echo COVERS_STILL_RESOLVES</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer && test "$(grep -c 'distance' db/hooks/trails.go)" = 0 && echo NO_GO_SAVE_HOOK_DISTANCE</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/web && npx vitest run src/lib/models/gpx/gpx-metrics-computation.test.ts 2>&1 | tail -12</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/web && npx vitest run src/lib/models/gpx/ 2>&1 | tail -8</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter test test/util/gpx_conversion_util_test.dart test/util/gpx_corpus_test.dart 2>&1 | tail -8</automated>
    <human-check>
**End-to-end against ground truth — the measurement this whole change rests on.**

1. Import `~/Downloads/19440058502_ACTIVITY.fit` on web. The summary header and the
   elevation-profile tooltip must **both** read **10.97 km** (device truth 10912.01 m; the summary
   read 10.55 km before this change).
2. Open the same file in the app: the create screen, then — after saving — the trail panel stat
   chip and the chart header must all read **10.97 km**.
3. **Elevation must not move**: 344 m up / 351 m down, before and after, on both platforms. This is
   the guard that `thresholdXY_m` survived intact. A change here means constraint 1 was violated.
4. A previously-saved trail keeps its stored distance until re-saved (no backfill, by decision);
   re-saving it updates the value.
    </human-check>
  </verify>
  <done>CONV-05 reads as superseded in REQUIREMENTS.md (both the requirement line and the status row) with the FIT ground-truth reason and a pointer to this quick task, scoped to the smoothed-distance half only; ROADMAP's Phase 33 scope note records the supersession without rewriting the historical plan lines; `covers: ["CONV-05"]` still resolves to a real requirement entry; gpx-metrics-computation.test.ts retitled with every number unchanged; the Go finding is recorded in the SUMMARY.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| third-party GPX/FIT file -> metrics computation | Untrusted coordinates reach the haversine accumulator; the reported total is now the unfiltered sum of every hop |
| corpus fixture on disk -> both language test suites | Fixture values are the parity oracle for the Dart port; a wrong expected value silently pins a wrong algorithm in both languages |
| uncommitted unrelated work in the tree -> this task's commits | Worktree isolation is off; five unrelated modified files are visible to any broad `git add` |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-opr-01 | Tampering | `totalDistance` accumulator fed by hostile coordinates | mitigate | Unchanged existing guard: `gpx-metrics-computation.ts:132` only adds a hop when `Number.isFinite(distance)`, and the Dart port mirrors it at `:239`; this plan alters which accumulator is *read*, never how either is fed |
| T-opr-02 | Tampering | Corpus expected values as the cross-language oracle | mitigate | Every new/changed value is derived from a freshly transcribed haversine, never from the implementation under test (corpus README's derive-don't-execute rule); Task 2 states explicit cross-check magnitudes (154.0841571 / 77.2922075) so a drifted point list fails loudly instead of pinning a wrong number |
| T-opr-03 | Tampering | Silent re-introduction of the 5 m distance gate by a future change | mitigate | `12-dense-switchback` fails by ~77 m in both languages if the gate returns; both suites' discovery floors raised to 12 so deleting the fixture also fails |
| T-opr-04 | Repudiation | A requirement flipped without a record | mitigate | Task 3 marks CONV-05 superseded with the FIT ground-truth reason and a pointer, rather than editing it in place; `covers: ["CONV-05"]` in two fixtures keeps resolving |
| T-opr-05 | Tampering | Unrelated uncommitted work swept into these commits | mitigate | Every task commits an explicit path list; `git add -A`/`git add .` forbidden; Task 1 verifies the five named files remain unstaged |
| T-opr-06 | Information disclosure | — | accept (n/a) | No new input surface, no network call, no credential, no user data path; the change swaps which of two already-computed in-memory numbers is reported |
| T-opr-07 | Tampering | npm/pub/cargo installs | accept (n/a) | No dependency is added or upgraded; no package-manager install task exists in this plan, so the Package Legitimacy Gate does not apply |
</threat_model>

<verification>
1. `cd web && npx vitest run src/lib/models/gpx/` — every GPX suite green (`gpx.test.ts`,
   `gpx-metrics-computation.test.ts`, `gpx-corpus.test.ts`).
2. `cd app && flutter test test/util/gpx_conversion_util_test.dart test/util/gpx_corpus_test.dart`
   — both green. (Three pre-existing unrelated failures — `feed_item_test.dart` x2,
   `settings_screen_test.dart` x1, recorded in Phase 18's `deferred-items.md` — are outside these
   two files and must not be fixed here.)
3. Corpus parity holds in both languages across all 12 fixtures; nine of the eleven original
   fixtures are byte-identical — confirmed by the suites passing without edits to them, and by
   `git status` showing only `04-*` and `08-*` modified under `fixtures/`.
4. `cd app && flutter analyze lib/util/gpx_conversion_util.dart` clean; `dart format` leaves no
   uncommitted churn on the edited Dart files.
5. End-to-end against the FIT ground truth, and the elevation non-regression — Task 3's
   `<human-check>` block. This is the load-bearing check: it is the measurement the entire change
   rests on.
6. `git log --stat` for the three commits shows only the files named in each task's `<files>`, and
   none of the five forbidden paths.
</verification>

<success_criteria>
- Web and app both report the raw haversine accumulator as a trail's distance; the summary and the
  elevation profile agree on one number, ~10.97 km for the reference FIT file against the device's
  own 10912.01 m.
- Elevation gain/loss is unchanged on both platforms; `thresholdXY_m`, both
  `GpxMetricsComputation(5, 5)` call sites, and both `totalDistanceSmoothed` fields (with their
  `lastFilteredPointXY` anchors) survive untouched, the latter marked not-reported.
- No chart or elevation-profile code was modified.
- `GPX.getTotals()`'s reported distance is provably the last `cumulativeDistance` entry — the
  executable invariant was inverted, not deleted — and the D-01 comment no longer claims otherwise.
- Fixtures 04 and 08 assert the raw values in both `metrics.distance` and `trail.distance`, with
  `notes` and `DERIVATION.md` rewritten so neither presents the gated value as correct.
- `12-dense-switchback` pins a real-density switchback (~3.852 m mean hop, 41 points) at its full
  ~154 m, records ~77.292 m as the gate's counterfactual, and is discovered by both suites whose
  floors now read 12.
- CONV-05 is recorded as superseded — scoped to its smoothed-distance half — with the FIT
  ground-truth reason and a pointer; the ROADMAP's historical plan lines are intact.
- The Go-side finding is recorded: nothing under `db/` computes or overrides `distance` on a client
  save.
- Three atomic commits, each with an explicit path list; the five unrelated modified files remain
  uncommitted.
</success_criteria>

<output>
Create `.planning/quick/260801-opr-report-raw-distance-instead-of-the-5m-ga/260801-opr-SUMMARY.md` when done.
</output>
</content>
