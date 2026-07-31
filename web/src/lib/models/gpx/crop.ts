// Extracted from `web/src/routes/trail/edit/[id]/+page.svelte`'s
// `updateCropMarkers()`/`getCoordinateAtDistance()`. This logic lived inline in a
// `.svelte` component's script block, where Vitest cannot reach it, which is why the
// NaN-coordinate defect described in `.planning/phases/33-conversion-correctness/33-VERIFICATION.md`
// (gap 2 / CR-01: `updateCropMarkers()`'s guard checked `!Number.isFinite(rawRouteTotal)`,
// missing the actual failure mode of `rawRouteTotal === 0` or a coincident leading point
// pair) shipped with no regression test. Moving it here makes it directly testable.
import type Waypoint from './waypoint';

/**
 * Binary-search + linear interpolation along a raw cumulative-distance array.
 * Returns `[lon, lat, index]`, where `index` is the point index of the segment's
 * later endpoint.
 */
export function getCoordinateAtDistance(
  points: Waypoint[],
  cumulative: number[],
  target: number,
): [number, number, number] {
  let low = 0,
    high = cumulative.length - 1;

  while (low < high) {
    const mid = Math.floor((low + high) / 2);
    if (cumulative[mid] < target) low = mid + 1;
    else high = mid;
  }

  const i = Math.max(1, low);
  const prevDist = cumulative[i - 1];
  const nextDist = cumulative[i];
  const ratio = (target - prevDist) / (nextDist - prevDist);

  const prev = points[i - 1];
  const next = points[i];

  return [
    prev.$.lon! + (next.$.lon! - prev.$.lon!) * ratio,
    prev.$.lat! + (next.$.lat! - prev.$.lat!) * ratio,
    i,
  ];
}
