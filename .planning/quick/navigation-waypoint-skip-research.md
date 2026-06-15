# Research: Waypoint Skip / Shortcut Handling in Wanderer Navigation

**Date:** 2026-06-15
**Domain:** Flutter mobile GPS navigation — maneuver advancement logic
**Confidence:** HIGH (codebase verified), MEDIUM (industry behavior from docs/community research)

---

## 1. Current Implementation Summary

### Entry point

`app/lib/routes/navigation_screen.dart` owns the GPS stream. Every position update is broadcast to two notifiers:

```
Geolocator.getPositionStream()
  → navigationProvider(response).notifier.onPosition(LatLng)
  → navigationStatsProvider(response).notifier.onPosition(Position)
```

### Advancement algorithm (`app/lib/provider/navigation_provider.dart`)

The entire waypoint-advance logic lives in `Navigation.onPosition()` (lines 83–111):

```dart
static const _kManeuverAdvanceThresholdMeters = 30.0;

void onPosition(LatLng pos) {
  // 1. Append to breadcrumb.
  state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);

  // 2. Compute next index.
  final next = state.currentManeuverIndex + 1;
  if (next >= state.response.maneuvers.length) return; // at end

  // 3. Look up the next maneuver's begin-shape point.
  final rawIndex = state.response.maneuvers[next].beginShapeIndex;
  final clampedIndex = rawIndex.clamp(0, validShape.length - 1).toInt();
  final targetLatLng = validShape[clampedIndex];

  // 4. Haversine distance to that single point.
  final meters = _distance.as(LengthUnit.Meter, pos, targetLatLng);

  // 5. Advance only if within threshold.
  if (meters <= _kManeuverAdvanceThresholdMeters) {
    state = state.copyWith(currentManeuverIndex: next);
  }
}
```

### The core problem

The algorithm only checks whether the user is within 30 m of the **next** maneuver's begin-shape point. It advances by exactly one maneuver per GPS fix. This creates two failure modes:

**Failure mode A — Shortcut skip:** User jumps from maneuver 2 directly to the area of maneuver 4. The app continues checking for proximity to maneuver 3. Since the user never passes within 30 m of maneuver 3's shape point, the app is stuck displaying maneuver 3 instructions indefinitely — even as the user approaches and passes maneuver 4 and 5.

**Failure mode B — Switchback mismatch:** GPS accuracy on trails is typically ±5–15 m. On tight switchbacks, the shape point for maneuver N+1 may be only 20–40 m from the shape point of maneuver N, and from the previous leg's path. A user walking the correct trail may accidentally trigger the wrong maneuver by being within 30 m of the wrong shape point.

### Data model available

`NavigateResponse` contains:
- `maneuvers: List<NavigateManeuver>` — each with `beginShapeIndex` (index into shape), `instruction`, `length`, `type`
- `shape: List<List<double>>` — the full route polyline as `[lat, lon]` pairs
- Extension `shapeAsLatLng` — converts shape to `List<LatLng>`

`NavigateManeuver` has `beginShapeIndex` but NOT `endShapeIndex` (end is inferred as the next maneuver's `beginShapeIndex`).

The existing `latlong2` package (already in `pubspec.yaml`) provides the `Distance` class used for Haversine calculation. No additional geospatial dependency exists.

---

## 2. Industry Approaches

### 2.1 Komoot

**Behavior:** Komoot tracks progress along the route polyline, not just waypoint proximity. When the user deviates, the route line changes color to indicate off-route status. Automatic rerouting recalculates a new path to the destination (requires internet). When offline, it continues showing the original route and alerts the user.

**Shortcut handling:** Komoot has NO built-in waypoint-skip feature. If a user skips completed sections, they must manually stop navigation, edit the route in the planner to remove those sections, and restart. Komoot's "waypoints" are route-shaping points and act as forced intermediate destinations — the router tries to route back to any missed waypoint rather than skipping it forward.

**Key insight:** Komoot uses the route-snapping model (user position projected onto route polyline) to determine progress, which means it knows "how far along the route" the user is, not just distance to the next point.

**Source:** Komoot Navigation FAQ, Komoot support docs [MEDIUM confidence — official docs]

### 2.2 OsmAnd

**Behavior:** OsmAnd detects waypoints as passed based on proximity (a configurable radius). The critical problem: intermediate destinations are not always auto-dismissed, meaning OsmAnd will continuously try to route back to a missed waypoint even when the user is far past it. This is a known UX complaint in the OsmAnd community (GitHub discussion #18662: "Feature: auto-dismiss markers / intermediate destinations").

**Shortcut handling:** If "recalculate route" is enabled, OsmAnd routes back to the missed waypoint. If disabled, the unit goes silent until the user returns to the original route. There is no automated "skip forward" logic. Users can manually skip intermediate destinations.

**Key insight:** OsmAnd's proximity-only model produces the exact same problem Wanderer currently has — the app gets stuck at a missed waypoint. The community has been requesting a distance-along-route approach to auto-dismiss waypoints.

**Source:** OsmAnd GitHub community, OsmAnd docs [MEDIUM confidence — community + official docs]

### 2.3 Garmin Devices (automotive/cycling)

**Behavior:** Garmin distinguishes between "shaping points" (intermediate route-shaping waypoints) and "via points" (must-stop destinations). Shaping points are auto-skipped when the user passes them — no action required. Via points require the user to reach them or manually skip.

**Shortcut handling for shaping points:** Auto-skip occurs when the user is clearly past the shaping point based on route progress. For via points, with "recalculate route" on, Garmin tries to route back to the missed point. With it off, Garmin goes silent.

**Key insight:** The shaping-point vs. via-point distinction is the cleanest model for hiking. Trail navigation waypoints are analogous to shaping points — they mark turns or landmarks but do not require physical visits. Auto-skip for shaping points is standard Garmin behavior.

**Source:** Garmin forums, Garmin support docs [MEDIUM confidence — community + official support]

### 2.4 Mapbox Navigation SDK (reference implementation)

Mapbox Navigation SDK is the closest analog to what Wanderer needs to build from scratch. Its model:

1. **Route snapping:** Every GPS fix is snapped to the nearest point on the route polyline. The snapped location (not raw GPS) drives all progress tracking.
2. **RouteProgress hierarchy:** RouteProgress → RouteLegProgress → RouteStepProgress. Each level tracks distance remaining and fraction completed.
3. **Step advancement:** When the snapped position passes the `beginShapeIndex` of the next step in the route-progress sense (i.e., the fraction-completed for the current step reaches 1.0 or the user's along-track distance exceeds the step's end point), the SDK advances to the next step.
4. **Off-route detection:** If the snapped distance from the user's raw GPS to the route polyline exceeds a threshold, an off-route event fires and rerouting is triggered.

**Key insight:** Mapbox's approach does not check "is the user within X meters of a single point." Instead it asks "where along the route (in meters from start) is the user, and have they passed the begin-shape-index of the next step?" This naturally handles shortcuts — if you jump from 0.3 to 0.8 of route progress, all intermediate steps whose progress is < 0.8 are auto-skipped.

**Source:** Mapbox Navigation SDK docs, Mapbox Android SDK architecture [MEDIUM confidence — official docs, some SDK is closed-source]

---

## 3. Recommended Approaches for Wanderer

### Option A — Multi-step lookahead (minimal change, maximum compatibility)

**What it does:**

Instead of checking only the next maneuver (index+1), scan forward through all remaining maneuvers and find the first one the user is NOT yet within threshold of. Any maneuvers between the current index and that first "not-yet-reached" one are skipped.

More precisely: on each GPS fix, scan forward from `currentManeuverIndex + 1` up to some lookahead limit (e.g., 5 maneuvers ahead). For each candidate maneuver, check if the user is within the threshold. Advance `currentManeuverIndex` to the furthest candidate that satisfies the threshold check. Because advancement is still forward-only and threshold-based, the existing test suite remains valid with minor extensions.

```dart
// Pseudocode addition to navigation_provider.dart
void onPosition(LatLng pos) {
  state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);

  final maneuvers = state.response.maneuvers;
  final shape = state.response.shapeAsLatLng;
  if (shape.isEmpty) return;

  var idx = state.currentManeuverIndex;
  const maxLookahead = 5; // tune as needed
  final limit = (idx + maxLookahead).clamp(0, maneuvers.length - 1);

  for (int candidate = idx + 1; candidate <= limit; candidate++) {
    final rawIndex = maneuvers[candidate].beginShapeIndex;
    final clampedIndex = rawIndex.clamp(0, shape.length - 1).toInt();
    final dist = _distance.as(LengthUnit.Meter, pos, shape[clampedIndex]);
    if (dist <= _kManeuverAdvanceThresholdMeters) {
      idx = candidate; // keep scanning for further matches
    }
  }

  if (idx > state.currentManeuverIndex) {
    state = state.copyWith(currentManeuverIndex: idx);
  }
}
```

**Implementation complexity:** LOW — 10–15 lines added to `navigation_provider.dart`. No new dependencies. No model changes. Existing tests remain valid; extend with multi-skip test cases.

**Hiking-specific trade-offs:**
- Pro: Handles the most common shortcut case (user bypasses 1–4 maneuvers on a well-known trail).
- Pro: Zero new dependencies; fits cleanly in the existing notifier pattern.
- Pro: Battery-friendly — same O(1) computation per GPS fix (just a bounded loop).
- Con: If user skips more than `maxLookahead` maneuvers in one GPS fix (unlikely but possible on fast transport), still gets stuck.
- Con: On very tight switchbacks where multiple shape points are close together, multiple maneuvers may fire at once — usually desirable behavior.
- Con: Does not detect that user has "passed" a waypoint along the route axis — only proximity.

**Best scenario:** User takes a shortcut that bypasses 1–5 maneuvers, jumping from one part of the trail to another. The app catches up on the next GPS fix.

---

### Option B — Route-progress projection (robust, industry-standard)

**What it does:**

Project the user's GPS position orthogonally onto the route polyline (find the nearest point on any segment). Track progress as a cumulative along-track distance from the route start. Advance `currentManeuverIndex` to the highest maneuver whose `beginShapeIndex` shape point is at or before the user's along-track position. This is the Mapbox model.

The key computation: for each polyline segment `(shape[i], shape[i+1])`, find the closest point on that segment to `pos`. The sum of segment lengths up to that point is the along-track distance. Maneuver N is "past" when along-track distance ≥ cumulative length to `maneuvers[N].beginShapeIndex`.

```dart
// Core helper — no new dependency needed, pure math using latlong2 Distance
double _alongTrackDistanceTo(LatLng pos, List<LatLng> shape) {
  // Find nearest segment and project.
  // Returns cumulative distance along shape to the projected point.
  // Uses latlong2 Distance for segment lengths.
}

void onPosition(LatLng pos) {
  state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);
  final shape = state.response.shapeAsLatLng;
  if (shape.isEmpty) return;

  final atd = _alongTrackDistanceTo(pos, shape);

  // Precompute cumulative distances for each maneuver's begin_shape_index
  // (or cache this on first call).
  final maneuvers = state.response.maneuvers;
  int newIndex = state.currentManeuverIndex;
  for (int i = state.currentManeuverIndex + 1; i < maneuvers.length; i++) {
    final maneuverAtd = _cumulativeDistance(shape, maneuvers[i].beginShapeIndex);
    if (atd >= maneuverAtd - _kManeuverAdvanceThresholdMeters) {
      newIndex = i;
    } else {
      break; // maneuvers are ordered; stop at first not-yet-reached
    }
  }

  if (newIndex > state.currentManeuverIndex) {
    state = state.copyWith(currentManeuverIndex: newIndex);
  }
}
```

The cumulative distance table for maneuver shape indices can be computed once during `build()` and cached as a field on the notifier — it never changes for a given `NavigateResponse`.

**Implementation complexity:** MEDIUM — 40–60 lines in `navigation_provider.dart`. Requires implementing a nearest-segment projection function in pure Dart (no new pub dependency; `latlong2` `Distance` class provides Haversine per-segment). The `_alongTrackDistanceTo` function is ~30 lines of well-understood geospatial math. A precomputed cumulative distance array avoids runtime overhead.

**Hiking-specific trade-offs:**
- Pro: Handles any number of skipped maneuvers in one jump — completely immune to the shortcut problem.
- Pro: Works correctly on switchbacks because it uses the polyline axis, not Euclidean distance to a waypoint dot.
- Pro: Enables accurate "distance remaining on route" display (future feature).
- Con: Requires correct projection implementation. The nearest-segment search is O(n) over the shape — for trails with 500 shape points this is ~500 Haversine calls per GPS fix (typically every 1–3 seconds). At ~5 µs per call on modern phones, this is <3 ms total — acceptable.
- Con: If GPS accuracy is poor (±15 m) and the trail doubles back close to itself (within 30 m), the projection could snap to the wrong segment. A forward-only constraint on segment search (only scan segments at or ahead of the current position) mitigates this.
- Con: The `_alongTrackDistanceTo` function requires unit tests; more test surface than Option A.

**Best scenario:** User takes any shortcut regardless of length, or GPS jumps significantly. App catches up immediately on the next fix regardless of how many maneuvers were skipped.

---

### Option C — "Dead-zone" / already-passed detection (additive guard)

**What it does:**

This is not a standalone approach — it is a complementary heuristic layered on top of Option A or Option B. A maneuver is marked as "already passed and can be auto-skipped" if the user's position is more than a dead-zone distance (e.g., 80 m) past the maneuver's begin-shape point in the direction of travel. "Past" is defined as: the along-track distance to the user's position exceeds the maneuver's cumulative distance.

Practically: when `currentManeuverIndex` has not advanced for N seconds (e.g., 30 seconds) AND the user is more than `dead_zone_meters` (e.g., 100 m) beyond the expected shape point (measured using the along-track computation from Option B), auto-advance.

**Implementation complexity:** MEDIUM — requires the along-track distance concept from Option B (cannot be done purely with proximity). Adds a time-based staleness check (a `DateTime? _lastAdvanced` field on the notifier).

**Hiking-specific trade-offs:**
- Pro: Prevents being stuck indefinitely even if Options A or B fail to advance (safety net).
- Con: Time-based staleness introduces a delay (30 s) before auto-skip fires — user sees wrong instruction for up to 30 s.
- Con: Adds behavioral complexity and state; harder to unit-test precisely.

**Best scenario:** User goes very slowly (GPS fix cadence drops) or GPS accuracy is so poor that projection doesn't stabilize. Acts as a backstop.

---

## 4. Suggested Default

**Recommended: Option B (route-progress projection) as the primary algorithm, with Option A's multi-lookahead threshold as a secondary trigger for edge cases.**

### Rationale

Option A is easy but fragile: a user who skips 6+ maneuvers still gets stuck. On a long hiking trail with many short switchback maneuvers, it is entirely plausible to shortcut 10+ maneuvers at once. The `maxLookahead = 5` cap is arbitrary and any cap creates a failure point.

Option B matches how every professional navigation SDK (Mapbox, HERE, TomTom) handles this problem. The along-track projection model is mathematically correct and handles any shortcut of any length in a single GPS fix. The implementation in pure Dart using the already-present `latlong2` package requires no new dependency.

### Concrete implementation plan

1. **Add precomputed cumulative distances** — In `Navigation.build()`, compute a `List<double> _maneuverCumulativeMeters` where index `i` = total path distance from `shape[0]` to `shape[maneuvers[i].beginShapeIndex]`. This is computed once per navigation session (not per GPS fix).

2. **Implement `_projectAlongTrack(LatLng pos, List<LatLng> shape, int fromShapeIndex)`** — A helper that linearly searches shape segments starting from the current maneuver's shape index (forward-only constraint). For each segment, computes the perpendicular foot (scalar projection clamped to [0,1]) and accumulates segment lengths to get a total along-track distance. Uses `latlong2 Distance` for per-segment Haversine.

3. **Replace the single-point proximity check** — The new `onPosition` computes `atd = _projectAlongTrack(pos, shape, maneuvers[currentManeuverIndex].beginShapeIndex)`. It then scans forward through maneuvers: any maneuver `i` where `_maneuverCumulativeMeters[i] <= atd + _kManeuverAdvanceThresholdMeters` is considered reached. Advance `currentManeuverIndex` to the highest such `i`.

4. **Retain forward-only invariant** — The existing test `advancement is forward-only: near earlier maneuver never decrements` continues to pass because the algorithm only scans forward.

5. **Retain breadcrumb behavior** — No change.

6. **Extend test suite** in `navigation_provider_test.dart`:
   - Test: position near shape-point of maneuver 3 (skipping maneuver 2) → advances directly to 3.
   - Test: position at route midpoint between two maneuvers (no maneuver shape point close by) → does not advance.
   - Test: cumulative distance table is computed correctly for a known shape.

### Performance note

A 500-point shape (the max from `buildNavShape` in `gpx_util.dart`) requires at most 500 Haversine calls per GPS fix. `latlong2`'s `Distance` class is a thin wrapper around standard Haversine math. At ~5 µs per call, worst case is ~2.5 ms per GPS fix — well within the 1-second cadence. The forward-only search constraint (starting from current maneuver's shape index) further reduces average search depth.

### GPS accuracy note

On hiking trails, GPS accuracy is typically ±5–15 m, with occasional jumps of 30–50 m in dense canopy. The existing 30 m threshold (`_kManeuverAdvanceThresholdMeters`) already accounts for this. With the projection model, the threshold acts as a forward lookahead buffer (the user is considered to have "reached" a maneuver when their along-track position is within 30 m of the maneuver's shape point along the route, not 30 m in Euclidean distance). This is strictly better behaved on switchbacks.

---

## References

- [Mapbox Route Progress docs](https://docs.mapbox.com/android/navigation/guides/turn-by-turn-navigation/route-progress/) — route progress hierarchy and step advancement model
- [OsmAnd GPX Navigation docs](https://osmand.net/docs/user/navigation/setup/gpx-navigation/) — track attachment and deviation handling
- [OsmAnd auto-dismiss feature request](https://github.com/osmandapp/OsmAnd/discussions/18662) — community evidence that proximity-only models fail
- [Komoot Navigation FAQ](https://support.komoot.com/hc/en-us/articles/10605424981402-Navigation-FAQ) — off-route color change, rerouting behavior
- [Komoot Replan while navigating](https://support.komoot.com/hc/en-us/articles/5315305499290-Replan-a-Tour-while-navigating) — no built-in waypoint skip
- [Garmin forums — skip waypoint while navigating](https://forums.garmin.com/sports-fitness/cycling/f/edge-530/260816/skip-a-waypoint-while-navigating-a-course) — shaping points vs via-points distinction
- [maps_toolkit Dart package](https://pub.dev/packages/maps_toolkit) — `PolygonUtil.distanceToLine` as reference for spherical point-to-line distance
- [latlong2 Dart package](https://pub.dev/packages/latlong2) — already in `pubspec.yaml`, `Distance` class for Haversine
