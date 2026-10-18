// Extracted from `web/src/routes/trail/edit/[id]/+page.svelte`'s
// `updateCropMarkers()`/`getCoordinateAtDistance()`. This logic lived inline in a
// `.svelte` component's script block, where Vitest cannot reach it, which is why a
// NaN-coordinate defect shipped with no regression test: `updateCropMarkers()`'s guard
// checked `!Number.isFinite(rawRouteTotal)`, missing the actual failure mode of
// `rawRouteTotal === 0` or a coincident leading point pair. Moving it here makes it
// directly testable.
import type Waypoint from './waypoint';

/**
 * Binary-search + linear interpolation along a raw cumulative-distance array.
 * Returns `[lon, lat, index]`, where `index` is the point index of the segment's
 * later endpoint.
 *
 * Degenerate-safe (never returns a non-finite number): a coincident-point pair
 * (`nextDist - prevDist === 0`) resolves to `ratio = 0` instead of dividing `0 / 0`,
 * and an empty/too-short input is handled with an explicit early return rather than
 * an out-of-bounds array read.
 */
export function getCoordinateAtDistance(
  points: Waypoint[],
  cumulative: number[],
  target: number,
): [number, number, number] {
  if (points.length === 0) {
    return [0, 0, 0];
  }
  if (points.length < 2 || cumulative.length < 2) {
    const only = points[0];
    return [only.$.lon ?? 0, only.$.lat ?? 0, 0];
  }

  let low = 0,
    high = cumulative.length - 1;

  while (low < high) {
    const mid = Math.floor((low + high) / 2);
    if (cumulative[mid] < target) low = mid + 1;
    else high = mid;
  }

  // Clamp to a valid adjacent pair within `points`, in case `cumulative` is longer
  // than `points` (shouldn't happen, but keeps the function total either way).
  const i = Math.max(1, Math.min(low, points.length - 1));
  const prevDist = cumulative[i - 1];
  const nextDist = cumulative[i];

  // The direct cause of the shipped [NaN, NaN, 1] defect: nextDist - prevDist === 0
  // (a coincident point pair, a routine GPS artefact) used to divide 0 / 0. Guarding
  // on `span > 0` (not `span !== 0`) also prevents a negative extrapolation if a
  // caller ever hands in a non-non-decreasing cumulative array.
  const span = nextDist - prevDist;
  const ratio = span > 0 ? (target - prevDist) / span : 0;

  const prev = points[i - 1];
  const next = points[i];

  // `Waypoint` already defaults missing `$.lat`/`$.lon` to -1 (never `undefined`), so
  // the `?? 0` here is defence in depth for the never-NaN invariant, not a behaviour
  // the app actually relies on.
  const prevLon = prev.$.lon ?? 0;
  const prevLat = prev.$.lat ?? 0;
  const nextLon = next.$.lon ?? 0;
  const nextLat = next.$.lat ?? 0;

  return [prevLon + (nextLon - prevLon) * ratio, prevLat + (nextLat - prevLat) * ratio, i];
}

/**
 * `true` only when `cumulative` can support interpolation: length >= 2 and a
 * finite last entry strictly greater than 0.
 *
 * The `> 0` check is the condition the shipped `updateCropMarkers()` guard was
 * missing (it only checked `!Number.isFinite(rawRouteTotal)`, which never rejects
 * `rawRouteTotal === 0`). A `true` result does NOT imply every adjacent span is
 * non-zero — a route can have a healthy positive total with a coincident leading
 * pair — which is why `getCoordinateAtDistance()`'s own `span > 0` guard above is
 * also required; this predicate alone cannot close that gap.
 */
export function hasCropInterpolationBasis(cumulative: number[]): boolean {
  const total = cumulative[cumulative.length - 1];
  return cumulative.length >= 2 && Number.isFinite(total) && total > 0;
}
