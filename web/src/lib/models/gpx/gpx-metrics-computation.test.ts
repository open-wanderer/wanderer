import { describe, expect, it } from "vitest";
import GPX from "./gpx";
import GpxMetricsComputation, { parseElevation } from "./gpx-metrics-computation";

describe("parseElevation", () => {
  it("coerces accepted values to their numeric elevation", () => {
    expect(parseElevation(1000)).toBe(1000);
    expect(parseElevation("1000")).toBe(1000);
    expect(parseElevation(0)).toBe(0);
    expect(parseElevation("0")).toBe(0);
  });

  it("returns undefined for anything that is not genuine elevation data", () => {
    expect(parseElevation(undefined)).toBeUndefined();
    expect(parseElevation(null)).toBeUndefined();
    expect(parseElevation("")).toBeUndefined();
    expect(parseElevation("   ")).toBeUndefined();
    expect(parseElevation("abc")).toBeUndefined();
    expect(parseElevation(Number.NaN)).toBeUndefined();
    expect(parseElevation(Infinity)).toBeUndefined();
    expect(parseElevation(-Infinity)).toBeUndefined();
  });
});

describe("GpxMetricsComputation — CONV-03 missing elevation", () => {
  it("carries elevation forward across a point whose <ele> tag is omitted", () => {
    const xml = gpxXml([
      trkptXml(47.000, 11.0, "1000"),
      trkptXml(47.001, 11.0, "1005"),
      trkptXml(47.002, 11.0), // <ele> omitted entirely
      trkptXml(47.003, 11.0, "1010"),
      trkptXml(47.004, 11.0, "1015"),
    ]);
    const gpx = GPX.parse(xml);

    // Pre-fix values on this exact fixture: elevationGain 1015, elevationLoss 1005 —
    // the missing tag coerced to 0, fabricating a plunge to sea level and back.
    expect(gpx.features.elevationGain).toBe(15);
    expect(gpx.features.elevationLoss).toBe(0);
  });

  it("treats an empty <ele></ele> tag identically to an omitted one, not as sea level", () => {
    const xml = gpxXml([
      trkptXml(47.000, 11.0, "1000"),
      trkptXml(47.001, 11.0, "1005"),
      trkptXml(47.002, 11.0, ""), // explicit empty <ele></ele>
      trkptXml(47.003, 11.0, "1010"),
      trkptXml(47.004, 11.0, "1015"),
    ]);
    const gpx = GPX.parse(xml);

    expect(gpx.features.elevationGain).toBe(15);
    expect(gpx.features.elevationLoss).toBe(0);
  });

  it("documents the parser's real output for omitted vs. empty <ele>, so parseElevation is not simplified away", () => {
    const omittedXml = gpxXml([
      trkptXml(47.000, 11.0, "1000"),
      trkptXml(47.001, 11.0, "1005"),
      trkptXml(47.002, 11.0),
    ]);
    const emptyXml = gpxXml([
      trkptXml(47.000, 11.0, "1000"),
      trkptXml(47.001, 11.0, "1005"),
      trkptXml(47.002, 11.0, ""),
    ]);

    expect(GPX.parse(omittedXml).flatten()[2].ele).toBeUndefined();
    expect(GPX.parse(emptyXml).flatten()[2].ele).toBe("");
  });
});

describe("GpxMetricsComputation — CONV-03 genuine sea level", () => {
  it("counts an explicit <ele>0</ele> as real sea-level data, not a missing tag", () => {
    // This is the Pitfall-3 guard: a truthiness check (`!point.ele` or
    // `point.ele || 0`) would wrongly treat this genuine 0 m reading as missing.
    const xml = gpxXml([
      trkptXml(47.000, 11.0, "0"),
      trkptXml(47.001, 11.0, "10"),
    ]);
    const gpx = GPX.parse(xml);

    expect(gpx.features.elevationGain).toBe(10);
    expect(gpx.features.elevationLoss).toBe(0);
  });
});

describe("GpxMetricsComputation — CONV-04 steep, low-horizontal-movement stretch", () => {
  it("registers the full climb of an 88 m scramble spread over ~4.4 m of horizontal movement", () => {
    const trkpts: string[] = [];
    for (let i = 0; i < 12; i++) {
      trkpts.push(trkptXml(47 + i * 0.0000036, 11.0, String(1000 + i * 8)));
    }
    const gpx = GPX.parse(gpxXml(trkpts));

    // Pre-fix value on this exact fixture: elevationGain 0 — the smoothed
    // elevation diff was gated behind the horizontal threshold, which this
    // stretch never clears.
    expect(gpx.features.elevationGain).toBe(88);
    expect(gpx.features.elevationLoss).toBe(0);
  });

  it("keeps totalDistanceSmoothed at 0 for the same stretch (distance smoothing must stay gated)", () => {
    const trkpts: string[] = [];
    for (let i = 0; i < 12; i++) {
      trkpts.push(trkptXml(47 + i * 0.0000036, 11.0, String(1000 + i * 8)));
    }
    const points = GPX.parse(gpxXml(trkpts)).flatten();

    const metrics = new GpxMetricsComputation(5, 5);
    points.forEach((point) => metrics.addAndFilter(point));

    expect(metrics.totalDistanceSmoothed).toBe(0);
    expect(metrics.totalElevationGainSmoothed).toBe(88);
  });
});

describe("GpxMetricsComputation — CONV-04 stationary GPS/altimeter noise", () => {
  it("reports elevationGain === 0 and elevationLoss === 0 for a fully-stationary track whose altitude oscillates +/-7 m and returns to its starting elevation", () => {
    // 61 samples, identical lat/lon, altitude alternating 1000/1007, ends at 1000.
    const trkpts: string[] = [];
    for (let i = 0; i <= 60; i++) {
      trkpts.push(trkptXml(47.0, 11.0, String(i % 2 === 0 ? 1000 : 1007)));
    }
    const gpx = GPX.parse(gpxXml(trkpts));

    // Pre-fix value on this exact fixture: elevationGain 210, elevationLoss 210 —
    // the flat threshold commit rule ratchets on every +/-7 m swing even though
    // the track never moves and returns exactly to its starting elevation.
    expect(gpx.features.elevationGain).toBe(0);
    expect(gpx.features.elevationLoss).toBe(0);
    expect(gpx.features.elevationGain).toBeGreaterThanOrEqual(0);
    expect(gpx.features.elevationLoss).toBeGreaterThanOrEqual(0);
  });

  it("reports exactly the one un-cancelled excursion when a stationary +/-7 m oscillation ends mid-swing", () => {
    // Same generator truncated to 60 trackpoints — ends at 1007, one swing
    // un-cancelled. 7 m is the track's genuine net displacement.
    const trkpts: string[] = [];
    for (let i = 0; i < 60; i++) {
      trkpts.push(trkptXml(47.0, 11.0, String(i % 2 === 0 ? 1000 : 1007)));
    }
    const gpx = GPX.parse(gpxXml(trkpts));

    // Pre-fix value on this exact fixture: elevationGain 210, elevationLoss 203.
    expect(gpx.features.elevationGain).toBe(7);
    expect(gpx.features.elevationLoss).toBe(0);
    expect(gpx.features.elevationGain).toBeGreaterThanOrEqual(0);
    expect(gpx.features.elevationLoss).toBeGreaterThanOrEqual(0);
  });

  it("rejects a stationary out-and-back bump but measures the genuine climb that follows in full", () => {
    // 6 trackpoints, all at the same lat/lon, elevations 1000, 1008, 1000,
    // 1008, 1016, 1024.
    const trkpts: string[] = [
      trkptXml(47.0, 11.0, "1000"),
      trkptXml(47.0, 11.0, "1008"),
      trkptXml(47.0, 11.0, "1000"),
      trkptXml(47.0, 11.0, "1008"),
      trkptXml(47.0, 11.0, "1016"),
      trkptXml(47.0, 11.0, "1024"),
    ];
    const gpx = GPX.parse(gpxXml(trkpts));

    // Pre-fix value on this exact fixture: elevationGain 32, elevationLoss 8.
    expect(gpx.features.elevationGain).toBe(24);
    expect(gpx.features.elevationLoss).toBe(0);
    expect(gpx.features.elevationGain).toBeGreaterThanOrEqual(0);
    expect(gpx.features.elevationLoss).toBeGreaterThanOrEqual(0);
  });
});

describe("GpxMetricsComputation — CONV-04 rolling terrain guard", () => {
  it("still reports full gain and loss for rolling terrain with genuine horizontal movement (noise rejection never eats real terrain)", () => {
    // 6 trackpoints spaced ~100 m apart, elevations 1000, 1008, 1000, 1008,
    // 1000, 1008. Green before AND after the noise-tolerant filter lands.
    const trkpts: string[] = [];
    const elevations = [1000, 1008, 1000, 1008, 1000, 1008];
    for (let i = 0; i < 6; i++) {
      trkpts.push(trkptXml(47 + i * 0.0009, 11.0, String(elevations[i])));
    }
    const gpx = GPX.parse(gpxXml(trkpts));

    expect(gpx.features.elevationGain).toBe(24);
    expect(gpx.features.elevationLoss).toBe(16);
    expect(gpx.features.elevationGain).toBeGreaterThanOrEqual(0);
    expect(gpx.features.elevationLoss).toBeGreaterThanOrEqual(0);
  });
});

describe("GpxMetricsComputation — distance smoothing is unchanged", () => {
  it("suppresses GPS jitter in totalDistanceSmoothed while totalDistance stays raw", () => {
    const trkpts: string[] = [trkptXml(47.0, 11.0)];
    let lat = 47.0;
    for (let i = 0; i < 5; i++) {
      lat += 0.00018;
      trkpts.push(trkptXml(lat, 11.0));
      lat += 0.000009;
      trkpts.push(trkptXml(lat, 11.0));
      lat -= 0.000009;
      trkpts.push(trkptXml(lat, 11.0));
    }
    const points = GPX.parse(gpxXml(trkpts)).flatten();

    const metrics = new GpxMetricsComputation(5, 5);
    points.forEach((point) => metrics.addAndFilter(point));

    expect(metrics.totalDistanceSmoothed).toBeCloseTo(100.075, 0);
    expect(metrics.totalDistance).toBeCloseTo(110.083, 0);
  });
});

function trkptXml(lat: number, lon: number, ele?: string): string {
  const eleElement = ele !== undefined ? `<ele>${ele}</ele>` : "";
  return `<trkpt lat="${lat}" lon="${lon}">${eleElement}</trkpt>`;
}

function gpxXml(trkptElements: string[]): string {
  return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
      ${trkptElements.join("\n      ")}
    </trkseg>
  </trk>
</gpx>`;
}
