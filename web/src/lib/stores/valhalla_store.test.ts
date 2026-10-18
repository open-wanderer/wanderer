import { beforeEach, describe, expect, it } from "vitest";
import GPX from "$lib/models/gpx/gpx";
import { deleteFromRoute, setRoute, valhallaStore } from "./valhalla_store.svelte";

/**
 * `features` is a cache refreshed by `getTotals()`, while `flatten()` reads the
 * live geometry. Crop interpolation indexes `features.cumulativeDistance`
 * against `flatten()`, so the two drifting apart silently places crop pins on
 * the wrong points.
 *
 * `deleteFromRoute` used to recompute the snapshot's features from
 * `valhallaStore.route` — the object it had NOT just spliced — which is exactly
 * how that drift was introduced.
 */
describe("deleteFromRoute — features stay consistent with the mutated route", () => {
  beforeEach(() => {
    setRoute(GPX.parse(gpxXml([segmentXml(47.0), segmentXml(47.01), segmentXml(47.02)])));
  });

  it("keeps cumulativeDistance index-aligned with the remaining points", () => {
    deleteFromRoute(1);

    expect(valhallaStore.route.features.cumulativeDistance.length).toBe(
      valhallaStore.route.flatten().length,
    );
  });

  it("shrinks the reported distance after a segment is removed", () => {
    const before = valhallaStore.route.features.distance ?? 0;
    expect(before).toBeGreaterThan(0);

    deleteFromRoute(1);

    // Stale-cache symptom: the pre-splice total survived the delete unchanged.
    expect(valhallaStore.route.features.distance ?? 0).toBeLessThan(before);
  });
});

function segmentXml(baseLat: number): string {
  const trkpts = Array.from({ length: 4 }, (_, i) =>
    `<trkpt lat="${baseLat + i * 0.001}" lon="11.0"><ele>${1000 + i * 5}</ele></trkpt>`,
  );
  return `<trkseg>${trkpts.join("")}</trkseg>`;
}

function gpxXml(trksegElements: string[]): string {
  return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer-test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    ${trksegElements.join("\n    ")}
  </trk>
</gpx>`;
}
