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
  // Point at which lastFilteredZ was last set. Used only to ask "has the
  // device moved horizontally since the elevation anchor?" on the
  // retraction path below. Distinct from lastFilteredPointXY, which is the
  // distance-smoothing anchor — the two must never be merged or reused.
  private lastFilteredZPointXY: any | null = null;
  // Signed elevation delta of the most recent smoothed commit, still
  // eligible for retraction. 0 means nothing is retractable.
  private retractableDelta = 0;
  // The value lastFilteredZ held immediately before the retractable commit.
  private preRetractZ: number | null = null;
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
        this.lastFilteredZPointXY = point;
        this.lastZ = initialElevation;
      }
      // D-01: push once per call, including this first call, so
      // cumulativeDistance stays index-aligned with the point count —
      // entry 0 is always the route-start distance, 0.
      this.cumulativeDistance.push(this.totalDistance);
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

    if (Number.isFinite(distance)) {
      this.totalDistance += distance;
    }
    // D-01: unconditional push keeps cumulativeDistance's entry count equal
    // to the call count even when a hostile coordinate yields a non-finite
    // haversine distance (T-33-11).
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

    // Commit-then-retract noise filter: an elevation excursion is credited
    // immediately (a genuine climb is never under-reported), and is
    // *retracted* when the track returns to the pre-excursion elevation
    // WITHOUT having moved horizontally — the signature of altimeter/GPS
    // noise on a paused or stationary device, and the only thing that
    // distinguishes that from rolling terrain (which produces an
    // identical elevation series). Horizontal stillness is checked only on
    // the retraction path, so a monotonic low-horizontal climb (the
    // CONV-04 case) is never affected. Because a retraction only ever
    // subtracts the exact amount the immediately preceding commit added,
    // neither totalElevationGainSmoothed nor totalElevationLossSmoothed
    // can go negative.
    if (elevation !== undefined) {
      if (this.lastFilteredZ === null) {
        // This point establishes the smoothed anchor; no diff to record yet.
        this.lastFilteredZ = elevation;
        this.lastFilteredZPointXY = point;
        this.retractableDelta = 0;
        this.preRetractZ = null;
      } else {
        const elevationDiffSmoothed = elevation - this.lastFilteredZ;

        if (Math.abs(elevationDiffSmoothed) < this.thresholdZ_m) {
          // Below the noise floor — nothing to commit or retract.
        } else {
          const retractDistance =
            this.lastFilteredZPointXY !== null
              ? haversineDistance(
                  this.lastFilteredZPointXY.$.lat,
                  this.lastFilteredZPointXY.$.lon,
                  point.$.lat,
                  point.$.lon
                )
              : Infinity;

          const canRetract =
            this.retractableDelta !== 0 &&
            this.preRetractZ !== null &&
            Math.sign(elevationDiffSmoothed) === -Math.sign(this.retractableDelta) &&
            Math.abs(elevation - this.preRetractZ) < this.thresholdZ_m &&
            Number.isFinite(retractDistance) &&
            retractDistance < this.thresholdXY_m;

          if (canRetract) {
            if (this.retractableDelta > 0) {
              this.totalElevationGainSmoothed -= this.retractableDelta;
            } else {
              this.totalElevationLossSmoothed += this.retractableDelta;
            }
            this.lastFilteredZ = this.preRetractZ;
            this.lastFilteredZPointXY = point;
            this.retractableDelta = 0;
            this.preRetractZ = null;
          } else {
            if (elevationDiffSmoothed > 0) {
              this.totalElevationGainSmoothed += elevationDiffSmoothed;
            } else {
              this.totalElevationLossSmoothed -= elevationDiffSmoothed;
            }
            this.preRetractZ = this.lastFilteredZ;
            this.retractableDelta = elevationDiffSmoothed;
            this.lastFilteredZ = elevation;
            this.lastFilteredZPointXY = point;
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
