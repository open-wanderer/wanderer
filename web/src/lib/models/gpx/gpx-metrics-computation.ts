import { haversineDistance } from './utils';

/**
 * Coerces a raw `<ele>` value into a finite elevation in metres, or `undefined`
 * when no genuine elevation reading is present.
 *
 * `<ele>` is XML element text, never coerced by xml2js's `attrValueProcessors`
 * (those only touch attributes), so `point.ele` is a `string` for every
 * XML-parsed GPX (`"1000"`, `"0"`, `""`) and only a real `number` for
 * programmatically constructed waypoints. This is the single coercion point:
 * `undefined`/`null`/empty-or-whitespace strings/non-numeric strings/`NaN`/
 * `Infinity` all map to `undefined` ("no data"), while a genuine `0` (numeric
 * or `"0"`) is preserved as real sea-level data.
 */
export function parseElevation(raw: unknown): number | undefined {
  if (raw === undefined || raw === null) {
    return undefined;
  }
  if (typeof raw === 'string' && raw.trim() === '') {
    return undefined;
  }
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : undefined;
}

class GpxMetricsComputation {
  private readonly thresholdXY_m: number;  // Distance threshold for filtering on the XY axis (latitude / longitude)
  private readonly thresholdZ_m: number;  // Distance threshold for filtering on the Z axis (elevation)
  private lastPointXY: any | null = null;
  private lastFilteredPointXY: any | null = null;
  private lastFilteredZ: number | null = null;
  private lastZ: number | null = null;
  totalElevationGain = 0;
  totalElevationLoss = 0;
  totalElevationGainSmoothed = 0;
  totalElevationLossSmoothed = 0;
  totalDistance = 0;
  totalDistanceSmoothed = 0;
  cumulativeDistance: number[] = []

  constructor(thresholdXY_m: number, thresholdZ_m: number) {
    this.thresholdXY_m = thresholdXY_m;
    this.thresholdZ_m = thresholdZ_m;
  }

  addAndFilter(point: any) {
    if (!this.lastPointXY || !this.lastFilteredPointXY) {
      // Initialize raw and smoothed anchors with the first point. When the
      // first point has no usable elevation, leave both anchors `null` so
      // the first point that *does* carry elevation becomes the anchor
      // instead of diffing against a fabricated `0`.
      this.lastPointXY = point;
      this.lastFilteredPointXY = point;
      const initialElevation = parseElevation(point.ele);
      if (initialElevation !== undefined) {
        this.lastFilteredZ = initialElevation;
        this.lastZ = initialElevation;
      }
      return;
    }

    const distance = haversineDistance(
      this.lastPointXY.$.lat,
      this.lastPointXY.$.lon,
      point.$.lat,
      point.$.lon
    );

    const smoothedDistance = haversineDistance(
      this.lastFilteredPointXY.$.lat,
      this.lastFilteredPointXY.$.lon,
      point.$.lat,
      point.$.lon
    );

    this.totalDistance += distance;
    this.cumulativeDistance.push(this.totalDistance)

    this.lastPointXY = point;

    const elevation = parseElevation(point.ele);
    if (elevation !== undefined) {
      if (this.lastZ === null) {
        // This point establishes the raw anchor; no diff to record yet.
        this.lastZ = elevation;
      } else {
        const elevationDiff = elevation - this.lastZ;
        this.lastZ = elevation;
        if (elevationDiff > 0) {
          this.totalElevationGain += elevationDiff;
        }
        if (elevationDiff < 0) {
          this.totalElevationLoss -= elevationDiff;
        }
      }
    }

    if (elevation !== undefined) {
      if (this.lastFilteredZ === null) {
        // This point establishes the smoothed anchor; no diff to record yet.
        this.lastFilteredZ = elevation;
      } else {
        const elevationDiffSmoothed = elevation - this.lastFilteredZ;

        if (Math.abs(elevationDiffSmoothed) >= this.thresholdZ_m) {
          this.lastFilteredZ = elevation;
          if (elevationDiffSmoothed > 0) {
            this.totalElevationGainSmoothed += elevationDiffSmoothed;
          } else {
            this.totalElevationLossSmoothed -= elevationDiffSmoothed;
          }
        }
      }
    }

    if (smoothedDistance >= this.thresholdXY_m) {
      this.totalDistanceSmoothed += smoothedDistance;
      this.lastFilteredPointXY = point;
    }
  }
}

export default GpxMetricsComputation;
