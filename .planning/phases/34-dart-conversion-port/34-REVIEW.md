---
phase: 34-dart-conversion-port
reviewed: 2026-08-01T08:34:12Z
depth: standard
files_reviewed: 23
files_reviewed_list:
  - app/lib/util/gpx_conversion_util.dart
  - app/lib/util/trail_import_util.dart
  - app/lib/util/route_planner_handoff_util.dart
  - app/lib/util/track_save_options_util.dart
  - app/lib/util/gpx_util.dart
  - app/lib/util/format_util.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/routes/route_planner_screen.dart
  - app/lib/models/trail.dart
  - app/lib/models/trail_summary.dart
  - app/lib/components/trail/elevation_profile.dart
  - app/lib/components/trail/trail_panel.dart
  - app/test/util/gpx_corpus_test.dart
  - app/test/util/gpx_conversion_util_test.dart
  - app/test/util/trail_import_util_test.dart
  - app/test/util/route_planner_handoff_util_test.dart
  - app/test/util/track_save_options_util_test.dart
  - web/src/routes/api/v1/trail/convert/+server.ts
  - web/src/routes/api/v1/trail/convert/convert.test.ts
  - web/src/lib/models/gpx/gpx-corpus.test.ts
  - web/src/lib/util/format_util.ts
  - db/migrations/1785300000_add_moving_duration_to_trails.go
  - app/lib/util/form_data_util.dart
findings:
  critical: 4
  warning: 12
  info: 5
  total: 21
status: issues_found
---

# Phase 34: Code Review Report

**Reviewed:** 2026-08-01T08:34:12Z
**Depth:** standard
**Files Reviewed:** 23
**Status:** issues_found

## Summary

The core numeric port (`GpxMetricsComputation` → Dart) is genuinely faithful. I diffed it line-for-line against `web/src/lib/models/gpx/gpx-metrics-computation.ts` and `gpx.ts:97-167` and found no divergence in the defer-then-publish filter, the `finalElevationGain`/`finalElevationLoss` vs `total*Smoothed` split, the `i = 0` loop bound, or the `summedPointCount` centroid divisor. The haversine constants match exactly: I recomputed fixture 01 independently in Python at `R = 6371000` and got `134.59240148587796`, bit-identical to `expected.json`, confirming `geobase`'s `SphericalGreatCircle` and the TS `haversineDistance` share a radius rather than the corpus having been seeded from one side. Both suites pass locally (Dart 74/74, TS 38/38), read the same 10 on-disk fixtures, use the same declared tolerances (1e-6 m / 1e-9 deg), skip no fixture, and fail loudly on a wrong CWD or a shrunk corpus. The corpus is real, not theatre — with one specific exception (WR-05).

The defects are almost entirely **outside** the ported arithmetic, in the surrounding plumbing that the port displaced:

- The import path loses user data when either post-capture toggle is enabled (CR-01).
- The sanitizer solves four crash inputs but `package:gpx` has eight more unguarded `parse` sites, and the sanitizer was never wired into two of the three remaining GPX parse sites in the app despite the doc comment promising exactly that (CR-02, CR-03).
- The convert endpoint's hard break removed its only content validation, turning it into an unauthenticated, CSRF-exempt verbatim echo with an active XML content type (CR-04).

## Critical Issues

### CR-01: Enabling either post-capture toggle on a file import silently discards the GPX's waypoints, name and description

**File:** `app/lib/util/trail_import_util.dart:97-145`

**Issue:** When `recalcHeights || followRoads` is true, the import rebuilds the track through `mergeHeightsIntoGpx`, which constructs a **bare** `Gpx()` carrying only `trks`:

```dart
Gpx mergeHeightsIntoGpx(...) {
  final gpx = Gpx();          // no metadata, no wpts, no trk name
  if (shape.isEmpty) return gpx;
  gpx.trks = [Trk(trksegs: [...])];
  return gpx;
}
```

`finalGpx` is then fed to `buildLocalTrail` → `trailFromGpx`, which reads `gpx.metadata?.name`, `gpx.trks.first.name`, `gpx.rtes.first.name` and `gpx.wpts` — all now empty. Consequences on every import where the user ticks "Recalculate heights" or "Follow roads":

1. Every `<wpt>` in the imported file is dropped (`trail.expand.waypointsViaTrail` is empty).
2. The trail is named after the *file* (`fallbackName: name`, e.g. `"track.gpx"`) instead of the GPX's own `<metadata><name>`.
3. `description` falls back to `''`.
4. Line 137 re-serialises `finalGpx` into `finalGpxData`, so the stripped document is what gets **uploaded and persisted** as the track file — the loss is permanent, not just in the draft.

This is a regression against the pre-phase behaviour, where the whole file went to the server and `gpx2trail` ran over the original document. `_saveRecordedTrack` and `finishPlanning` are unaffected (they have no source waypoints to lose), so only the import path is broken — and no test covers the toggled-on import (`trail_import_util_test.dart:361-388` only exercises the offline `(false, false)` branch).

**Fix:** Carry the source document's non-geometry content onto the transformed GPX before building the trail:

```dart
finalGpx = mergeHeightsIntoGpx(
  workingShape,
  heights,
  startTime: originalWaypoints.firstOrNull?.time,
  endTime: originalWaypoints.lastOrNull?.time,
)
  ..metadata = gpx.metadata
  ..wpts = gpx.wpts
  ..rtes = gpx.rtes;
if (gpx.trks.isNotEmpty) {
  finalGpx.trks.first.name = gpx.trks.first.name;
  finalGpx.trks.first.desc = gpx.trks.first.desc;
}
```

Add a test asserting that an import with `recalcHeights: true` over a fixture containing `<wpt>` and `<metadata><name>` preserves both.

---

### CR-02: The sanitizer neutralises 4 of `package:gpx`'s 12 unguarded parse sites — common real-world GPX still throws

**File:** `app/lib/util/gpx_conversion_util.dart:33-72`

**Issue:** `sanitizeGpxEleAndTime` handles `<ele>` and `<time>` only. But `gpx_reader.dart` (v2.3.0) calls `double.parse`/`int.parse` on raw accumulated element text with no guard for **eight** other elements:

```dart
double? _readDouble(Iterator<XmlEvent> iterator, String tagName) {
  final doubleString = _readString(iterator, tagName);
  return doubleString != null ? double.parse(doubleString) : null;   // '' -> FormatException
}
int? _readInt(...) => intString != null ? int.parse(intString) : null;
```

Reached for `<hdop>`, `<vdop>`, `<pdop>`, `<magvar>`, `<geoidheight>`, `<ageofdgpsdata>` (`_readDouble`, `gpx_reader.dart:292-308`) and `<sat>`, `<dgpsid>` (`_readInt`). `_readString` returns `''` — not `null` — for any non-self-closing element, so `<hdop></hdop>`, `<hdop>   </hdop>`, `<sat></sat>` and `<pdop>N/A</pdop>` all throw `FormatException` and abort the whole import with the generic `trail_source_import_error` toast.

`<hdop>`/`<vdop>`/`<pdop>`/`<sat>` are among the most commonly emitted optional GPX elements (Garmin, Locus, OsmAnd, many track loggers) and empty-element forms are routine in exporter output. Before this phase these files were parsed server-side by `xml2js`, which never coerces and so never threw. The doc comment's claim that the four handled cases are "the four confirmed `GpxReader` crash inputs" is true only of the cases that were tested, not of the parser's actual surface.

**Fix:** Generalise the sanitize pass over the full set of numerically-parsed tags rather than enumerating two:

```dart
const _numericGpxTags = [
  'ele', 'hdop', 'vdop', 'pdop', 'magvar', 'geoidheight', 'ageofdgpsdata',
];
const _intGpxTags = ['sat', 'dgpsid'];

String sanitizeGpxNumericAndTime(String xml) {
  var out = xml;
  for (final tag in _numericGpxTags) {
    out = out.replaceAllMapped(RegExp('<$tag>([^<]*)</$tag>'), (m) {
      final v = double.tryParse((m[1] ?? '').trim());
      return (v != null && v.isFinite) ? m[0]! : '<$tag/>';
    });
  }
  for (final tag in _intGpxTags) {
    out = out.replaceAllMapped(RegExp('<$tag>([^<]*)</$tag>'), (m) =>
        int.tryParse((m[1] ?? '').trim()) != null ? m[0]! : '<$tag/>');
  }
  return out.replaceAllMapped(RegExp(r'<time>([^<]*)</time>'), (m) =>
      DateTime.tryParse((m[1] ?? '').trim()) != null ? m[0]! : '<time/>');
}
```

Add one corpus fixture or unit test per newly-covered tag.

---

### CR-03: `parseGpxSafely` is not applied at two of the three GPX parse sites, contradicting its own contract

**File:** `app/lib/util/gpx_conversion_util.dart:56-72`, `app/lib/provider/trail/trail_provider.dart:42-43`, `app/lib/entities/trail_entity.dart:187`

**Issue:** The function documents itself as "The single sanctioned parse entry point for any GPX this app did not itself produce" and states that "Later plans redirect every existing `GpxReader().fromString(...)` call site in the app through this function." That redirect never happened. A repo-wide grep finds three production parse sites; two still use the old, `<ele>`/`<time>`-unsafe path:

```dart
// trail_provider.dart:42-43 — GPX downloaded from the server
final sanitizedGpx = sanitizeGpxEmail(gpxResponse.data as String);
final parsedGpx = GpxReader().fromString(sanitizedGpx);

// trail_entity.dart:187 — GPX read back out of the offline ObjectBox cache
gpx: gpxData != null ? GpxReader().fromString(sanitizeGpxEmail(gpxData!)) : null,
```

Both handle GPX the app did not produce (uploaded by any user on the instance, or federated in from another instance). Impact:

- `trail_provider.dart`: the `FormatException` is swallowed by the broad `catch (_)` at line 54 and silently degrades to the offline-cache path — the user sees a stale trail or an error with no indication why.
- `trail_entity.dart`: `toModel()` is invoked at line 74 of that same `catch` block, *outside* any further try, so the exception escapes the notifier and the trail becomes permanently un-openable offline once cached.

Note the asymmetry this creates: the very file the app just imported successfully (via `parseGpxSafely`) can fail to re-open after being saved and re-downloaded, because `importTrailFile` uploads the *unsanitised* original text (`finalGpxData = gpxXml`, line 95) while only sanitising in memory.

**Fix:** Route both call sites through `parseGpxSafely` (which already chains `sanitizeGpxEmail`):

```dart
// trail_provider.dart
final parsedGpx = parseGpxSafely(gpxResponse.data as String);

// trail_entity.dart
gpx: gpxData != null ? parseGpxSafely(gpxData!) : null,
```

Then add a repo-guard test in the style of the existing `PORT-03 gate` test (`trail_import_util_test.dart:391-431`) asserting `GpxReader()` appears in `lib/` exactly once, inside `gpx_conversion_util.dart`.

---

### CR-04: The convert endpoint reflects arbitrary unauthenticated request bodies verbatim with an active XML content type

**File:** `web/src/routes/api/v1/trail/convert/+server.ts:63-79`

**Issue:** The hard break removed the endpoint's only content validation. Previously, the raw-text and JSON branches passed through `gpx2trail`, which threw and produced a `400 "Invalid GPX content"` for anything that was not a parseable GPX document. That guard is gone; the handler now echoes whatever arrived:

```ts
} else {
    gpxData = await event.request.text();   // arbitrary attacker-controlled bytes
}
if (!gpxData || !gpxData.trim().length) { /* 400 */ }
return new Response(gpxData, {
    headers: { "Content-Type": "application/gpx+xml" },   // no nosniff, no attachment
});
```

The endpoint is unauthenticated (`/api/v1/trail/convert` is absent from `privateRoutes` in `web/src/lib/util/authorization_util.ts:4-8` and the handler performs no `locals.pb.authStore` check), and CSRF protection is explicitly disabled for this path — `hooks.server.ts:196` calls `csrf(['/api/v1'])`, and `hooks.server.ts:21-28` skips the origin check for any allow-listed prefix. A cross-origin auto-submitting form with `enctype="text/plain"` is therefore a simple request that reaches the `else` branch and performs a **top-level navigation** whose response body is fully attacker-controlled, served from the victim's origin as `application/gpx+xml` with no `X-Content-Type-Options: nosniff` and no `Content-Disposition: attachment`. `*+xml` types are parsed as XML by browsers, which brings XSLT processing-instruction and XHTML-namespace script vectors into range. Independently of exploitability, an unauthenticated verbatim echo of unvalidated input is a defect: `convert.test.ts:36-47` actively asserts the byte-for-byte reflection as intended behaviour, and `convert.test.ts:124-135` only rejects the empty body.

**Fix:** Restore validation and neutralise the response:

```ts
try {
    GPX.parse(gpxData);   // reuse the existing parser as the validator
} catch {
    throw new ClientResponseError({ status: 400, response: { message: "Invalid GPX content" } });
}

return new Response(gpxData, {
    headers: {
        "Content-Type": "application/gpx+xml",
        "X-Content-Type-Options": "nosniff",
        "Content-Disposition": "attachment; filename=\"track.gpx\"",
    },
});
```

Add a test asserting a non-GPX raw body 400s, and one asserting the `nosniff` header is present.

## Warnings

### WR-01: Off-by-one in `segmentPolylinesFromTrack`'s not-found fallback drops one polyline and mis-pairs every following segment

**File:** `app/lib/util/route_planner_handoff_util.dart:532-537`

**Issue:** When anchor `i` cannot be located, the fallback emits straight lines starting at `k = i`:

```dart
if (idx == -1) {
  for (var k = i; k < anchors.length - 1; k++) {
    polylines.add([anchors[k], anchors[k + 1]]);
  }
  return polylines;
}
```

At that point the loop has added polylines for pairs `(0,1) … (i-2,i-1)` — that is `i-1` entries — and the fallback adds `n-1-i` more, for `n-2` total instead of the required `n-1`. The **missing entry is the `(i-1, i)` pair**, which should come *before* the fallback entries. The result is a list that is both one short and mis-ordered relative to the anchor pairs.

The consumer is `RouteAnchorsNotifier.seedFromTrack`, which indexes positionally (`route_anchor_provider.dart:577`: `i < segmentPolylines.length ? segmentPolylines[i] : [straight line]`). Every segment from index `i-1` onward therefore receives the polyline belonging to the *next* anchor pair, and the last one silently degrades to a straight line — a scrambled route on re-edit rather than a clean fallback. No test covers the `idx == -1` branch.

**Fix:**

```dart
if (idx == -1) {
  for (var k = i > 0 ? i - 1 : 0; k < anchors.length - 1; k++) {
    polylines.add([anchors[k], anchors[k + 1]]);
  }
  return polylines;
}
```

Add a test feeding anchors that include a coordinate absent from the track and asserting `polylines.length == anchors.length - 1`.

---

### WR-02: A failed road-snap falls back to the *downsampled* shape, not the original full-resolution one

**File:** `app/lib/util/route_planner_handoff_util.dart:120-150`; call sites `route_planner_handoff_util.dart:321` and `trail_import_util.dart:111-115`

**Issue:** `snapShapeToRoads` returns its `shape` parameter unchanged on error or rejection — but every caller passes `buildNavShape(points)`, i.e. the ≤500-point downsampled Valhalla *request hint*, not the source geometry:

```dart
workingShape = await snapShapeToRoads(ref, buildNavShape(points), 'pedestrian');
```

So a snap that fails (timeout, offline flip, 5xx) or is rejected by `snapResultAcceptable` silently replaces a full-resolution track with a 500-point decimation, which is then persisted. `navigation_screen.dart:788-793` carries an explicit comment warning that `buildNavShape`'s cap "applies only to this outbound hint" — the fallback path violates exactly that invariant. A 5000-point recorded track loses 90% of its vertices because a network call failed.

**Fix:** Return the caller's real geometry on the fallback path:

```dart
Future<List<Map<String, double>>> snapShapeToRoads(
  WidgetRef ref,
  List<Map<String, double>> requestShape,
  String costing, {
  List<Map<String, double>>? fallbackShape,
}) async {
  final fallback = fallbackShape ?? requestShape;
  try {
    ...
    return snapResultAcceptable(requestShape, snapped) ? snapped : fallback;
  } catch (_) {
    return fallback;
  }
}
```

and pass the full-resolution list as `fallbackShape` at all three call sites.

---

### WR-03: A silently-failed snap still invalidates the leg's elevations, which a failed refetch then destroys

**File:** `app/lib/util/route_planner_handoff_util.dart:323-337`

**Issue:** `legElevations[i] = null` is executed for every leg whose `snapped[k].length >= 2`, with no way to distinguish "the snap succeeded and the indices really are stale" from "`snapShapeToRoads` swallowed the error and handed back the input unchanged". Combined with the height backfill's own silent fallback (`if (heights.isEmpty) continue;`, line 359), a leg that previously carried perfectly good elevations ends up with `ele: null` on every point purely because two independent network calls failed. The trail's `elevationGain`/`elevationLoss` then compute to 0 and are saved that way. Ticking "Follow roads" while on a flaky connection is enough to destroy a route's elevation data.

**Fix:** Have `snapShapeToRoads` signal whether a real snap occurred (e.g. return a record `(shape, snapped: bool)` or `null` on fallback), and only clear `legElevations[i]` when it did.

---

### WR-04: `_onFinish`'s double-tap guard no longer covers the gate, contradicting its own doc comment

**File:** `app/lib/routes/route_planner_screen.dart:513-533`

**Issue:** The comment states "`[_finishing]` still guards the forward-push branch against a double-tap firing two concurrent conversions/navigations", but in the non-edit branch `setState(() => _finishing = true)` is now deferred until *after* `await resolveTrackSaveOptions(...)`:

```dart
if (_finishing) return;
try {
  if (_editMode) {
    setState(() => _finishing = true);      // set before the await
    ...
  } else {
    final options = await resolveTrackSaveOptions(ref, context);   // guard still false here
    if (options == null) return;
    if (!mounted) return;
    setState(() => _finishing = true);
```

Before this phase the flag was set unconditionally on the first line. Now a second tap while the sheet is open (the finish action stays enabled — `_buildFinishAction:487` reads `!_finishing`) re-enters, opens a second sheet, and can drive two concurrent `finishPlanning` runs, each doing its own snap/height fetches and its own `pushReplacement`. `navigation_screen.dart:736-739` has the same shape but at least re-checks `if (_savingTrack) return;` immediately after the await; `_onFinish` has no such re-check.

**Fix:** Re-check the guard after the await, mirroring `_saveRecordedTrack`:

```dart
final options = await resolveTrackSaveOptions(ref, context);
if (options == null) return;
if (!mounted || _finishing) return;
setState(() => _finishing = true);
```

---

### WR-05: The Dart corpus's waypoint-icon assertion is tautological and cannot fail

**File:** `app/test/util/gpx_corpus_test.dart:330-338`

**Issue:** The expected value is computed with the identical expression the production code uses:

```dart
// test
final expectedIconData = expectedIcon != null
    ? (fontAwesomeIconsMap[expectedIcon] ?? FontAwesomeIcons.circle)
    : FontAwesomeIcons.circle;
expect(actualWaypoint.icon, expectedIconData);

// production, gpx_conversion_util.dart:467
icon: fontAwesomeIconsMap[wpt.sym] ?? FontAwesomeIcons.circle,
```

If `fontAwesomeIconsMap` lost the `campground` or `mountain` key, *both* sides would evaluate to `FontAwesomeIcons.circle` and the assertion would still pass. Fixture 10's `notes` explicitly claims this test confirms both sym strings "are present in `app/lib/util/icon_util.dart`'s `fontAwesomeIconsMap` on the Dart side" — it does not. The TS counterpart (`gpx-corpus.test.ts:227-236`) compares against the literal `"campground"` string and is a real assertion, so the two suites are not field-for-field comparable here despite the file header claiming they are.

**Fix:** Assert the map lookup succeeded before comparing:

```dart
final expectedIcon = expectedWaypoint['icon'] as String?;
if (expectedIcon != null) {
  expect(fontAwesomeIconsMap[expectedIcon], isNotNull,
      reason: '${fixture.dir}: sym "$expectedIcon" missing from fontAwesomeIconsMap');
}
expect(actualWaypoint.icon,
    expectedIcon != null ? fontAwesomeIconsMap[expectedIcon] : FontAwesomeIcons.circle);
```

---

### WR-06: TS and Dart diverge on a non-empty but unparseable `<time>`, and no fixture covers it

**File:** `app/lib/util/gpx_conversion_util.dart:45-53` vs `web/src/lib/models/gpx/waypoint.ts:56-58` + `web/src/lib/models/gpx/gpx.ts:116-123`

**Issue:** For `<time></time>` the two agree (xml2js yields `''`, which is falsy, so TS leaves `time` undefined; Dart rewrites to `<time/>` → `null`). For a **non-empty** unparseable body they do not:

- TS: `if (object.time) this.time = new Date(object.time)` — a garbage string is truthy, producing an `Invalid Date`. `getTotals` then evaluates `startTime && endTime` as true and computes `endTime.getTime() - startTime.getTime()` → `NaN`, so `duration` is `NaN`.
- Dart: `DateTime.tryParse` fails → `<time/>` → `null` → the segment contributes 0.

The same divergence covers formats JS `Date` accepts but `DateTime.tryParse` rejects (e.g. `2024/01/01T10:00:00Z`, `Jan 1 2024`): TS yields a real duration, Dart yields 0. No corpus fixture exercises a malformed `<time>`, so the parity suite cannot see it — the phase's central claim ("proven identical to the TS by a shared fixture corpus") does not hold on this input class.

**Fix:** Add a corpus fixture with a malformed `<time>` and align the TS side to the (better) Dart semantics — `parseElevation`'s sibling for dates:

```ts
if (object.time) {
  const d = new Date(object.time);
  if (!Number.isNaN(d.getTime())) this.time = d;
}
```

---

### WR-07: `buildFinalPlannedGpx`'s "no network when both flags are off" claim is false, and the offline test is constructed to avoid the real case

**File:** `app/lib/util/route_planner_handoff_util.dart:290-297, 341-362`; test `app/test/util/route_planner_handoff_util_test.dart:771-808`

**Issue:** The doc comment states "both underlying network steps are skipped entirely when their flag is off — the combination that makes this function safe to call with no network access at all (D-15's offline path)". That is not what the code does: the `pending` list is built unconditionally from legs whose `legElevations[i] == null`, so any leg without resolved elevations triggers a `/valhalla/height` request even with `refetchAllHeights: false` and `snapCosting: null`. The behaviour is *safe* (`fetchHeightsForShape` swallows the failure and returns `const []`), but the stated invariant is wrong.

The test that claims to prove the invariant sidesteps it: it first seeds against a *working* API so every leg resolves elevations, then transplants that state into the failing session (`ref.read(routeAnchorsProvider.notifier).state = seededState;`). `expect(failingApi.heightCalls, 0)` therefore only holds because `pending` is empty. A session with any un-elevated leg would issue N failed requests, and nothing in the suite would notice.

**Fix:** Either gate the backfill on connectivity (`if (ref.read(onlineStatusProvider))`) and make the doc true, or correct the doc and add a test that seeds legs with `elevations == null` against a failing API and asserts the call is attempted-and-tolerated.

---

### WR-08: A zero-length recording persists `moving_duration = 0`, the exact state D-10 forbids

**File:** `app/lib/routes/navigation_screen.dart:861-867`, `app/lib/util/gpx_conversion_util.dart:510`, `app/lib/util/form_data_util.dart:29-35`

**Issue:** `NavigationStats.elapsed` defaults to `Duration.zero` (`navigation_stats_provider.dart:29`) and stays zero until the 1-second tick starts. `_saveRecordedTrack` passes it unconditionally, `trailFromGpx` maps it to `0.0` (not `null`), and `form_data_util`'s guard is `if (movingDuration != null)` — so `moving_duration=0` is written. That is precisely what the comment two lines above says must not happen: *"sending an empty string for an absent value would write 0 into PocketBase and defeat D-10's 'no value' state."* The display rule then masks it (`> 0` falls back to `duration`), so the bad row is invisible until something else reads the field.

**Fix:** Treat zero as absent at the source:

```dart
movingDuration: navStats.elapsed > Duration.zero ? navStats.elapsed : null,
```

or make `trailFromGpx` map a zero `movingDuration` to `null`.

---

### WR-09: `moving_duration` is documented in the OpenAPI response schemas but silently stripped by the JSON API's Zod schemas

**File:** `web/src/lib/models/api/trail_schema.ts:5-54` vs `web/src/lib/models/api/openapi_schemas.ts:399, 476, 534`

**Issue:** Three OpenAPI response schemas now advertise `moving_duration`, and `web/src/lib/models/trail.ts` carries the field — but neither `TrailCreateSchema` nor `TrailUpdateSchema` declares it. Zod object schemas strip unknown keys by default, so a client following the published API contract and POSTing `moving_duration` to `/api/v1/trail` has it silently discarded with no error. Only the multipart `/api/v1/trail/form` route works (it bypasses Zod entirely via `uploadCreate`/`uploadUpdate`), which is why the Flutter app happens to function. The documented interface and the enforced interface disagree.

**Fix:** Add `moving_duration: z.number({ coerce: true }).nonnegative().optional()` to both schemas, or remove it from the request-side OpenAPI docs and document it as response-only.

---

### WR-10: Editing a recorded trail's route on the web leaves a stale `moving_duration` that then wins the display rule

**File:** `web/src/lib/util/format_util.ts:26-34`, `web/src/lib/models/trail.ts:114`

**Issue:** The phase's stated integrity property is that no recompute path can destroy moving time. The converse case is unhandled: the web trail-edit page recomputes `duration` from a newly-drawn route, but `moving_duration` is copied through untouched (`buildFormData` skips only `undefined`). `trailDisplayDuration` then prefers the stale value — so a trail whose route was replaced displays the moving time of the *old* route in preference to the correctly recomputed elapsed time, everywhere (`trail_card`, `trail_list_item`, `trail_table`, `trail_info_panel`, map popups). The same applies in the app via `mergeRouteIntoTrail` (`route_planner_handoff_util.dart:558-584`), which sets `duration` from the planner estimate and leaves `movingDuration` alone.

**Fix:** Clear `moving_duration` whenever the track geometry is replaced — in the web edit page's route-replacement path and in `mergeRouteIntoTrail` (`movingDuration: null` when `finalGpx` differs from the existing track). Requires making the field explicitly nullable in the form encoder so `null` is transmitted rather than skipped.

---

### WR-11: A zero-point leg emits an empty `<trkseg>`, which `anchorsFromTrack` then drops — losing that anchor

**File:** `app/lib/util/route_planner_handoff_util.dart:375-382` and `468-488`

**Issue:** The trkseg builder's inner bound explicitly handles the degenerate **1**-point leg ("A degenerate 1-point leg keeps its single point rather than emitting an empty trkseg, which `anchorsFromTrack` would skip and so lose that anchor") but a **0**-point leg falls through to `legPoints[i].length` = 0 and emits `Trkseg(trkpts: [])`. `anchorsFromTrack:473-478` filters empty segments out, so that anchor disappears on re-edit and the route silently loses a waypoint. `segmentPolylinesFromTrack` can produce a 1-point polyline (`allPoints.sublist(prevIndex, idx + 1)` with `prevIndex == idx`), and WR-01's short list feeds straight lines into the tail, so an empty `legPoints` is reachable through a combination of the defensive paths rather than being purely hypothetical.

**Fix:** Skip empty legs when emitting, or better, assert/repair upstream:

```dart
for (var i = 0; i < legs.length; i++)
  if (legPoints[i].isNotEmpty)
    Trkseg(trkpts: [...]),
```

and document that an empty leg is unrepresentable in the round-trip format.

---

### WR-12: `importTrailFile`'s blanket `catch` also wraps the navigation push and discards the exception entirely

**File:** `app/lib/util/trail_import_util.dart:76-153`

**Issue:** The single `try` spans read → transcode → parse → sheet → transforms → trail build → `pendingImportedTrail = trail` → `navContext.push(...)`, and the handler is `catch (e) { showError(); }` with `e` unused and no logging of any kind. Two consequences:

1. A throw from `navContext.push` (or from anything after `pendingImportedTrail` is assigned) shows the user "import failed" for an import that in fact succeeded, and leaves a stale non-null `pendingImportedTrail` global behind.
2. Every distinct failure mode in a ~75-line block — unreadable file, `transcodeToGpx`'s `StateError`, an unsanitised-tag `FormatException` (CR-02), a `StateError` from a `<trkpt>` missing `lat`/`lon` — collapses into one untyped toast with the exception dropped on the floor. Field diagnosis of the import path is impossible.

**Fix:** Narrow the `try` to end after `buildLocalTrail`, and log the exception:

```dart
} catch (e, st) {
  debugPrint('importTrailFile failed: $e\n$st');
  showError();
}
```

## Info

### IN-01: `totalElevationGain` / `totalElevationLoss` are ported dead fields

**File:** `app/lib/util/gpx_conversion_util.dart:115-116, 208-222`
**Issue:** The raw (unsmoothed) accumulators have zero consumers — `computeTrailMetrics` reads only `totalDistanceSmoothed`, `finalElevationGain` and `finalElevationLoss`, and no test reads them either (only `totalDistance` is asserted, at `gpx_conversion_util_test.dart:346`). This directly contradicts the file's own rationale for omitting `cumulativeDistance`: "its only consumer is the web trail-edit crop slider, which has no Dart equivalent, so porting it would be dead code."
**Fix:** Either drop the two fields and the `if (elevation != null)` block that maintains them, or add a doc line explaining why they are retained where `cumulativeDistance` was not.

---

### IN-02: `trailDisplayDuration` (Dart) declares a nullable return it can never produce

**File:** `app/lib/util/format_util.dart:15-21`
**Issue:** `TrailSummary.duration` is a non-nullable `double`, so the fallback branch always returns a value — the `double?` return type is unreachable-null. It forces dead null handling at all three call sites (`trail_card.dart:383` and `trail_list_item.dart:364` write `?? 0`; `trail_panel.dart:213` writes `displayDuration != null && displayDuration > 0`).
**Fix:** Change the return type to `double` and drop the `?? 0` / `!= null` guards.

---

### IN-03: `sanitizeGpxEmail` interpolates matched text into attribute values without XML-escaping

**File:** `app/lib/util/gpx_util.dart:8-13`
**Issue:** Pre-existing, but this phase promotes it to the sanctioned entry point for all third-party GPX via `parseGpxSafely`. `'<email id="${m[1]}" domain="${m[2]}"/>'` performs no escaping, so an `<email>` body containing `"` or `&` yields either malformed XML (parse throws, import fails) or an injected extra attribute.
**Fix:** Escape `&`, `<`, `"` in both captures before interpolation, and skip the rewrite entirely if either capture still contains a quote.

---

### IN-04: Fixture 10 — the only fixture with waypoints and a non-zero duration — has no DERIVATION.md

**File:** `fixtures/gpx-corpus/10-realistic-track/`
**Issue:** The integrity check only requires `DERIVATION.md` when `derivation == "hand"` (`gpx_corpus_test.dart:358-367`, `gpx-corpus.test.ts:249-252`), and fixture 10 is marked `"seeded"`. It is nonetheless the only fixture pinning `durationMs`, `date`, `description` and waypoints, and its expected values were produced by running the TS implementation over its own input — there is no independent record of the seeding procedure beyond a prose `notes` field.
**Fix:** Add a `DERIVATION.md` recording the exact seeding command, or extend the integrity check to require one for every fixture.

---

### IN-05: Unused catch-clause variables

**File:** `app/lib/util/route_planner_handoff_util.dart:146` (`catch (e)`), `app/lib/util/trail_import_util.dart:150` (`catch (e)`)
**Issue:** Both bind an exception variable and never use it. Elsewhere in the same files the intentional-discard form `catch (_)` is used consistently (`route_planner_handoff_util.dart:180`, `trail_import_util.dart:233`).
**Fix:** Use `catch (_)`, or log the exception (see WR-12).

---

_Reviewed: 2026-08-01T08:34:12Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
