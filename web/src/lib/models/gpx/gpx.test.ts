import { describe, expect, it } from "vitest";
import GPX from "./gpx";
import Track from "./track";
import TrackSegment from "./track-segment";
import Waypoint from "./waypoint";
import { haversineDistance } from "./utils";

describe("GPX.getTotals — CONV-01 first point of a segment", () => {
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

describe("GPX.getTotals — CONV-01/CONV-02 centroid and bounding box", () => {
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

  it("divides the centroid by exactly the number of points it summed (CONV-02 invariant)", () => {
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

describe("GPX.getTotals — CONV-01 multi-leg planner route", () => {
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
