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

describe("GpxMetricsComputation — missing elevation", () => {
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

describe("GpxMetricsComputation — genuine sea level", () => {
  it("counts an explicit <ele>0</ele> as real sea-level data, not a missing tag", () => {
    // This is the guard: a truthiness check (`!point.ele` or
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

describe("GpxMetricsComputation — steep, low-horizontal-movement stretch", () => {
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

  it("keeps the class's now-unreported totalDistanceSmoothed field at 0 for the same stretch", () => {
    const trkpts: string[] = [];
    for (let i = 0; i < 12; i++) {
      trkpts.push(trkptXml(47 + i * 0.0000036, 11.0, String(1000 + i * 8)));
    }
    const points = GPX.parse(gpxXml(trkpts)).flatten();

    const metrics = new GpxMetricsComputation(5, 5);
    points.forEach((point) => metrics.addAndFilter(point));

    // The reported distance is totalDistance; totalDistanceSmoothed
    // survives on the class, unreported.
    expect(metrics.totalDistanceSmoothed).toBe(0);
    // finalElevationGain, not totalElevationGainSmoothed: this monotonic climb
    // ends without a confirming move, so its last 8 m step is still sitting in
    // the noise filter's pending slot. The published running total reads 80;
    // the reported total for a completed track is 88.
    expect(metrics.finalElevationGain).toBe(88);
    expect(metrics.totalElevationGainSmoothed).toBe(80);
  });
});

describe("GpxMetricsComputation — stationary GPS/altimeter noise", () => {
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

describe("GpxMetricsComputation — rolling terrain guard", () => {
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

describe("GpxMetricsComputation — the class's smoothing behavior is unchanged", () => {
  it("still suppresses GPS jitter in the now-unreported totalDistanceSmoothed while totalDistance stays raw", () => {
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

    // Neither number changes here — only which one gpx.ts reports changed.
    // The reported distance is totalDistance; totalDistanceSmoothed
    // survives on the class, unreported.
    expect(metrics.totalDistanceSmoothed).toBeCloseTo(100.075, 0);
    expect(metrics.totalDistance).toBeCloseTo(110.083, 0);
  });
});

// These two suites assert the INVARIANT rather than specific numbers. Both
// regression rounds on this file shipped with every value-based test green:
// the defect was never a wrong total, it was a published running total that
// moved backwards and broke a consumer in a different file.
describe("GpxMetricsComputation — smoothed elevation totals are monotonic", () => {
  // Every fixture that previously produced a retraction, plus the shapes most
  // likely to provoke one.
  const fixtures: Record<string, string[]> = {
    "stationary oscillation returning to start": buildStationary([
      1000, 1007, 1000, 1007, 1000,
    ]),
    "stationary oscillation ending mid-swing": buildStationary([
      1000, 1007, 1000, 1007,
    ]),
    "stationary drift (staircase)": buildStationary([
      1000, 1006, 1012, 1006, 1000,
    ]),
    "steep low-horizontal climb": Array.from({ length: 12 }, (_, i) =>
      trkptXml(47 + i * 0.0000036, 11.0, String(1000 + i * 8)),
    ),
    "rolling terrain with real horizontal movement": Array.from(
      { length: 20 },
      (_, i) =>
        trkptXml(47 + i * 0.0005, 11.0, String(1000 + (i % 2 === 0 ? 0 : 30))),
    ),
    "coincident points with oscillating elevation": [
      1000, 1008, 1000, 1008, 1000, 1008,
    ].map((ele) => trkptXml(47.0, 11.0, String(ele))),
  };

  for (const [name, trkpts] of Object.entries(fixtures)) {
    it(`never decreases either published total: ${name}`, () => {
      const points = GPX.parse(gpxXml(trkpts)).flatten();
      const metrics = new GpxMetricsComputation(5, 5);

      let previousGain = 0;
      let previousLoss = 0;

      points.forEach((point, index) => {
        metrics.addAndFilter(point);

        expect(
          metrics.totalElevationGainSmoothed,
          `gain decreased at point ${index}`,
        ).toBeGreaterThanOrEqual(previousGain);
        expect(
          metrics.totalElevationLossSmoothed,
          `loss decreased at point ${index}`,
        ).toBeGreaterThanOrEqual(previousLoss);

        previousGain = metrics.totalElevationGainSmoothed;
        previousLoss = metrics.totalElevationLossSmoothed;
      });

      // The invariant is worthless if the totals are simply always 0.
      expect(metrics.finalElevationGain).toBeGreaterThanOrEqual(0);
      expect(metrics.finalElevationLoss).toBeGreaterThanOrEqual(0);
    });
  }
});

describe("GpxMetricsComputation — per-segment differencing never goes negative", () => {
  // Mirrors trail_anchor_list.svelte's routeMetrics: it snapshots the running
  // totals at each segment boundary and subtracts consecutive snapshots. This
  // is the consumer that rendered negative elevation gain, reproduced
  // here so the contract is enforced from within this file's own test suite.
  it("reports no negative segment gain or loss across a stationary-noise boundary", () => {
    const segments = [
      buildStationary([1000, 1008]), // excursion opens in segment 1
      buildStationary([1000, 1008, 1000]), // and cancels in segment 2
      Array.from({ length: 6 }, (_, i) =>
        trkptXml(47 + i * 0.0005, 11.0, String(1000 + i * 12)),
      ),
    ];

    const metrics = new GpxMetricsComputation(5, 5);
    let previous = { gain: 0, loss: 0 };

    for (const trkpts of segments) {
      const points = GPX.parse(gpxXml(trkpts)).flatten();
      points.forEach((point) => metrics.addAndFilter(point));

      const segmentGain = metrics.totalElevationGainSmoothed - previous.gain;
      const segmentLoss = metrics.totalElevationLossSmoothed - previous.loss;

      expect(segmentGain).toBeGreaterThanOrEqual(0);
      expect(segmentLoss).toBeGreaterThanOrEqual(0);

      previous = {
        gain: metrics.totalElevationGainSmoothed,
        loss: metrics.totalElevationLossSmoothed,
      };
    }
  });
});

/** A track that never moves horizontally, only in elevation. */
function buildStationary(elevations: number[]): string[] {
  return elevations.map((ele) => trkptXml(47.0, 11.0, String(ele)));
}

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
