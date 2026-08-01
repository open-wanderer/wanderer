---
status: diagnosed
trigger: "a) pass. However: the same trail uploaded on the web vs. the app produces different lengths: ~/Downloads/19440058502_ACTIVITY.fit — On web: 10.51km | On app: 10.97km. Elevation matches: 344m up, 351 down"
created: 2026-08-01T14:10:00Z
updated: 2026-08-01T15:20:00Z
---

## Current Focus

hypothesis: CONFIRMED — the app's ElevationProfile stats header renders the RAW
  cumulative-distance accumulator (the chart's plotting axis) as the trail's
  distance, while the web renders the SMOOTHED accumulator. Same GPX, same
  point set, two different accumulators.
test: ran both pipelines end-to-end on the real ~/Downloads/19440058502_ACTIVITY.fit
expecting: app number == web `GpxMetricsComputation.totalDistance` (raw)
next_action: none — diagnosis only (goal: find_root_cause_only). Hand to plan-phase --gaps.

## Symptoms

expected: |
  Importing ~/Downloads/19440058502_ACTIVITY.fit produces the same trail distance
  on web and in the Flutter app. Both clients compute distance/elevation
  client-side from the same server-transcoded GPX (Phase 34, PORT-05 / D-08).
actual: |
  Web: 10.51 km. App: 10.97 km. Divergence ~4.4% (app longer).
  Elevation gain/loss match EXACTLY: 344 m up, 351 m down.
errors: none — no crash, no toast, silent numeric divergence
reproduction: |
  Test 3 in .planning/phases/34-dart-conversion-port/34-UAT.md.
  Import the same .fit file once via the web trail-edit page picker and once via
  the Flutter app while online.
started: Discovered during UAT of Phase 34 (dart-conversion-port)

## Eliminated

- hypothesis: "Different earth model / haversine radius (turf vs geobase vs Vincenty)"
  evidence: |
    TS `haversineDistance` (web/src/lib/models/gpx/utils.ts:23-35) uses R = 6371 km.
    Dart `haversineMeters` (app/lib/util/gpx_conversion_util.dart:98-105) delegates to
    geobase `SphericalGreatCircle.distanceTo`, default radius 6371000.0 m — same model.
    Empirically the two accumulators agree to 5e-9 m over 10.97 km of track
    (TS raw 10971.376198346872 vs Dart raw 10971.376198347361).
  timestamp: 2026-08-01T15:05Z

- hypothesis: "Different smoothing/filtering before summing (one side filters, one doesn't)"
  evidence: |
    Both ARE the same filter. Dart `GpxMetricsComputation` is a line-for-line port of
    the TS class; both are constructed with thresholds (5, 5) and both compute
    totalDistance AND totalDistanceSmoothed. The defect is which of the two each
    CLIENT DISPLAYS, not how they are computed.
  timestamp: 2026-08-01T15:10Z

- hypothesis: "3D vs 2D distance (one side adds the elevation delta to segment length)"
  evidence: |
    Neither side uses elevation in any distance computation. Both accumulate pure
    great-circle horizontal hops. Verified by reading both accumulators end to end.
  timestamp: 2026-08-01T15:10Z

- hypothesis: "Segment/track-part boundary handling (auto-pause producing multiple <trkseg>)"
  evidence: |
    The transcoded GPX has exactly 1 <trk> with exactly 1 <trkseg> and 2569 <trkpt>.
    Both sides read trkCount=1, trksegCount=[1], pointCount=2569.
  timestamp: 2026-08-01T15:12Z

- hypothesis: "The two clients receive different GPX bytes (server transcode differs from web transcode)"
  evidence: |
    They receive byte-identical GPX. The convert endpoint
    (web/src/routes/api/v1/trail/convert/+server.ts:94) calls the SAME `fromFile()`
    the web page calls client-side (web/src/routes/trail/edit/[id]/+page.svelte:480).
    Both routes therefore run the same vendored FitParser and the same
    `GPX.toString()` serializer. First 3 points parsed on each side are identical
    to the last decimal:
      lat 48.72054937295616 / lon 8.24297639541328 / ele 298.6
  timestamp: 2026-08-01T15:12Z

- hypothesis: "Unit or rounding drift / double-counted lap"
  evidence: |
    Both display helpers do (m/1000).toFixed(2) in metric. No unit mismatch;
    the divergence is a genuinely different accumulator value.
  timestamp: 2026-08-01T15:12Z

## Evidence

- timestamp: 2026-08-01T15:05Z
  checked: |
    Ran the real web pipeline on the actual UAT file
    (~/Downloads/19440058502_ACTIVITY.fit, 307448 bytes) via a throwaway vitest
    spec: fromFile() -> gpx2trail(). Dumped the transcoded GPX for reuse.
  found: |
    distance_m (smoothed, what the web reports): 10553.460724712246  -> "10.55 km"
    rawDistance_m (totalDistance, last cumulativeDistance entry): 10971.376198346872 -> "10.97 km"
    elevation_gain: 344.1999999999997  -> "344 m"
    elevation_loss: 351.1999999999997  -> "351 m"
    pointCount: 2569, trkCount: 1, trksegCount: [1]
  implication: |
    The app's reported 10.97 km IS the web's own raw (unsmoothed) accumulator,
    to the centimetre. The app is not using a different formula — it is using
    the OTHER accumulator that already exists on both sides.

- timestamp: 2026-08-01T15:08Z
  checked: |
    Ran the Dart pipeline on that exact transcoded GPX via a throwaway flutter
    test: parseGpxSafely() -> computeTrailMetrics() / trailFromGpx().
  found: |
    pointCount: 2569 (identical to TS)
    metrics_distance / trail_distance (smoothed): 10553.460724716606
    acc_totalDistance (raw):                      10971.376198347361
    metrics_gain: 344.1999999999997 ; metrics_loss: 351.1999999999997
  implication: |
    The Dart port is bit-for-bit parity with TS (agrees to 5e-9 m). `Trail.distance`
    built by trailFromGpx is 10.55 km — the SAME value the web shows. So the port
    is correct and the persisted value is correct. The 10.97 km the user saw is
    produced somewhere else in the app.

- timestamp: 2026-08-01T15:15Z
  checked: |
    app/lib/components/trail/elevation_profile.dart — the widget
    trail_create_screen.dart:746 renders for an imported (unsaved) trail.
  found: |
    line 80:  final maxDist = _points.last.distanceM;
    line 143: formatDistance(maxDist, unit: ...)   <- the ruler-icon stat in the header
    _points comes from buildElevationTrackPoints() (line 661-720), whose loop is a
    plain unfiltered accumulator:
        final hop = haversineMeters(prevPoint, wpt);
        if (hop.isFinite) cumDist += hop;
    while the same header's up/down-arrow stats read trail.elevationGain /
    trail.elevationLoss (lines 126-131) — i.e. the SMOOTHED totals.
  implication: |
    One stats row mixes provenance: raw distance next to smoothed elevation.
    That is exactly the user's triple (10.97 km / 344 m / 351 m).

- timestamp: 2026-08-01T15:18Z
  checked: |
    Re-ran the Dart probe calling buildElevationTrackPoints(gpx, 5) directly on the
    same transcoded GPX.
  found: |
    chart_maxDist: 10971.376198347361  -> formatDistance -> "10.97 km"
    chart_points: 252 (downsampled for plotting; the distance axis is still raw
    cumulative over all 2569 source points)
  implication: |
    CONFIRMED end to end. The number the user read in the app is the elevation
    chart's plotting axis maximum, surfaced as if it were the trail's length.

- timestamp: 2026-08-01T15:19Z
  checked: |
    Where each surface gets its distance.
    App:  trail_panel.dart:211, trail_card.dart:378, trail_list_item.dart:359 ->
          formatDistance(trail.distance)  == 10.55 km (smoothed, correct)
          elevation_profile.dart:143      -> formatDistance(maxDist) == 10.97 km (raw)
          form_data_util.dart:25          -> persists trail.distance (10.55 km, correct)
    Web:  trail/edit/[id]/+page.svelte:2276 -> formatDistance($formData.distance)
          == features.distance == smoothed. map_with_elevation_maplibre.svelte has NO
          distance stat at all, so the web never surfaces the raw accumulator.
  implication: |
    Nothing is persisted wrongly. The bug is display-only and app-only, on the
    single screen the import flow lands on (trail_create_screen).

- timestamp: 2026-08-01T15:20Z
  checked: |
    Residual: user reported the web as 10.51 km; the measured web value is
    10553.46 m -> "10.55 km".
  found: |
    A 42 m (0.4%) unexplained residual. Every web display path found
    (formatDistance($formData.distance), updateTotals(), cropPreview totals) uses
    features.distance, and formatDistance is (m/1000).toFixed(2), so all of them
    render "10.55 km" for this file.
  implication: |
    Most likely a transcription slip in the UAT note. It does not affect the
    diagnosis: the ~4.4% gap under investigation is fully and exactly accounted
    for by raw-vs-smoothed (10.97 vs 10.55).

## Resolution

root_cause: |
  app/lib/components/trail/elevation_profile.dart:143 renders the elevation
  chart's RAW cumulative-distance axis maximum (`maxDist = _points.last.distanceM`,
  built by the unfiltered accumulator at lines 686-693) as the trail's distance
  stat, while the same header's elevation stats — and every other distance
  surface in both clients — use the 5 m-threshold SMOOTHED accumulator
  (`GpxMetricsComputation.totalDistanceSmoothed`). For a 1 Hz GPS recording with
  ~4.3 m mean point spacing, raw exceeds smoothed by ~4 %.
fix: [not applied — goal was find_root_cause_only]
verification: [n/a]
files_changed: []
