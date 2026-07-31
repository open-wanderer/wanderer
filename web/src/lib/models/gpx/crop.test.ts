import { describe, expect, it } from "vitest";
import { getCoordinateAtDistance } from "./crop";
import Waypoint from "./waypoint";

// Regression tests for VERIFICATION gap 2 (CR-01): the shipped crop-panel guard
// (`cumulativeRoute.length < 2 || !Number.isFinite(rawRouteTotal)`) never catches
// `rawRouteTotal === 0`, and `getCoordinateAtDistance()` itself divides `0 / 0` when
// two adjacent cumulative entries are equal, returning `[NaN, NaN, 1]`. In the app,
// `cropStartMarker.setLngLat([NaN, NaN])` then throws
// `Invalid LngLat object: (NaN, NaN)` inside MapLibre — an uncaught exception on
// crop-panel open, since `DoubleSlider` fires `onupdate([0, 100])` unconditionally on
// mount (target 0 is always queried first).

describe("getCoordinateAtDistance — degenerate interpolation bases", () => {
  it("returns a finite coordinate for a leading-duplicate-point route queried at 0%", () => {
    // Today: prevDist === nextDist === 0 at index 1, so ratio is 0/0 === NaN, and the
    // shipped code returns [NaN, NaN, 1] here.
    const points = [
      waypointAt(47.0, 11.0),
      waypointAt(47.0, 11.0), // coincident with the first point (routine GPS artefact)
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const cumulative = [0, 0, 111.19, 222.39];

    const [lon, lat] = getCoordinateAtDistance(points, cumulative, 0);

    expect(Number.isFinite(lon)).toBe(true);
    expect(Number.isFinite(lat)).toBe(true);
    expect(lon).toBeCloseTo(11, 6);
    expect(lat).toBeCloseTo(47, 6);
  });

  it("returns a finite coordinate for an all-identical-points route queried at 0%", () => {
    // Today: cumulative is entirely 0, so the same 0/0 division returns [NaN, NaN, 1].
    const points = [waypointAt(47.0, 11.0), waypointAt(47.0, 11.0), waypointAt(47.0, 11.0)];
    const cumulative = [0, 0, 0];

    const [lon, lat] = getCoordinateAtDistance(points, cumulative, 0);

    expect(Number.isFinite(lon)).toBe(true);
    expect(Number.isFinite(lat)).toBe(true);
    expect(lon).toBeCloseTo(11, 6);
    expect(lat).toBeCloseTo(47, 6);
  });
});

describe("getCoordinateAtDistance — normal (non-degenerate) route, faithfulness check", () => {
  const points = [waypointAt(47.0, 11.0), waypointAt(47.001, 11.0), waypointAt(47.002, 11.0)];
  const cumulative = [0, 111.19, 222.39];

  it("interpolates halfway along the second leg at 50%", () => {
    const [lon, lat] = getCoordinateAtDistance(points, cumulative, 166.79);

    expect(Number.isFinite(lon)).toBe(true);
    expect(Number.isFinite(lat)).toBe(true);
    expect(lat).toBeCloseTo(47.0015, 4);
    expect(lon).toBeCloseTo(11, 6);
  });

  it("returns the last point at 100%", () => {
    const [lon, lat] = getCoordinateAtDistance(points, cumulative, 222.39);

    expect(Number.isFinite(lon)).toBe(true);
    expect(Number.isFinite(lat)).toBe(true);
    expect(lat).toBeCloseTo(47.002, 6);
    expect(lon).toBeCloseTo(11, 6);
  });
});

function waypointAt(lat: number, lon: number): Waypoint {
  return new Waypoint({ $: { lat, lon } });
}
