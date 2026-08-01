---
phase: 34-dart-conversion-port
reviewed: 2026-08-01T14:20:00Z
depth: standard
round: 3
files_reviewed: 22
files_reviewed_list:
  - app/lib/vendor/gpx/gpx_reader.dart
  - app/lib/vendor/gpx/gpx_tag.dart
  - app/lib/vendor/gpx/LICENSE
  - app/lib/vendor/gpx/README.md
  - app/lib/util/gpx_conversion_util.dart
  - app/lib/util/gpx_util.dart
  - app/lib/util/route_planner_handoff_util.dart
  - app/lib/util/trail_import_util.dart
  - app/lib/util/track_save_options_util.dart
  - app/lib/components/trail/elevation_profile.dart
  - app/lib/provider/trail/trail_library_provider.dart
  - app/lib/provider/trail/trail_provider.dart
  - app/lib/entities/trail_entity.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/routes/route_planner_screen.dart
  - app/test/util/gpx_conversion_util_test.dart
  - app/test/util/gpx_util_test.dart
  - app/test/util/gpx_corpus_test.dart
  - app/test/util/route_planner_handoff_util_test.dart
  - app/test/util/trail_import_util_test.dart
  - app/test/components/trail/elevation_profile_test.dart
  - app/test/entities/trail_entity_test.dart
  - web/src/routes/api/v1/trail/convert/+server.ts
  - web/src/lib/models/gpx/gpx-corpus.test.ts
findings:
  critical: 2
  warning: 8
  info: 2
  total: 12
status: issues_found
---

# Phase 34: Code Review Report (Round 3)

**Reviewed:** 2026-08-01T14:20:00Z
**Depth:** standard
**Files Reviewed:** 22
**Status:** issues_found
**Diff base:** `fb381452..HEAD`

## Summary

The two highest-risk fixes hold. The chart revert (`867e19b9`) is faithful — I diffed
`buildElevationTrackPoints` against the `fb381452` baseline `_parseGpx` and the only
differences are the shared haversine, the `continue` on a coordinate-less point, and the
`@visibleForTesting` hoist; the raw-vs-smoothed divergence, the sub-5 m axis collapse (round-2
WR-05) and the untimed-gap duration change (round-2 WR-07) all revert with it. The vendored
reader's tolerance set is now **complete**: I grepped upstream `gpx-2.3.0/lib/src/gpx_reader.dart`
for every throwing site (`firstWhere` without `orElse`, `double/int/DateTime.parse`, the
`as XmlStartElementEvent` cast) and all six are accounted for by LOCAL MODIFICATIONS 1–6.
`XmlParserException implements FormatException` (verified), so modification 6's documented
`Throws [FormatException]` contract is genuinely exhaustive for the fromString entry point. The
`LICENSE` is byte-identical to `~/.pub-cache/.../gpx-2.3.0/LICENSE` and is Apache 2.0 — round 2
mislabelled it BSD-3-Clause; that finding is withdrawn as stated and closed as fixed.

**However the anchor-seed guard (`867e19b9`) is a half-fix, and the half it does not cover is the
more reachable one.** The guard lives only in `addAndFilter`'s *initialisation* branch and tests
only for `null`. Two consequences, both reproduced by running the shipped code:

| Probe | Result |
|---|---|
| 20-point flat track (all ele 1000) with ONE coordinate-less `<trkpt ele="3000">` at index 10 | `elevationGain = 2000.0`, `elevationLoss = 2000.0` — entirely fabricated |
| 20-point track whose first `<trkpt>` is `lat="NaN" lon="NaN"` | `distance = 0.0`, `trail.lat = NaN`; `jsonEncode` throws `Converting object to an encodable object failed: NaN` |
| Same fixture the fix's own test uses, but reading `trail.lat` | `trailLat = null`, `trailLon = null`, `centroidLat = 45.86` (should be ~47.00) |
| `<wpt><name>Summit</name></wpt>` (no lat/lon) | `Waypoint(lat: 0.0, lon: 0.0)` — a Null Island waypoint, persisted on save |

The pattern across all four: LOCAL MODIFICATION 2 converts "malformed point → hard error" into
"malformed point → `null` coordinate", and the *new* Dart code downstream of it
(`computeTrailMetrics`, `trailFromGpx`) was only taught about that in one of the five places it
matters. This is the direct answer to the brief's question 3 — yes, the modification set creates
downstream paths that silently produce a wrong-but-plausible trail rather than an error, and
`importTrailFile`'s `hasUsablePoint` guard does not catch any of them.

Four of the nine round-2 warnings remain genuinely open (mutable-collection aliasing,
multi-segment flattening, the email splice, the substring repo guard) — all re-verified against
current code, all re-reported below with current line numbers.

## Closure verification (round 2)

| ID | Claim | Verdict |
|---|---|---|
| CR-01 | Anchor seed no longer takes a coordinate-less point | **PARTIAL.** `gpx_conversion_util.dart:189-191` closes the zero-distance case for a *leading* `null` point (measured 867 m on the fix's own fixture, was 0.0). The two other consequences round 2 listed — `trail.lat/lon == null` and centroid contamination — are unchanged, and the guard does not cover mid-track points (BL-02) or non-finite coordinates (BL-01). See BL-01, BL-02, WR-01, WR-02. |
| CR-02 | Chart gradient | **CLOSED.** `elevation_profile.dart:687-737` is a raw per-sample accumulation again; `elevation_profile_test.dart:57-67` asserts `closeTo(10.0, 0.5)` on a true 10 % grade, and `:69-81` asserts strict x monotonicity. Both fail under the reverted-away implementation. |
| CR-03 | Waypoint markers on a mismatched scale | **CLOSED** by the same revert — the axis is a raw haversine sum again, the same scale `Waypoint.distanceFromStart` uses from both producers, so `elevation_profile.dart:197`'s `<= maxDist` filter no longer drops tail waypoints. |
| WR-01 | `<copyright>` without `author` | **CLOSED.** `gpx_reader.dart:618-652` (LOCAL MODIFICATION 5), with a non-vacuity assertion that the published reader really throws (`gpx_conversion_util_test.dart:804`). |
| WR-02 | `parseGpxSafely` not exception-free | **PARTIAL.** `fromString` now throws a documented `FormatException` (`gpx_reader.dart:89-91`) and `TrailLibraryNotifier.build()` guards per row (`trail_library_provider.dart:36-45`), which was the important half. The sibling unguarded call site was missed — see WR-08. |
| WR-03 | `mergeHeightsIntoGpx(source:)` aliasing + multi-track drop | **OPEN.** Re-verified: `identical(merged.wpts, source.wpts) == true`, and appending through the merged document changed `source.wpts.length` from 1 to 2. See WR-04. |
| WR-04 | Import transforms flatten `<trkseg>` structure | **OPEN.** Re-verified: a 2-segment source (3 anchors) becomes 1 segment (2 anchors). See WR-05. |
| WR-05 | Sub-5 m track collapses the x-axis to 0 | **CLOSED** by the chart revert. |
| WR-06 | Vendored source carries no licence | **CLOSED.** `app/lib/vendor/gpx/LICENSE` is byte-identical to upstream's (Apache 2.0, not BSD-3-Clause), `README.md` documents §4(a)/§4(b) compliance. `gpx_tag.dart` is still byte-identical to upstream and unmodified, so §4(b) does not apply to it — see IN-02 for the residual nit. |
| WR-07 | Chart duration semantics changed silently | **CLOSED** by the revert — `elevation_profile.dart:699-704` is adjacent-pair again, so an untimed gap contributes nothing. |
| WR-08 | `_readEmail` attribute/text splice | **OPEN.** Re-verified: `<email domain="a.com">u@b.com</email>` still yields `id=u domain=a.com`. See WR-06. |
| WR-09 | Repo guard is a substring grep | **OPEN.** `gpx_conversion_util_test.dart:894-895` unchanged. See WR-07. |

---

## Critical Issues

### CR-01: A non-finite `lat`/`lon` re-opens the exact zero-distance poisoning the new anchor guard was written to close

**File:** `app/lib/util/gpx_conversion_util.dart:189-191`, `:295-298`
(enabled by `app/lib/vendor/gpx/gpx_reader.dart:427-433`)

**Issue:** the guard added by `867e19b9` tests for `null` only:

```dart
if (point.lat == null || point.lon == null) {
  return;
}
```

but `_readCoordinateAttribute` uses `double.tryParse`, and `double.tryParse` accepts the literals
`NaN`, `Infinity`, `-Infinity` and silently overflows `1e999` to `Infinity`. None of those are
`null`, so all of them pass the guard and become `_lastFilteredPointXY`. From that anchor every
`haversineMeters` is `NaN` (by design — see its doc), `NaN >= _thresholdXYm` is always `false`, so
line 297 never runs and `totalDistanceSmoothed` stays `0.0` for the entire remainder of the file.
That is verbatim the failure mode the guard's own 10-line comment describes.

Measured on the shipped code (one leading `<trkpt lat="NaN" lon="NaN">`, then 20 valid points):

```
tryParse NaN=NaN  1e999=Infinity  Infinity=Infinity
PROBE A distance=0.0  minLat=NaN  centroid=NaN  trailLat=NaN  allWaypoints=21
PROBE A jsonEncode threw: Converting object to an encodable object failed: NaN
PROBE B (lat="1e999") distance=0.0  maxLat=Infinity
```

`NaN` does not stay contained either — it reaches `Trail.lat`, and:

- `app/lib/components/map/trail_layer.dart:81-90` builds the trail's GeoJSON from
  `gpx.allPoints`, which filters `null` but not non-finite, so `jsonEncode` **throws** and the
  trail's route layer silently fails to render;
- `form_data_util.dart:36-37` stringifies it, so the literal `"NaN"` is what gets POSTed as the
  trail's latitude.

`GpxMappingUtils.allWaypoints` (`gpx_util.dart:71`) and `importTrailFile`'s `hasUsablePoint`
(`trail_import_util.dart:89-93`) both test `!= null` only, so nothing upstream rejects the file.

**Fix:** make the guard (and every sibling coordinate filter) test finiteness, not nullness:

```dart
// gpx_conversion_util.dart
bool _hasUsableCoordinates(Wpt p) =>
    p.lat != null && p.lon != null && p.lat!.isFinite && p.lon!.isFinite;

if (!_hasUsableCoordinates(point)) return;
```

and apply the same predicate in `GpxMappingUtils.allWaypoints`, `anchorsFromTrack`,
`segmentPolylinesFromTrack` and `importTrailFile`'s emptiness guard. Cheaper alternative that
fixes it at the source: reject non-finite values in the reader itself, which is where the
tolerance is supposed to live —

```dart
// gpx_reader.dart:_readCoordinateAttribute
final parsed = raw == null ? null : double.tryParse(raw);
return (parsed != null && parsed.isFinite) ? parsed : null;   // LOCAL MODIFICATION 2, extended
```

Add a regression test with `lat="NaN"` at index 0 asserting `distance > 0` and
`trail.lat!.isFinite`.

---

### CR-02: A coordinate-less `<trkpt>`'s `<ele>` is still folded into elevation gain and loss — 2 km of fabricated climb on a flat track

**File:** `app/lib/util/gpx_conversion_util.dart:212-227`, `:248-293`

**Issue:** the new guard is in the **initialisation branch only**. Once the anchors are seeded,
`addAndFilter` runs the elevation accumulators for every subsequent point regardless of whether it
has a position:

```dart
final elevation = parseGpxElevation(point.ele);   // line 212 — no coordinate check
if (elevation != null) { ...totalElevationGain/Loss... }
...
if (elevation != null) { ...defer-then-publish smoothed filter... }   // line 248
```

A `<trkpt>` that the vendored reader accepted *because* it lacks `lat`/`lon` (LOCAL
MODIFICATION 2) still contributes its `<ele>` to the trail's headline elevation numbers. Measured
on a 20-point, perfectly flat track (every point `ele=1000`) with one coordinate-less
`<trkpt ele="3000">` inserted at index 10:

```
PROBE C gain=2000.0  loss=2000.0  distance=422.54
```

`computeTrailMetrics` returns those as `elevationGain`/`elevationLoss`, which become
`Trail.elevationGain`/`elevationLoss` and are persisted on save. The trail reports +2000 m / -2000 m
of climbing that never happened, with a correct-looking distance beside it — the definition of
wrong-but-plausible.

The same point also becomes `_lastFilteredZPointXY` (lines 252, 289), so the subsequent
`returnDistance` is `NaN`, `cancelsPending` can never be true, and the noise filter is disabled
until the next above-threshold move — i.e. the fabricated excursion is not even eligible for
cancellation.

This is not caught by the existing suite: `gpx_conversion_util_test.dart`'s
`trackWithLeadingBadPoint` puts the bad point at index **0**, where the new `return` drops it
before any elevation handling (its `elevationGain` equality assertion therefore passes
vacuously), and `elevation_profile_test.dart:108-124` puts one at index 10 but asserts only on
the chart, which skips it via its own `continue`. Chart and stats now disagree on the same file.

**Fix:** gate elevation on the same predicate as position, so a point with no place on the map
cannot move the trail's elevation:

```dart
void addAndFilter(Wpt point) {
  if (!_hasUsableCoordinates(point)) return;   // before ANY accumulator, not just the seed
  ...
}
```

Add a regression test with the coordinate-less point at a **mid-track** index asserting
`elevationGain == 0` on a flat track, and pin chart/metrics agreement on the same fixture.

---

## Warnings

### WR-01: `trailFromGpx` still takes its start coordinate from an unfiltered `firstOrNull` — a leading malformed point yields a position-less trail

**File:** `app/lib/util/gpx_conversion_util.dart:495-497`

**Issue:** round 2 raised this as part of CR-01; only the distance half was fixed.

```dart
final trackPoints = gpx.trks.firstOrNull?.trksegs.firstOrNull?.trkpts;
final routePoints = gpx.rtes.firstOrNull?.rtepts;
final startPoint = trackPoints?.firstOrNull ?? routePoints?.firstOrNull;
```

Measured on the fix commit's own fixture (one leading coordinate-less `<trkpt>`, 40 valid points):

```
PROBE1 distance=867.3  trailLat=null  trailLon=null
```

The trail now measures correctly but has no map position. `buildLocalTrail`
(`trail_import_util.dart:261-263`) gates the reverse-geocode on `lat != null && lon != null`, so
`location` is silently left empty too. `hasFiniteBounds` still populates the bounding box, so the
trail looks complete everywhere except where it matters.

**Fix:**

```dart
bool _usable(Wpt p) => p.lat != null && p.lon != null && p.lat!.isFinite && p.lon!.isFinite;
final startPoint = trackPoints?.firstWhereOrNull(_usable) ??
    routePoints?.firstWhereOrNull(_usable);
```

(`collection` is already imported in this file.)

### WR-02: The centroid divides by a count that includes points it summed `0` for

**File:** `app/lib/util/gpx_conversion_util.dart:373-391`

**Issue:** `totalLat += point.lat ?? 0` while `summedPointCount++` runs unconditionally, so every
coordinate-less point drags the centroid toward (0, 0). One bad point out of 21 moved it from
47.00 to 44.76 in the mid-track probe; one out of 41 moved it from 47.004 to 45.86 in the
leading-point probe. The error is proportional to the fraction of bad points and is unbounded.

Two things make this worse than it looks:

- `GpxTrailMetrics.centroidLat`/`centroidLon` are **never read in `app/lib/`** (grep: only
  `gpx_conversion_util.dart` itself and two test files). It is a public field of a new value class
  carried purely for TS parity — so the error is invisible in the app but *is* pinned by
  `gpx_corpus_test.dart:132-139` and its TS counterpart, and no corpus fixture contains a
  coordinate-less point, so the parity suite cannot detect the divergence either.
- The TS side is not equivalent: `web/src/lib/models/gpx/waypoint.ts:96-97` falls a missing
  `lat`/`lon` back to `-1`, so TS sums `-1` where Dart sums `0`. The two ports produce different
  centroids, different bounding boxes and different distances for the same malformed file, and
  nothing pins it.

**Fix:** count the coordinates you actually summed, and add a corpus fixture with a malformed
`<trkpt>` so the TS/Dart divergence becomes visible:

```dart
if (point.lat != null && point.lon != null) {
  totalLat += point.lat!;
  totalLon += point.lon!;
  summedCoordinateCount++;      // centroid divides by this
}
summedPointCount++;             // pointCount stays a raw count
```

### WR-03: A `<wpt>` with no coordinates becomes a waypoint at (0, 0), and is re-serialised as an attribute-less `<wpt>`

**File:** `app/lib/util/gpx_conversion_util.dart:477-493`

**Issue:** `lat: wpt.lat ?? 0, lon: wpt.lon ?? 0`. Before LOCAL MODIFICATION 2 such a file threw
and the import failed loudly; now it succeeds and produces a waypoint in the Gulf of Guinea that
is uploaded with the trail on save. Verified:

```
PROBE F waypoint name=Summit lat=0.0 lon=0.0
PROBE F waypoint name=Hut    lat=47.1 lon=11.1
```

Compounding it, when either post-capture toggle is on the document is re-serialised
(`trail_import_util.dart:170`) and `GpxWriter` emits the waypoint with no attributes at all —
`<wpt><name>Summit</name></wpt>` — which is what gets persisted as the trail's track file:

```
PROBE G out=...<wpt><name>Summit</name></wpt>...
```

**Fix:** drop waypoints with no usable coordinate rather than inventing one, and say so:

```dart
for (final wpt in gpx.wpts)
  if (wpt.lat != null && wpt.lon != null && wpt.lat!.isFinite && wpt.lon!.isFinite)
    Waypoint(...)
```

### WR-04: `mergeHeightsIntoGpx(source:)` aliases the source's mutable collections, and silently drops every track but the first

**File:** `app/lib/util/route_planner_handoff_util.dart:62-74` (round-2 WR-03, still open)

**Issue:** re-verified against current code.

```dart
gpx.metadata = source.metadata;
gpx.wpts = source.wpts;      // reference, not copy
gpx.rtes = source.rtes;      // reference, not copy
gpx.extensions = source.extensions;
```

```
PROBE3 mergedTrks=1  identicalWpts=true  identicalRtes=true
PROBE3 after mutating merged.wpts, source.wpts.length=2   (was 1)
```

Two `Gpx` objects share one `List<Wpt>`, one `List<Rte>`, one `Metadata` and one extensions map.
Latent today, but `trail_import_util.dart` hands both documents to the same downstream code, so
any future edit through one silently rewrites the other.

Separately, `sourceTrk = source.trks.first` (lines 72-74) narrows a multi-track document to one
track's metadata while `gpx.allPoints` (which built `shape` at `trail_import_util.dart:122-126`)
flattened **all** of them into the geometry. The doc comment does not state this narrowing.

**Fix:** `gpx.wpts = [...source.wpts]; gpx.rtes = [...source.rtes];` and
`gpx.extensions = Map.of(source.extensions)`; document the single-track narrowing on the
parameter, or carry the remaining tracks through.

### WR-05: The import path's post-capture transforms flatten every `<trkseg>` into one, destroying planner anchor structure

**File:** `app/lib/util/trail_import_util.dart:120-170` (round-2 WR-04, still open)

**Issue:** re-verified. `workingShape` comes from `gpx.allPoints`, which `gpx_util.dart:75-79`
flattens across every `Trk` and `Trkseg`; `mergeHeightsIntoGpx` then emits exactly one `Trk` with
one `Trkseg`, and line 170 re-serialises that as the uploaded track file.

```
PROBE4 sourceAnchors=3  ->  mergedSegs=1  mergedAnchors=2
```

`buildFinalPlannedGpx`'s own doc comment (`route_planner_handoff_util.dart:336-341`) states this
consequence explicitly for the planner — "collapsed every intermediate anchor to a bare start/end
pair on re-edit". The import path does exactly that to any multi-segment GPX, including one
Wanderer's own planner produced, the moment the user ticks either box.

**Fix:** transform per segment and reassemble one `Trkseg` per source segment; or, minimally, skip
the transforms for a multi-segment source and document the narrowing on `importTrailFile`.

### WR-06: `_readEmail`'s partial-attribute fallback splices an attribute domain onto a text-form local part

**File:** `app/lib/vendor/gpx/gpx_reader.dart:727-737` (round-2 WR-08, still open)

**Issue:** re-verified against the shipped reader — the fallback fires when *either* attribute is
missing, then fills each field independently, so `<email domain="a.com">u@b.com</email>` yields:

```
PROBE2 id=u  domain=a.com     -> "u@a.com", an address present in neither form
```

This contradicts LOCAL MODIFICATION 4's own comment ("Attributes still win when present; the text
form is only consulted as a fallback"). Impact stays low — nothing in `app/lib/` reads
`Metadata.author.email` — but it is a mis-parse in the one function whose job is parsing, and it
will propagate to any future consumer.

**Fix:** treat the two forms as mutually exclusive.

```dart
if (id == null && domain == null) {   // only when the attribute form is absent entirely
  ...
}
```

### WR-07: The single-call-site repo guard is a substring grep and does not cover the wrong-import case

**File:** `app/test/util/gpx_conversion_util_test.dart:894-895` (round-2 WR-09, still open)

**Issue:** unchanged — the gate is `line.contains('GpxReader(')` on lines not starting with `//`.
It misses a construction split across lines, `GpxReader ()` with whitespace, and
`const c = GpxReader; c();`. It also does not address the inverse risk it is really guarding
against: the vendored and published classes are name-identical, so a new file with a bare
`import 'package:gpx/gpx.dart';` (no `hide GpxReader`) is silently opted out of the tolerance
while reading as correct. Nine files in `lib/` currently import that package without `hide` —
only `gpx_conversion_util.dart` uses it.

**Fix:** gate on the import instead. Assert that every `import 'package:gpx/gpx.dart'` under
`lib/` outside `lib/vendor/gpx/` carries `hide GpxReader`, and that
`lib/vendor/gpx/gpx_reader.dart` is imported by exactly one file. Keep the existing
`dartFilesScanned > 100` non-vacuity floor.

### WR-08: `TrailNotifier`'s offline fallback calls `toModel()` unguarded — the sibling of the call site that was fixed

**File:** `app/lib/provider/trail/trail_provider.dart:79`

**Issue:** `cf1ad326` wrapped `toModel()` per row in `TrailLibraryNotifier.build()`, which was the
important half of round-2 WR-02. The other call site was not touched:

```dart
final entity = query.findFirst();
query.close();
if (entity != null) return entity.toModel();   // can throw FormatException
rethrow;
```

`toModel()` parses the cached GPX (`trail_entity.dart:190`) and, per this phase's own
`trail_entity_test.dart:60-70`, **throws** `FormatException` for a corrupt `gpxData`. Inside a
`catch (_)` block, that throw replaces the real (network) failure with a parse error, so the trail
screen reports a GPX problem for what is actually an offline/server condition — and the user gets
nothing at all for a trail whose metadata is perfectly readable.

**Fix:** degrade to metadata-only rather than losing the row:

```dart
if (entity != null) {
  try {
    return entity.toModel();
  } catch (e, st) {
    debugPrint('TrailNotifier: cached trail "$id" failed toModel(): $e\n$st');
  }
}
rethrow;
```

Better still, give `parseGpxSafely` a `tryParseGpxSafely` sibling returning `null` and use it in
`toModel()` so a corrupt cache costs one track preview, not one whole trail.

---

## Info

### IN-01: `buildElevationTrackPoints` traverses the track twice for an emptiness check the loop already covers

**File:** `app/lib/components/trail/elevation_profile.dart:662`, `:722`

**Issue:** `if (gpx.allWaypoints.isEmpty) return [];` materialises a full filtered `List<Wpt>`
purely to decide whether to proceed, and is then made redundant by `if (result.isEmpty) return [];`
at line 722 — which is the authoritative check, since it applies the same coordinate filter. The
early return can only ever agree with it.

**Fix:** delete line 662 and keep the `result.isEmpty` guard.

### IN-02: `gpx_tag.dart` carries no per-file provenance header

**File:** `app/lib/vendor/gpx/gpx_tag.dart:1`

**Issue:** the file is byte-identical to upstream (verified by `diff` — zero output), so Apache 2.0
§4(b)'s modified-file notice requirement does not apply and `README.md` covers the directory.
But nothing *in* the file identifies it as third-party, which is the one thing a reader who opens
it directly needs.

**Fix:** prepend one line — `// Vendored verbatim from package:gpx 2.3.0 (lib/src/model/gpx_tag.dart) — Apache 2.0, see LICENSE.`

---

_Reviewed: 2026-08-01T14:20:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
_Round: 3 — verification of the round-2 closures, regression hunt on the chart revert and the anchor-seed guard, and a completeness audit of the vendored reader's six local modifications_
