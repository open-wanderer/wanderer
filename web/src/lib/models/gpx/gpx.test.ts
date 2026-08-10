import { describe, expect, it } from "vitest";
import GPX from "./gpx";
import Track from "./track";
import TrackSegment from "./track-segment";
import Waypoint from "./waypoint";
import { haversineDistance } from "./utils";

describe("GPX.getTotals — first point of a segment", () => {
  it("reports the full hop length of a 2-point segment instead of 0", () => {
    const a = waypointAt(47.0, 11.0);
    const b = waypointAt(47.001, 11.001);
    const gpx = gpxFromSegments([[a, b]]);

    // Pre-fix value was exactly 0 — the loop started at i = 1, so the only
    // point of the segment ever fed to metrics.addAndFilter() was `b`, which
    // is the metrics instance's very first call and only initializes its
    // anchors (no distance is added on that call).
    expect(gpx.features.distance).toBeCloseTo(hopMetres(a, b), 1);
    expect(gpx.features.distance).toBeCloseTo(134.592, 1);
  });
});

describe("GPX.getTotals — centroid and bounding box", () => {
  it("includes every point, including the geographic-extreme first point, in the bounding box", () => {
    const first = waypointAt(40.0, 10.0);
    const second = waypointAt(47.0, 11.0);
    const third = waypointAt(48.0, 12.0);
    const gpx = gpxFromSegments([[first, second, third]]);

    // Pre-fix, `first` (the segment's own first point, and also the
    // geographic extreme) was skipped by the i = 1 loop bound, so the
    // bounding box reported 47.0 / 11.0 instead of 40.0 / 10.0.
    expect(gpx.features.boundingBox.minLat).toBe(40.0);
    expect(gpx.features.boundingBox.minLon).toBe(10.0);
    expect(gpx.features.boundingBox.maxLat).toBe(48.0);
    expect(gpx.features.boundingBox.maxLon).toBe(12.0);

    expect(gpx.features.centroid.lat).toBeCloseTo(45.0, 6);
    expect(gpx.features.centroid.lon).toBeCloseTo(11.0, 6);
  });

  it("divides the centroid by exactly the number of points it summed", () => {
    const points = [
      waypointAt(40.0, 10.0),
      waypointAt(47.0, 11.0),
      waypointAt(48.0, 12.0),
    ];
    const gpx = gpxFromSegments([points]);

    const meanLat = points.reduce((sum, p) => sum + (p.$.lat ?? 0), 0) / points.length;
    const meanLon = points.reduce((sum, p) => sum + (p.$.lon ?? 0), 0) / points.length;

    expect(gpx.features.centroid.lat).toBeCloseTo(meanLat, 6);
    expect(gpx.features.centroid.lon).toBeCloseTo(meanLon, 6);
  });
});

describe("GPX.getTotals — multi-leg planner route", () => {
  it("reports the full polyline across both legs instead of dropping the opening hop", () => {
    // Shaped like valhalla_store.svelte.ts's insertIntoRoute() output: each
    // planner leg is its own TrackSegment, and the shared anchor point is
    // deliberately repeated as the next leg's first point.
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);

    const expectedTotal =
      hopMetres(leg1[0], leg1[1]) +
      hopMetres(leg1[1], leg1[2]) +
      hopMetres(leg2[0], leg2[1]) +
      hopMetres(leg2[1], leg2[2]);

    // Pre-fix value was 333.585 — the i = 1 loop bound dropped each
    // segment's own first point (leg1's opening hop and leg2's zero-length
    // anchor duplicate), losing one real hop's worth of distance.
    expect(gpx.features.distance).toBeCloseTo(expectedTotal, 1);
    expect(gpx.features.distance).toBeCloseTo(444.78, 1);
  });

  it("includes leg 1's opening point in the bounding box", () => {
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);

    expect(gpx.features.boundingBox.minLat).toBe(47.0);
  });
});

describe("GPX.getTotals — zero-point regression guard", () => {
  it("keeps the pre-existing sentinel behavior for a track with no points (not a fix)", () => {
    const gpx = gpxFromSegments([[]]);

    expect(Number.isNaN(gpx.features.centroid.lat)).toBe(true);
    expect(gpx.features.boundingBox.minLat).toBe(Infinity);
    expect(gpx.features.boundingBox.maxLat).toBe(-Infinity);
    expect(gpx.features.distance).toBe(0);
    expect(gpx.flatten()).toHaveLength(0);
  });
});

describe("GPX.getTotals — reports the raw accumulator", () => {
  it("reports the raw jitter-inflated sum, not the smoothed forward travel, for a jittery track", () => {
    // Single segment: forward hop (~20 m) then a jitter out-and-back
    // (~1 m each way) that never clears the 5 m threshold, repeated 5
    // times. The raw haversine sum over every consecutive pair (what
    // cumulativeDistance's last entry holds) is ~110.083 m — this is what
    // getTotals() reports as `distance`
    // (2026-08-01). The now-unreported smoothed accumulator, which held the
    // real forward travel with jitter suppressed, is ~100.075 m. Pre-33-01
    // (i = 1 loop bug), the reported value was 90.068 m — a different
    // defect entirely.
    const points = [waypointAt(47.0, 11.0)];
    let lat = 47.0;
    for (let i = 0; i < 5; i++) {
      lat += 0.00018;
      points.push(waypointAt(lat, 11.0));
      lat += 0.000009;
      points.push(waypointAt(lat, 11.0));
      lat -= 0.000009;
      points.push(waypointAt(lat, 11.0));
    }
    const gpx = gpxFromSegments([points]);

    expect(gpx.features.distance).toBeCloseTo(110.083, 0);
  });

  it("equals the last cumulativeDistance entry — same accumulator, by construction", () => {
    const points = [waypointAt(47.0, 11.0)];
    let lat = 47.0;
    for (let i = 0; i < 5; i++) {
      lat += 0.00018;
      points.push(waypointAt(lat, 11.0));
      lat += 0.000009;
      points.push(waypointAt(lat, 11.0));
      lat -= 0.000009;
      points.push(waypointAt(lat, 11.0));
    }
    const gpx = gpxFromSegments([points]);

    const rawTotal =
      gpx.features.cumulativeDistance[gpx.features.cumulativeDistance.length - 1];

    // Executable invariant, inverted (not deleted) now that smoothing is
    // superseded: addAndFilter pushes this.totalDistance onto
    // cumulativeDistance immediately after every totalDistance += call, so
    // the reported distance and the raw cumulative array's last entry are
    // provably the same accumulator — strict equality, not a closeness
    // matcher.
    expect(gpx.features.distance).toBe(rawTotal);
  });
});

describe("GPX.getTotals — cumulativeDistance index alignment", () => {
  it("index-aligns a 2-point segment with a leading 0 entry", () => {
    const a = waypointAt(47.0, 11.0);
    const b = waypointAt(47.001, 11.001);
    const gpx = gpxFromSegments([[a, b]]);

    expect(gpx.features.cumulativeDistance).toHaveLength(gpx.flatten().length);
    expect(gpx.features.cumulativeDistance).toHaveLength(2);
    expect(gpx.features.cumulativeDistance[0]).toBe(0);
    expect(gpx.features.cumulativeDistance[1]).toBeCloseTo(134.592, 1);
  });

  it("index-aligns the two-leg planner route, staying non-decreasing across the shared anchor", () => {
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);
    const cumulative = gpx.features.cumulativeDistance;

    expect(cumulative).toHaveLength(gpx.flatten().length);
    expect(cumulative).toHaveLength(6);
    expect(cumulative[0]).toBe(0);
    expect(cumulative[cumulative.length - 1]).toBeCloseTo(444.78, 1);

    for (let i = 1; i < cumulative.length; i++) {
      expect(cumulative[i]).toBeGreaterThanOrEqual(cumulative[i - 1]);
    }
  });

  it("stays empty for a track with no points", () => {
    const gpx = gpxFromSegments([[]]);

    expect(gpx.features.cumulativeDistance).toHaveLength(gpx.flatten().length);
    expect(gpx.features.cumulativeDistance).toEqual([]);
  });
});

describe("GPX.getTotals — smoothing does not regress the planner route", () => {
  it("still reports the full polyline distance when every hop clears the smoothing threshold", () => {
    // Every hop in this fixture exceeds the 5 m threshold, so raw and
    // smoothed totals coincide — this value is unaffected by the
    // supersession (the reported distance is now raw, but raw and smoothed
    // agree here) and stays at 33-01's baseline.
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);

    expect(gpx.features.distance).toBeCloseTo(444.78, 1);
  });
});

// An unparseable-but-non-empty <time> body used to become an
// `Invalid Date`, which is a truthy object — so getTotals()'s
// `startTime && endTime` guard passed and produced a NaN duration, while the
// Dart port reported 0 for the same document. Times are now parsed with the
// grammar Dart's DateTime.parse accepts.
describe("Waypoint.time — Dart-aligned <time> parsing", () => {
  const rejected: Array<[string, string]> = [
    ["a non-numeric body", "N/A"],
    ["a whitespace-only body", "   "],
    ["a legacy US format V8 accepts but Dart does not", "Jan 1 2024"],
    ["a slash-separated date V8 accepts but Dart does not", "2024/01/01"],
    ["a bare year", "2024"],
  ];

  for (const [label, raw] of rejected) {
    it(`leaves time undefined for ${label}`, () => {
      // @ts-expect-error xml2js hands the constructor a raw string here.
      expect(new Waypoint({ $: { lat: 47, lon: 11 }, time: raw }).time).toBeUndefined();
    });
  }

  it("still accepts a conforming ISO-8601 instant", () => {
    // @ts-expect-error xml2js hands the constructor a raw string here.
    const wpt = new Waypoint({ $: { lat: 47, lon: 11 }, time: "2024-01-01T10:30:00Z" });
    expect(wpt.time?.toISOString()).toBe("2024-01-01T10:30:00.000Z");
  });

  it("reports a 0 duration - never NaN - when the start time is unparseable", () => {
    // @ts-expect-error xml2js hands the constructor a raw string here.
    const start = new Waypoint({ $: { lat: 47.0, lon: 11.0 }, time: "N/A" });
    // @ts-expect-error xml2js hands the constructor a raw string here.
    const end = new Waypoint({ $: { lat: 47.001, lon: 11.001 }, time: "2024-01-01T10:30:00Z" });
    const gpx = gpxFromSegments([[start, end]]);

    expect(Number.isNaN(gpx.features.duration)).toBe(false);
    expect(gpx.features.duration).toBe(0);
  });
});

function waypointAt(lat: number, lon: number, ele?: number): Waypoint {
  return new Waypoint({ $: { lat, lon }, ele });
}

function gpxFromSegments(segments: Waypoint[][]): GPX {
  return new GPX({
    trk: [
      new Track({
        trkseg: segments.map(points => new TrackSegment({ trkpt: points })),
      }),
    ],
  });
}

function hopMetres(a: Waypoint, b: Waypoint): number {
  return haversineDistance(a.$.lat ?? 0, a.$.lon ?? 0, b.$.lat ?? 0, b.$.lon ?? 0);
}
