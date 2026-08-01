---
phase: 34-dart-conversion-port
reviewed: 2026-08-01T10:42:31Z
depth: standard
round: 2
files_reviewed: 18
files_reviewed_list:
  - app/lib/util/gpx_conversion_util.dart
  - app/lib/util/gpx_util.dart
  - app/lib/util/trail_import_util.dart
  - app/lib/util/route_planner_handoff_util.dart
  - app/lib/util/track_save_options_util.dart
  - app/lib/vendor/gpx/gpx_reader.dart
  - app/lib/vendor/gpx/gpx_tag.dart
  - app/lib/components/trail/elevation_profile.dart
  - app/lib/provider/trail/trail_provider.dart
  - app/lib/provider/trail/trail_library_provider.dart
  - app/lib/entities/trail_entity.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/routes/route_planner_screen.dart
  - app/test/util/gpx_util_test.dart
  - app/test/util/gpx_conversion_util_test.dart
  - app/test/util/gpx_corpus_test.dart
  - app/test/components/trail/elevation_profile_test.dart
  - web/src/routes/api/v1/trail/convert/+server.ts
  - web/src/routes/api/v1/trail/convert/convert.test.ts
findings:
  critical: 3
  warning: 9
  info: 0
  total: 12
status: issues_found
---

# Phase 34: Code Review Report (Round 2)

**Reviewed:** 2026-08-01T10:42:31Z
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found
**Diff base:** `fb381452..HEAD` (20 commits)

## Summary

All four round-1 criticals and all twelve warnings are genuinely closed — verified against
current code and, where behaviour was claimed, verified by running the shipped functions rather
than trusting the commit message. Details in "Closure verification" below.

However **the fix pass introduced three new defects**, all in the two commits the brief flagged
as highest risk (`738a06a2` vendored reader, `afb2b434` haversine + elevation-profile rewrite).
Two are in the elevation-profile rewrite; one is the downstream consequence of the vendored
reader's null-lat/lon tolerance meeting an accumulator that was never taught about null
coordinates. All three were confirmed empirically:

| Probe | Result |
|---|---|
| ~2 km track whose FIRST `<trkpt>` lacks `lat`/`lon` | `trail.distance == 0.0`, `trail.lat == null`, centroid 44.66 (should be ~47.01) |
| 60-point track at 1.5 m spacing on a true 10 % grade | 46/60 chart points report gradient `0.0 %`; the other 14 report `2.5 %` |
| corpus `08-jittery-track` | chart x-axis total 100.1 m vs. the `distance_from_start` scale's 110.1 m — a 10 % mismatch |

The vendored reader's four local modifications are individually correct, and `diff` against
`~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/gpx_reader.dart` confirms nothing undeclared was
changed. But the tolerance set is **incomplete**: `_readCopyright` still throws `StateError` on
a `<copyright>` with no `author` attribute, killing an entire otherwise-valid document — the
exact failure class this phase set out to eliminate (WR-01). And `parseGpxSafely` is not
exception-free at all: it throws `_TypeError` on any input without a `<gpx>` root, and one of
its three call sites is unguarded and runs once per cached trail (WR-02).

## Closure verification (round 1)

| ID | Claim | Verdict |
|---|---|---|
| CR-01 | `mergeHeightsIntoGpx` gained `Gpx? source` and merges into it | **CLOSED.** `route_planner_handoff_util.dart:62-85` copies `version`/`creator`/`metadata`/`wpts`/`rtes`/`extensions` and the first `<trk>`'s metadata; `trail_import_util.dart:160-166` passes `source: gpx`. The recording path (`navigation_screen.dart:809,816`) correctly omits it. Residual in WR-03. |
| CR-02 | Regex sanitizers deleted; tolerance moved into a vendored reader | **CLOSED.** No `sanitizeGpx*` remains anywhere in `app/`. `diff` vs. pub-cache `gpx-2.3.0` shows exactly the 4 documented modifications plus `dart format` churn. Models still come from the published package via `hide GpxReader`, so `Gpx`/`Wpt` types stay shared with `GpxWriter`. |
| CR-03 | `trail_provider`/`trail_entity` route through `parseGpxSafely`; repo-guard test | **CLOSED.** `trail_provider.dart:49`, `trail_entity.dart:190`. The gate at `gpx_conversion_util_test.dart:802-851` has a real non-vacuity floor (`dartFilesScanned > 100`) and pins the sole site's path. Blind spots in WR-09. |
| CR-04 | Convert endpoint validates via `GPX.parse`; `nosniff` + `Content-Disposition` | **CLOSED.** `+server.ts:24-39, 117-128`. Verified `GPX.parse` genuinely throws for a non-`<gpx>` root (`xml.gpx.$` on `undefined`), so the HTML/XHTML/JSON reflector cases in the suite really are rejected. `gpx2trail` and reverse-geocoding are not reintroduced. Also probed the parser for an entity-expansion (billion-laughs) surface: `sax` rejects custom `<!ENTITY>` outright, so none was opened. |
| WR-01 | `segmentPolylinesFromTrack` fallback emits the outstanding pair | **CLOSED** (`route_planner_handoff_util.dart:668`, `k = i > 0 ? i - 1 : 0`). |
| WR-02 | `fallbackShape` = full-resolution geometry | **CLOSED** (`snapShapeToRoadsResult:204`; call sites `navigation_screen.dart:797`, `trail_import_util.dart:139`). |
| WR-03 | Leg elevations kept on a silent snap fallback | **CLOSED** (`buildFinalPlannedGpx:428`, `if (!snapped[k].snapped) continue;`). |
| WR-04 | Double-tap guard re-checked after the gate await | **CLOSED** (`route_planner_screen.dart:534`, `if (!mounted || _finishing) return;`). |
| WR-05 | Corpus icon assertion made non-tautological | **CLOSED** (`gpx_corpus_test.dart:339-349` pins `fontAwesomeIconsMap[sym] != null` independently of the production expression). |
| WR-06 | TS `<time>` semantics aligned with the Dart port | **CLOSED** (`web/src/lib/models/gpx/waypoint.ts`, `DART_DATETIME_GRAMMAR` + `parseGpxTime`, replacing the bare `new Date(...)`). |
| WR-07 | Planner height backfill gated on connectivity | **CLOSED** (`buildFinalPlannedGpx:453-461`, `!online ? const <int>[] : ...`). |
| WR-08 | Zero moving duration mapped to "no value" | **CLOSED** (`trailFromGpx:500-503`, `movingDuration.inSeconds > 0`; sub-second case tested). |
| WR-09 | `moving_duration` accepted by the JSON trail request schemas | **CLOSED** (`trail_schema.ts:25, 56`). |
| WR-10 | Stale `movingDuration` cleared when geometry is replaced | **CLOSED** (`mergeRouteIntoTrail:718`, `movingDuration: 0`, with the Freezed-can't-assign-null rationale documented). |
| WR-11 | Zero-point leg skipped rather than emitted as an empty `Trkseg` | **CLOSED** (`buildFinalPlannedGpx:494`). |
| WR-12 | `importTrailFile`'s `try` narrowed; exception logged | **CLOSED** (`trail_import_util.dart:81-193`; the push is outside the `try`, `debugPrint` carries `$e\n$st`). |

---

## Critical Issues

### CR-01: A `<trkpt>` with null `lat`/`lon` as the FIRST point permanently freezes the smoothing anchor — the whole trail measures 0 m

**File:** `app/lib/util/gpx_conversion_util.dart:170-185, 279-282` (root cause enabled by `app/lib/vendor/gpx/gpx_reader.dart:293-294`)

**Issue:** `addAndFilter`'s initialisation branch seeds both XY anchors with whatever the first
point is, guarding only elevation:

```dart
if (_lastPointXY == null || _lastFilteredPointXY == null) {
  _lastPointXY = point;
  _lastFilteredPointXY = point;          // <-- no coordinate guard
  final initialElevation = parseGpxElevation(point.ele);
  if (initialElevation != null) { ... }  // elevation IS guarded
  return;
}
```

`_lastFilteredPointXY` is only ever reassigned inside `if (smoothedDistance >= _thresholdXYm)`
(line 279). If that anchor has a null coordinate, `haversineMeters` returns `double.nan` for
every subsequent point (deliberately, per its own doc), `NaN >= 5` is always `false`, so the
anchor is **never replaced** and `totalDistanceSmoothed` stays `0.0` for the entire remainder of
the file. `totalDistanceSmoothed` is exactly what `computeTrailMetrics` returns as `distance`,
which becomes `trail.distance`.

This path did not exist before this phase. LOCAL MODIFICATION 2 of the vendored reader is what
turned "a `<trkpt>` missing `lat`/`lon` throws `StateError`" into "it yields
`Wpt(lat: null, lon: null)`". `elevation_profile_test.dart:77-92` covers a null-coordinate point
inserted at **index 10** — never at index 0 — so the suite stays green.

Measured on the shipped code (one leading coordinate-less `<trkpt>`, then 19 valid points
spanning ~2.1 km):

```
DISTANCE=0.0  pointCount=20  centroidLat=44.6595
TRAIL lat=null lon=null distance=0.0
```

Two further consequences of the same root cause, both confirmed by the same probe:

- `trailFromGpx:481` takes `startPoint` from `trackPoints?.firstOrNull` (unfiltered), so
  `trail.lat`/`trail.lon` are `null` — the trail has no map position, and `buildLocalTrail`
  skips the reverse-geocode entirely because of its own `lat != null && lon != null` gate.
- `computeTrailMetrics:361-363` does `totalLat += point.lat ?? 0` while incrementing
  `summedPointCount` unconditionally, so each null-coordinate point drags the centroid toward
  (0, 0) — 47.01 → 44.66 with a single bad point out of 20.

`importTrailFile`'s `hasUsablePoint` guard (line 89) does not catch this: it passes as soon as
*any* point has coordinates. So the import completes, the user is navigated to the create
screen, and a 0-metre, position-less trail is offered for saving.

**Fix:** never let a coordinate-less point become an XY anchor, and exclude it from the centroid.

```dart
void addAndFilter(Wpt point) {
  final hasXY = point.lat != null && point.lon != null;

  if (_lastPointXY == null || _lastFilteredPointXY == null) {
    if (hasXY) {
      _lastPointXY = point;
      _lastFilteredPointXY = point;
    }
    final initialElevation = parseGpxElevation(point.ele);
    if (initialElevation != null) {
      _lastFilteredZ = initialElevation;
      _lastFilteredZPointXY = hasXY ? point : null;
      _lastZ = initialElevation;
    }
    return; // a coordinate-less leading point simply does not anchor; the next one retries
  }
  ...
}
```

and in `computeTrailMetrics`:

```dart
if (point.lat != null && point.lon != null) {
  totalLat += point.lat!;
  totalLon += point.lon!;
  summedCoordinateCount++;   // divide the centroid by THIS
}
summedPointCount++;          // pointCount stays a raw point count
```

and in `trailFromGpx:479-481`:

```dart
final startPoint =
    trackPoints?.firstWhereOrNull((p) => p.lat != null && p.lon != null) ??
    routePoints?.firstWhereOrNull((p) => p.lat != null && p.lon != null);
```

Add a regression test with the coordinate-less point at index **0**, asserting `distance > 0`
and `trail.lat != null`.

---

### CR-02: The elevation chart's gradient divides a one-sample elevation delta by a multi-sample smoothed distance jump — it reads ~0 % on a real climb

**File:** `app/lib/components/trail/elevation_profile.dart:703-730` (introduced by `afb2b434`)

**Issue:** `distanceM` is now `metrics.totalDistanceSmoothed`, which is a **step function** — it
only advances when a point is ≥ 5 m from the last smoothing anchor. `elevationM`, by contrast,
still advances on every point. The gradient loop divides one by the other:

```dart
final dElev = result[i].elevationM - result[i - 1].elevationM;  // per-sample
final dDist = result[i].distanceM - result[i - 1].distanceM;    // 0, or a ~5 m jump
result[i].gradient = dDist > 0 ? (dElev / dDist) * 100 : 0;
```

For any track sampled denser than 5 m — i.e. every 1 Hz recording, and anything Strava/Garmin
exports — most consecutive pairs have `dDist == 0` and get gradient `0`, while the pair that
does step gets a ~5 m denominator against a **single sample's** elevation change. The result is
understated by roughly the number of samples per 5 m.

Measured on the shipped `buildElevationTrackPoints` with 60 points at ~1.5 m spacing on a true
**10 %** grade:

```
POINTS=60  ZEROGRAD=46  maxDist=84.06
grads=[0.0, 0.0, 0.0, 0.0, 2.5, 0.0, 0.0, 0.0, 2.5, 0.0, 0.0, 0.0]
```

77 % of points read `0.0 %`; the rest read `2.5 %` on a 10 % slope. This is not cosmetic —
`gradient` drives three user-visible things:

- `_gradientColor` (lines 424, 464, 729), which is the entire colour of the chart line and its
  fill. Both `0 %` and `2.5 %` land in the `g < 3` "soft purple / flat" bucket, so a sustained
  10 % climb now renders flat purple end to end instead of amber/orange. The gradient colouring
  is the chart's signature visual.
- the scrub stats header (line 116), which prints `+0.0%` for roughly 3 of every 4 scrub
  positions.
- `getTouchedSpotIndicator`'s dot colour (line 342).

The line-gradient dedup at lines 422/462 (`if (stop <= stops.last) continue`) hides the zeros
from the stops array, which is likely why this survived visual inspection — but it cannot fix
the magnitude, and the tooltip and dot paths have no such dedup. No test asserts on `gradient`.

**Fix:** keep the smoothed distance for the axis (that is the whole point of the change), but
compute the gradient against the distance actually travelled between the two plotted points.
Carry a raw cumulative distance alongside:

```dart
class TrackPoint {
  final double distanceM;      // smoothed — axis + stats, unchanged
  final double rawDistanceM;   // raw cumulative — gradient denominator only
  ...
}
```

accumulate `rawDistanceM` with `haversineMeters` over the same traversal, then:

```dart
final dDist = result[i].rawDistanceM - result[i - 1].rawDistanceM;
result[i].gradient = dDist > 0 ? (dElev / dDist) * 100 : 0;
```

Add a test asserting that a synthetic constant-10 %-grade track sampled below the threshold
reports gradients near 10, not near 0.

---

### CR-03: Waypoint markers are positioned by a raw `distance_from_start` against a now-smoothed x-axis — they drift, and tail waypoints are silently filtered out

**File:** `app/lib/components/trail/elevation_profile.dart:195-199, 366, 381`

**Issue:** the chart's x-axis (`maxDist = _points.last.distanceM`, line 80) is now the
**smoothed** total, but `Waypoint.distanceFromStart` is a **raw** cumulative distance from two
independent producers, neither of which changed:

- server-side: `web/src/lib/vendor/maplibre-elevation-profile/elevationprofile.ts:965` assigns
  `cumulatedDistanceAdjustedUnit[i] * 1000` — a raw per-point accumulation;
- app-side: `GpxMappingUtils.distanceFromStartTo` (`gpx_util.dart:107`) sums `haversineMeters`
  over every consecutive vertex — explicitly raw, and the commit that unified the haversines
  deliberately kept it raw.

Before `afb2b434` the chart axis was also a raw haversine sum over `allWaypoints`, so the two
scales matched by construction. They no longer do. Two symptoms:

1. **Drift.** Every marker sits too far right by the raw/smoothed ratio, growing along the
   track. Measured on corpus `08-jittery-track`: raw 110.1 m vs. smoothed 100.1 m — a 10 % error
   at the far end. The commit's own test fixture is 232.6 m vs. 80.6 m (2.9×).
2. **Silent disappearance.** Line 197 filters `w.distanceFromStart! <= maxDist`. Any waypoint
   whose raw distance exceeds the smoothed total — for a jittery track that is *every* waypoint
   past the crossover point; for a clean track it is at minimum an end-of-trail waypoint — is
   dropped from `waypoints` entirely: no vertical line (line 366), no icon marker (line 380).
   The user sees fewer waypoints on the profile than the trail has, with no indication why.

**Fix:** plot markers on the same scale as the axis. Both scales are monotonic over the same
traversal, so with `rawDistanceM` on `TrackPoint` (see CR-02's fix) the mapping is a lookup:

```dart
double smoothedXFor(double rawDistance) {
  for (var i = 1; i < _points.length; i++) {
    if (_points[i].rawDistanceM >= rawDistance) {
      final p0 = _points[i - 1], p1 = _points[i];
      final span = p1.rawDistanceM - p0.rawDistanceM;
      final t = span > 0 ? (rawDistance - p0.rawDistanceM) / span : 0.0;
      return p0.distanceM + t * (p1.distanceM - p0.distanceM);
    }
  }
  return _points.last.distanceM;   // clamp, do not drop
}
```

and replace the `<= maxDist` filter with a clamp so a tail waypoint pins to the right edge
rather than vanishing. Add a test with a waypoint whose `distanceFromStart` exceeds the smoothed
total, asserting it is still rendered.

---

## Warnings

### WR-01: The vendored reader's tolerance set is incomplete — `<copyright>` without an `author` attribute still aborts the whole document

**File:** `app/lib/vendor/gpx/gpx_reader.dart:590-593`

**Issue:** `_readCopyright` was carried over from upstream unchanged:

```dart
copyright.author = elm.attributes
    .firstWhere((attr) => attr.name == GpxTagV11.author)
    .value;
```

No `orElse`. Confirmed against the shipped code, on an otherwise perfectly valid track carrying
`<metadata><copyright><year>2024</year></copyright></metadata>`:

```
copyright no author => THREW StateError: Bad state: No element
```

This is the *same* user-visible failure the phase set out to eliminate: in `trail_provider.dart`
the throw is swallowed by the broad `catch (_)` at line 60 and silently degrades to the offline
cache — exactly the behaviour the CR-03 commit comment describes for `<hdop></hdop>`.

The file header asserts the numbered list is the complete tolerance set and that behaviour is
"otherwise byte-for-byte upstream". That is literally true, but it leaves a throwing
`firstWhere` on a required-by-spec attribute of an optional element the app never reads — which
is precisely the shape of LOCAL MODIFICATIONS 2 and 3.

**Fix:** reuse the same pattern and record it as LOCAL MODIFICATION 5:

```dart
// LOCAL MODIFICATION 5: <copyright author> is required by spec, but <copyright> is
// optional metadata the app never reads — losing it must not cost the track.
final authorAttr =
    elm.attributes.firstWhereOrNull((attr) => attr.name == GpxTagV11.author);
copyright.author = authorAttr?.value ?? '';
```

Add a test next to the existing "`<trkpt>` missing lat/lon" case, including the
published-reader-throws non-vacuity assertion the neighbouring tests use.

### WR-02: `parseGpxSafely` is not exception-free, and one of its three call sites is unguarded and runs once per cached trail

**Files:** `app/lib/util/gpx_conversion_util.dart:43-45`, `app/lib/entities/trail_entity.dart:185-190`, `app/lib/provider/trail/trail_library_provider.dart:29`

**Issue:** `parseGpxSafely`'s doc presents it as "the single sanctioned parse entry point …
[malformed inputs] now degrade to 'no reading' on the affected field and leave the rest of the
track intact", and `trail_entity.dart:187-189` claims the fix means a cached trail is no longer
"permanently un-openable offline". Both overstate what was fixed. Confirmed against the shipped
function:

```
non-gpx root    => THREW _TypeError: Null check operator used on a null value
html error page => THREW _TypeError: Null check operator used on a null value
plain text      => THREW _TypeError: Null check operator used on a null value
empty string    => THREW _TypeError: Null check operator used on a null value
```

The throw is `gpx_reader.dart:70`, `iterator.current as XmlStartElementEvent`, reached when the
scan loop never finds a `<gpx>` element. `parseEvents` is also lazy, so genuinely malformed XML
raises `XmlParserException` mid-iteration.

`TrailEntity.toModel()` calls it with no guard, and `TrailLibraryNotifier.build()` maps
`toModel()` over **every** row in the account's library inside a synchronous provider `build()`
with no try/catch. One unparseable cached `gpxData` therefore fails the entire offline library
screen — a strictly *worse* blast radius than the single-trail failure the comment claims to
have fixed.

**Fix:** make the contract true and guard the call site:

```dart
/// Returns `null` when [xml] is not a GPX document at all (no `<gpx>` root, or
/// malformed XML). Field-level coercion failures still degrade to "no reading".
Gpx? tryParseGpxSafely(String xml) {
  try {
    return GpxReader().fromString(xml);
  } catch (_) {
    return null;
  }
}
```

then `trail_entity.dart:190` becomes `gpx: gpxData != null ? tryParseGpxSafely(gpxData!) : null`,
so a corrupt cache entry costs one trail's track preview rather than the whole library.

### WR-03: `mergeHeightsIntoGpx(source:)` aliases the source's mutable collections and drops every track but the first

**File:** `app/lib/util/route_planner_handoff_util.dart:62-85`

**Issue:** two things.

1. `gpx.metadata = source.metadata; gpx.wpts = source.wpts; gpx.rtes = source.rtes;
   gpx.extensions = source.extensions;` assigns **references**, not copies. Two `Gpx` objects now
   share one `List<Wpt>`, one `List<Rte>`, one `Metadata` and one extensions map. Nothing mutates
   them on today's paths, so this is latent rather than active — but it is an invisible landmine
   for any future caller that edits waypoints on the returned document and unknowingly mutates
   the source.
2. `sourceTrk` is `source.trks.first` only (lines 72-74). A GPX with multiple `<trk>` elements
   loses every track after the first, and its metadata, even though `gpx.allPoints` — which built
   `shape` at `trail_import_util.dart:122` — flattened **all** of them into the geometry. The doc
   comment does not mention this narrowing.

**Fix:** copy the collections (`gpx.wpts = [...source.wpts]`, likewise `rtes`; `Map.of(...)` for
`extensions`), and either document the single-track narrowing explicitly or carry the remaining
tracks' metadata through.

### WR-04: The import path's post-capture transforms flatten every `<trkseg>` into one segment, destroying route-planner anchor structure

**File:** `app/lib/util/trail_import_util.dart:122-170`

**Issue:** when either toggle is on, `workingShape` is built from `gpx.allPoints`, which
`gpx_util.dart:75-79` flattens across every `Trk` and every `Trkseg`. `mergeHeightsIntoGpx` then
emits a **single** `Trk` with a **single** `Trkseg`, and the re-serialised `finalGpxData` (line
170) is what gets uploaded and persisted.

`buildFinalPlannedGpx`'s own doc comment (`route_planner_handoff_util.dart:336-341`) states this
exact consequence for the planner: "Emitting a single flattened segment here (as an earlier
version did, via `buildNavShape` + `mergeHeightsIntoGpx`) collapsed every intermediate anchor to
a bare start/end pair on re-edit." The import path now does precisely that to any imported
multi-segment GPX — including one Wanderer's own planner produced — the moment the user ticks
either box. `anchorsFromTrack` (line 595) derives anchors from `trkseg` boundaries, so a later
planner edit sees only two anchors.

**Fix:** preserve segment structure through the transform — build `workingShape` per segment,
transform each, and reassemble one `Trkseg` per source segment. At minimum, skip the transforms
for a multi-segment source and document the narrowing on `importTrailFile`.

### WR-05: The chart's x-axis range collapses to exactly zero for any track shorter than the 5 m smoothing threshold

**File:** `app/lib/components/trail/elevation_profile.dart:80, 230-231`

**Issue:** `maxDist = _points.last.distanceM` is now the smoothed total, which is exactly `0.0`
whenever no point ever reaches 5 m from the start anchor. Confirmed on a 4-point, ~2 m track:

```
SHORT maxDist=0.0
```

`LineChartData(minX: 0, maxX: 0)` hands fl_chart a zero horizontal range to divide by when
mapping spots to pixels. `_niceInterval`, `_buildLineGradient` and `_buildFillGradient` all have
explicit zero guards, and `(0.0/0.0).clamp(0.0, 1.0)` returns `1.0` rather than `NaN` (verified),
so this is degraded rendering rather than a crash — but the whole chart collapses onto a single
x-position. Previously only a track whose points were all *identical* could reach this; now any
sub-5 m track does, and CR-01 makes it reachable for arbitrarily long tracks too.

**Fix:** guard the render path, e.g. `final axisMax = maxDist > 0 ? maxDist : 1.0;` passed to
`LineChartData.maxX` and `_niceInterval`, or return `_EmptyState` when `maxDist == 0`.

### WR-06: Vendored BSD-3-Clause source carries no licence file, and `gpx_tag.dart` has no attribution at all

**Files:** `app/lib/vendor/gpx/gpx_tag.dart:1`, `app/lib/vendor/gpx/` (no `LICENSE`)

**Issue:** `gpx_reader.dart` names the licence in a comment but does not reproduce the notice.
`gpx_tag.dart` is **byte-identical to upstream** (verified — `diff` produced zero output) and
carries no header whatsoever, so nothing in the file identifies it as third-party code.
BSD-3-Clause requires source redistributions to retain the copyright notice, the condition list
and the disclaimer. The repo already has the right pattern one directory over:
`app/vendor/tiptap_flutter/LICENSE`.

**Fix:** add `app/lib/vendor/gpx/LICENSE` containing
`~/.pub-cache/hosted/pub.dev/gpx-2.3.0/LICENSE` verbatim, and prepend a
"Vendored from package:gpx 2.3.0 — BSD-3-Clause, see LICENSE" header to `gpx_tag.dart`.

### WR-07: `buildElevationTrackPoints` silently changed the chart's duration semantics — untimed gaps are now bridged

**File:** `app/lib/components/trail/elevation_profile.dart:665-697`

**Issue:** the rewrite replaced `prevTime = rawPoints[i - 1].time` with a `prevTimed` cursor that
holds the last point which *had* a time:

```dart
final prevTime = prevTimed?.time;
...
if (wpt.time != null) prevTimed = wpt;
```

Old behaviour: a gap spanning any untimed point contributed **nothing** to `cumDuration`.
New behaviour: the whole gap is added when the next timed point arrives. For a track with sparse
or partially-unparseable `<time>` elements — a class the vendored reader now produces *more* of,
since an unparseable `<time>` becomes `null` instead of aborting the parse — the chart's
displayed total duration changes. This may well be the better behaviour, but it is an
undocumented, untested side effect of a commit whose stated scope was distance; neither the
commit message nor the in-file comment mentions it.

**Fix:** either restore the adjacent-pair semantics or state the change in the comment and pin
it with a test (e.g. a 5-point track where points 2–4 carry no `<time>`, asserting the expected
total).

### WR-08: `_readEmail`'s partial-attribute fallback can splice an attribute domain onto a text-form local part, fabricating an address present in neither form

**File:** `app/lib/vendor/gpx/gpx_reader.dart:693-703`

**Issue:** the fallback fires when *either* attribute is missing, then fills each field
independently:

```dart
if (id == null || domain == null) {
  final at = text.trim().lastIndexOf('@');
  if (at > 0 && at < text.trim().length - 1) {
    final trimmed = text.trim();
    id ??= trimmed.substring(0, at);
    domain ??= trimmed.substring(at + 1);
  }
}
```

For `<email domain="a.com">u@b.com</email>` this yields `id = "u"`, `domain = "a.com"` — i.e.
`u@a.com`, an address that appears in neither the attributes nor the text. Verified:

```
EMAIL id=u domain=a.com
```

This contradicts the modification's own comment ("Attributes still win when present; the text
form is only consulted as a fallback"). Impact is low — nothing in the app reads
`Metadata.author.email` — but it is a mis-parse in a function whose sole purpose is correct
parsing, and it will silently propagate to any future consumer.

**Fix:** treat the two forms as mutually exclusive, so a partially-specified attribute form is
never completed from the text form:

```dart
if (id == null && domain == null) {   // only when the attribute form is absent entirely
  ...
}
```

### WR-09: The single-call-site repo guard is a plain substring grep and is trivially bypassed

**File:** `app/test/util/gpx_conversion_util_test.dart:813-851`

**Issue:** the gate scans for the literal `GpxReader(` on lines that do not start with `//`.
It misses:

- a construction split across lines (`GpxReader` / `()`), or `const r = GpxReader; r();`
- any whitespace between the identifier and the paren
- the inverse risk it does not address at all: a new file importing the **published** reader
  instead of the vendored one. The two classes are name-identical, so a stray
  `import 'package:gpx/gpx.dart';` without `hide GpxReader` silently opts that file out of the
  tolerance while looking correct.

It also relies on `Directory('lib')` resolving relative to `app/`, guarded only by a `> 100`
file-count heuristic and a `reason:` string.

**Fix:** gate on the import rather than the constructor — assert that every
`import 'package:gpx/gpx.dart'` in `lib/` (outside `lib/vendor/gpx/`) carries `hide GpxReader`,
and that `lib/vendor/gpx/gpx_reader.dart` is imported by exactly one file. That closes both the
whitespace bypass and the wrong-reader-import case, and keeps the existing non-vacuity floor.

---

_Reviewed: 2026-08-01T10:42:31Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
_Round: 2 — verification of round-1 closures plus regression hunt across the 20 fix commits_
