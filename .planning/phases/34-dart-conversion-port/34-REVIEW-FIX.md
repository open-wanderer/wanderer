---
phase: 34-dart-conversion-port
fixed_at: 2026-08-01T09:24:11Z
review_path: .planning/phases/34-dart-conversion-port/34-REVIEW.md
iteration: 1
findings_in_scope: 16
fixed: 15
skipped: 1
status: partial
---

# Phase 34: Code Review Fix Report

**Fixed at:** 2026-08-01T09:24:11Z
**Source review:** `.planning/phases/34-dart-conversion-port/34-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope (Critical + Warning): 16
- Fixed: 15
- Skipped: 1 (CR-03, deferred by explicit phase constraint — not a failure)

**Verification (all run after the final commit):**

| Check | Baseline | After fixes |
|---|---|---|
| `npx vitest run` (web) | 91 pass / 9 files | **114 pass / 10 files** |
| `npx svelte-check` | 0 errors, 0 warnings | **0 errors, 0 warnings** |
| `flutter test` | 580 pass, 1 skip, 4 fail | **653 pass, 1 skip, 4 fail** |
| `flutter analyze` | 41 issues (33 `deprecated_member_use` in `icon_util.dart`, 7 vendor, 1 `unnecessary_import`) | **41 issues — identical set** |

The 4 Flutter failures are the confirmed pre-existing
`app/test/components/route_planner/settings_tab_test.dart` set (identical names, baseline
commit `fb381452`); no new failure was introduced. +73 Dart tests and +23 TS tests were added.
The GPX corpus parity suites stayed green on both sides with **unwidened** tolerances
(1e-6 m distance/elevation, exact duration/points/bbox), and gained an 11th fixture.

## Fixed Issues

### CR-01: Enabling either post-capture toggle on a file import silently discards the GPX's waypoints, name and description

**Files modified:** `app/lib/util/route_planner_handoff_util.dart`, `app/lib/util/trail_import_util.dart`, `app/test/util/route_planner_handoff_util_test.dart`
**Commit:** `a7917fa8`
**Status:** fixed

**Applied fix:** `mergeHeightsIntoGpx` gained an optional `Gpx? source` parameter. When given, it
copies the ORIGINAL document's `metadata`, `wpts`, `rtes`, `extensions`, `version`/`creator` and
the first `<trk>`'s `name`/`cmt`/`desc`/`src`/`number`/`type`/`links`/`extensions` onto the
rebuilt document, so only the track GEOMETRY is replaced — exactly as the phase constraint
requires (merge back into the original, not return a bare `Gpx()`). `importTrailFile` passes
`source: gpx`; the recording and planner call sites omit it and are behaviourally unchanged.

Copying `rtes` through cannot double-count metrics: both `computeTrailMetrics` and
`GpxMappingUtils.allPoints` read `trks` only — verified by direct source read before making the
change.

**Regression tests added** (5, using a waypoint- and metadata-bearing fixture as required):
the merged document keeps `metadata.name`/`desc`, `wpts`, `rtes` and the trk name; the draft
trail built from it keeps the GPX's own name/description/waypoints; the **re-serialised**
document (what actually gets uploaded) still contains them; non-track content survives an empty
shape; and `mergeHeightsIntoGpx` without `source:` still returns a bare document.

---

### CR-02: The sanitizer neutralises 4 of `package:gpx`'s 12 unguarded parse sites

**Files modified:** `app/lib/util/gpx_conversion_util.dart`, `app/test/util/gpx_conversion_util_test.dart`
**Commit:** `8c161d42`
**Status:** fixed

**Applied fix:** `sanitizeGpxEleAndTime` was extended and renamed `sanitizeGpxNumericAndTime`.
It now covers all eight additional tags the verifier confirmed against `package:gpx` 2.3.0's
`gpx_reader.dart:339-347` — `hdop`, `vdop`, `pdop`, `magvar`, `geoidheight`, `ageofdgpsdata`
(`_readDouble`) and `sat`, `dgpsid` (`_readInt`) — alongside the existing `ele`/`time`, using
the same regex-rewrite-to-self-closing-tag precedent.

**Non-corruption guarantees** (each pinned by its own test):
- Only an exact, attribute-less `<tag>…</tag>` pair matches, so a namespaced `<gpx:hdop>` or a
  longer same-suffix tag `<myele>` is untouched.
- Rewriting happens **only outside** CDATA sections and XML comments — a new
  `_rewriteOutsideProtectedRegions` helper splices those spans back verbatim, so a
  `<desc><![CDATA[… <ele>N/A</ele>]]></desc>` is not mutated. (XML forbids a raw `<` inside an
  attribute value, so attributes can never contain a matchable span.)
- A body is rewritten only when it fails to parse, so `<ele>0</ele>` and a pretty-printed
  `<ele>\n 1000.5\n</ele>` survive verbatim; the pass is idempotent.

`<number>` (rte/trk) and `<year>` (copyright) also reach `_readInt` but are deliberately NOT
rewritten — both names are generic enough that a context-free rewrite could clobber same-named
content inside an `<extensions>` block, and neither is in the commonly-emitted set the finding is
about. This is documented in the source.

**Tests added:** 32 — empty/whitespace/non-numeric rewrite plus a
"raw parser throws, `parseGpxSafely` does not" proof for each of the 8 new tags, valid-value
preservation per tag, `int.parse` whitespace-trim and decimal-rejection cases, a realistic
Garmin-style trkpt carrying all ten tags empty, and the four non-corruption guarantees above.

---

### CR-04: The convert endpoint reflects arbitrary unauthenticated request bodies verbatim

**Files modified:** `web/src/routes/api/v1/trail/convert/+server.ts`, `web/src/routes/api/v1/trail/convert/convert.test.ts`
**Commit:** `38c6db7b`
**Status:** fixed

**Applied fix:** Input **validation** plus response hardening — `gpx2trail` was NOT reintroduced
and the reverse-geocode block was NOT restored (D-05/D-07 respected). A new `assertParsableGpx`
runs `GPX.parse` purely as a validator on every input branch and throws a
`400 "Invalid GPX content"` on failure; nothing derived from the parse is computed, persisted or
returned. `GPX.parse` throws both for non-XML and for XML whose root is not `<gpx>`, which is the
acceptance boundary wanted. The 200 response gained `X-Content-Type-Options: nosniff` and
`Content-Disposition: attachment; filename="track.gpx"`. The OpenAPI 400 description was updated.

**`convert.test.ts:36-47` was updated as required** — the byte-reflection assertion was retitled
to make explicit that the round trip is the contract only for an already-validated GPX, and a
sibling test now asserts the two hardening headers. Seven rejection tests were added (plain text,
an HTML document, well-formed non-GPX XML, an XHTML+script payload, malformed XML claiming to be
GPX, a JSON blob, and a non-GPX body through the JSON branch), each also asserting the response
does not echo the submitted body.

---

### WR-01: Off-by-one in `segmentPolylinesFromTrack`'s not-found fallback

**Files modified:** `app/lib/util/route_planner_handoff_util.dart`, `app/test/util/route_planner_handoff_util_test.dart`
**Commit:** `9d164e59`
**Status:** fixed: requires human verification

**Applied fix:** The fallback now resumes at `k = i > 0 ? i - 1 : 0`, emitting the still-outstanding
`(i-1, i)` pair first. Two tests assert `polylines.length == anchors.length - 1` and correct
pair-to-polyline ordering, for both a missing middle anchor and a missing first anchor.

**Why human verification:** this is an index-arithmetic correction on a defensive path whose
consumer (`RouteAnchorsNotifier.seedFromTrack`) indexes positionally. The tests pin the count and
ordering, but a human should confirm the intended pairing semantics.

---

### WR-02: A failed road-snap falls back to the downsampled shape

**Files modified:** `app/lib/util/route_planner_handoff_util.dart`, `app/lib/util/trail_import_util.dart`, `app/lib/routes/navigation_screen.dart`, `app/test/util/route_planner_handoff_util_test.dart`
**Commit:** `c86a3412`
**Status:** fixed

**Applied fix:** `snapShapeToRoads` gained an optional `fallbackShape`, returned on error/rejection
instead of the ≤500-point request hint. Both persisting call sites (`trail_import_util.dart`,
`navigation_screen.dart`) now pass their full-resolution geometry. Four tests cover a failed
request, a `snapResultAcceptable`-rejected result, an accepted snap, and the no-`fallbackShape`
default (unchanged for callers whose hint IS their geometry).

---

### WR-03: A silently-failed snap still invalidates the leg's elevations

**Files modified:** `app/lib/util/route_planner_handoff_util.dart`, `app/test/util/route_planner_handoff_util_test.dart`
**Commit:** `23f7a402`
**Status:** fixed: requires human verification

**Applied fix:** Added `snapShapeToRoadsResult`, returning
`({List<Map<String, double>> shape, bool snapped})`. `buildFinalPlannedGpx` now `continue`s on
`!snapped`, so a leg that fell back keeps both its geometry and its elevations.
`snapShapeToRoads` is a thin wrapper over it, so existing callers are unchanged.

**Regression test:** a session seeded with real elevations, then run against an api where BOTH
`/valhalla/trace-route` and `/valhalla/height` fail, still exports `ele: 500` on every point.
Pre-fix that test produces `ele: null` throughout.

**Why human verification:** this changes state-invalidation logic under partial network failure.

---

### WR-04: `_onFinish`'s double-tap guard no longer covers the gate

**Files modified:** `app/lib/routes/route_planner_screen.dart`
**Commit:** `ee18e7f5`
**Status:** fixed

**Applied fix:** `if (!mounted || _finishing) return;` after `await resolveTrackSaveOptions(...)`,
mirroring `_saveRecordedTrack`'s own post-await re-check. The doc comment was corrected to state
why the re-check (not just `mounted`) is required.

No automated test: driving two concurrent taps through a modal bottom sheet inside
`tester.runAsync` is inherently flaky, and a flaky test here is worse than none. The change is
three lines with an explicit rationale comment.

---

### WR-05: The Dart corpus's waypoint-icon assertion is tautological

**Files modified:** `app/test/util/gpx_corpus_test.dart`
**Commit:** `7b6324f5`
**Status:** fixed

**Applied fix:** The expected value is no longer computed with the production expression. The test
now asserts `fontAwesomeIconsMap[expectedIcon]` is non-null first (with a reason naming the
missing sym), then compares against that lookup directly — so losing the `campground` or
`mountain` key fails this suite instead of collapsing both sides to `circle`. This restores
field-for-field comparability with the TS counterpart.

---

### WR-06: TS and Dart diverge on a non-empty but unparseable `<time>`

**Files modified:** `web/src/lib/models/gpx/waypoint.ts`, `web/src/lib/models/gpx/gpx.test.ts`, `web/src/lib/models/gpx/gpx-corpus.test.ts`, `app/test/util/gpx_corpus_test.dart`, `fixtures/gpx-corpus/README.md`, `fixtures/gpx-corpus/11-malformed-time/{input.gpx,expected.json,DERIVATION.md}`
**Commit:** `0f379569`
**Status:** fixed: requires human verification

**Applied fix:** Went beyond the reviewer's `Number.isNaN` guard to close the whole divergence
class. `Waypoint`'s constructor now routes `<time>` through a `parseGpxTime` helper that gates on
`DART_DATETIME_GRAMMAR` — Dart's `DateTime._parseFormat` transcribed verbatim from the SDK — AND
an `Invalid Date` check. That covers both the `Invalid Date`-is-truthy → `NaN` duration bug and
the "formats V8 accepts but `DateTime.tryParse` rejects" class (`Jan 1 2024`, `2024/01/01`) the
reviewer's suggested fix would have left open. GPX 1.1 mandates ISO 8601, so nothing a conforming
exporter emits is excluded.

**Corpus fixture 11 (`11-malformed-time`) added**, hand-derived per D-02 with a full
`DERIVATION.md`: distance from a fresh independent haversine transcription (`R = 6371000`,
`269.18400325278367 m`), bounding box/centroid from plain arithmetic, `durationMs = 0` from the
documented guard rules. No value came from executing either implementation. Both suites' minimum
corpus-size guards were bumped 10 → 11.

Seven TS unit tests were also added, including one asserting `gpx.features.duration` is `0` and
**never `NaN`** when the start time is unparseable.

**Why human verification:** this narrows what the web accepts as a `<time>`. A residual, documented
asymmetry remains in the opposite direction (a 6-digit year or an out-of-range month that Dart's
`DateTime` normalises but V8's `Date` rejects) — unreachable from conforming GPX, but a human
should confirm the strictness is acceptable for this codebase's real inputs.

---

### WR-07: `buildFinalPlannedGpx`'s "no network when both flags are off" claim is false

**Files modified:** `app/lib/util/route_planner_handoff_util.dart`, `app/test/util/route_planner_handoff_util_test.dart`
**Commit:** `2c29db61`
**Status:** fixed: requires human verification

**Applied fix:** Took the reviewer's first option — gate the backfill on connectivity and make the
doc TRUE — rather than merely correcting the doc. The `pending` list is now empty when
`onlineStatusProvider` is false, so an offline finish issues ZERO requests instead of one per
un-elevated leg. This is safe for existing tests because `OnlineStatus.build()` is optimistically
`true`. The doc comment was rewritten to describe the two distinct gates (flag for snap,
connectivity for heights).

**Two tests added** covering the case the pre-existing test structurally could not reach (it
transplanted a fully-elevated state, leaving `pending` empty): offline with **un-elevated** legs
asserts `requestCount == 0`; online with un-elevated legs and a failing api asserts the call IS
attempted and tolerated.

Test-harness note: `_pumpSnapRef` and `_pumpHeightRefWith` now both override
`onlineStatusProvider`. This is required, not cosmetic — Riverpod asserts a `ProviderScope`'s
override COUNT never changes across a rebuild, and several tests pump both helpers in one body.

**Why human verification:** this is a behaviour change (skip vs. attempt-and-tolerate) driven by
the app's connectivity heuristic.

---

### WR-08: A zero-length recording persists `moving_duration = 0`

**Files modified:** `app/lib/util/gpx_conversion_util.dart`, `app/test/util/gpx_conversion_util_test.dart`
**Commit:** `164bbd3a`
**Status:** fixed: requires human verification

**Applied fix:** Took the reviewer's second option — map zero in `trailFromGpx` rather than at the
`navigation_screen.dart` call site. That is the single choke point covering every caller, and it
catches a case the suggested `navStats.elapsed > Duration.zero` check would miss: `inSeconds`
truncates, so a 500 ms elapsed also yields 0. Three tests: `Duration.zero` → null, 500 ms → null,
1 s → `1.0`.

**Why human verification:** it changes what the app writes for `moving_duration` in an edge case.

---

### WR-09: `moving_duration` documented in OpenAPI but stripped by the Zod schemas

**Files modified:** `web/src/lib/models/api/trail_schema.ts`, `web/src/lib/models/api/trail_schema.test.ts` (new)
**Commit:** `21498f3c`
**Status:** fixed

**Applied fix:** `moving_duration: z.number({ coerce: true }).nonnegative().optional()` added to
both `TrailCreateSchema` and `TrailUpdateSchema`, matching the sibling `duration` field's shape.
A new test file asserts the field survives both schemas, coerces a numeric string, stays optional,
and rejects a negative value.

---

### WR-10: Editing a recorded trail's route leaves a stale `moving_duration`

**Files modified:** `web/src/routes/trail/edit/[id]/+page.svelte`, `app/lib/util/route_planner_handoff_util.dart`, `app/test/util/route_planner_handoff_util_test.dart`
**Commit:** `7d393431`
**Status:** fixed: requires human verification

**Applied fix:** Both sides now clear the field when the geometry is replaced. Web: `updateTotals`
— the single choke point for all ~14 route-mutation paths — sets `moving_duration: 0`. Dart:
`mergeRouteIntoTrail` sets `movingDuration: 0`.

Cleared as `0`, **not** `null`, for a concrete reason verified in source: `buildFormData` returns
early on `null`/`undefined` and `form_data_util.dart`'s guard is `if (movingDuration != null)`, so
a null is simply never transmitted and the stale stored value survives. Freezed's `copyWith` also
cannot assign null. `0` is exactly what the display rule
(`moving_duration > 0 ? moving_duration : duration`) reads as "no moving time", so it both
overwrites the stale value and restores the correct fallback.

**Why human verification:** this is a product-policy decision (a re-drawn route loses its recorded
moving time) rather than a mechanical correction.

---

### WR-11: A zero-point leg emits an empty `<trkseg>`

**Files modified:** `app/lib/util/route_planner_handoff_util.dart`, `app/test/util/route_planner_handoff_util_test.dart`
**Commit:** `9324962d`
**Status:** fixed

**Applied fix:** The export loop now guards with `if (legPoints[i].isNotEmpty)`, so a 0-point leg
emits no `Trkseg` at all. A test seeds a 3-leg route with an empty middle polyline and asserts
2 trksegs out, none of them empty.

Honest scoping note: because `anchorsFromTrack` already filters empty segments, the observable
change is that no meaningless empty `<trkseg>` is written into the persisted file — anchor
recovery on re-edit is identical either way. The reviewer's stated harm ("that anchor disappears")
occurs regardless, since the leg genuinely has no points. The fix is still correct (an empty leg is
deliberately unrepresentable in the round-trip format) but is narrower than the finding implies.

---

### WR-12: `importTrailFile`'s blanket `catch` also wraps the navigation push

**Files modified:** `app/lib/util/trail_import_util.dart`, `app/test/util/trail_import_util_test.dart`
**Commit:** `df16e906`
**Status:** fixed

**Applied fix:** The `try` now ends after `buildLocalTrail`; `pendingImportedTrail = trail` and
`navContext.push(...)` moved outside it, so a throw from the push can no longer report a successful
import as failed. The handler is `catch (e, st)` with
`debugPrint('importTrailFile failed for "$name": $e\n$st')` before the toast. A test with a
`<trkpt>` missing `lat`/`lon` asserts the toast fires and `pendingImportedTrail` stays null; the
log line was confirmed to fire (`importTrailFile failed for "broken.gpx": Bad state: No element`).

## Skipped Issues

### CR-03: `parseGpxSafely` is not applied at two of the three GPX parse sites

**Files:** `app/lib/provider/trail/trail_provider.dart:42-43`, `app/lib/entities/trail_entity.dart:187`
**Reason:** skipped: deferred by explicit phase constraint — CR-03 sits outside every plan's declared `files_modified`, the verifier recorded it as a warning rather than a blocker, and the fix instructions state it is out of scope for this pass and must be noted as deferred rather than fixed.
**Original issue:** `parseGpxSafely` documents itself as "the single sanctioned parse entry point
for any GPX this app did not itself produce", but two production parse sites still call
`GpxReader().fromString(sanitizeGpxEmail(...))` directly — server-downloaded GPX (whose
`FormatException` is swallowed by a broad `catch (_)` and silently degrades to the offline cache)
and the offline ObjectBox cache read-back (whose exception escapes the notifier, making the trail
permanently un-openable offline).

**Note for whoever picks this up:** CR-02's fix has now widened the crash class
`parseGpxSafely` protects against from 2 tags to 10, which makes redirecting these two sites
strictly more valuable than when the review was written. The redirect itself is a two-line change
plus the repo-guard test the reviewer describes; it was left undone only because of the scope
constraint, not because of any technical obstacle.

---

_Fixed: 2026-08-01T09:24:11Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
