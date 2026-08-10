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
  // discard path below. Distinct from lastFilteredPointXY, which is the
  // distance-smoothing anchor — the two must never be merged or reused.
  private lastFilteredZPointXY: any | null = null;
  // Signed elevation delta of the most recent above-threshold move, held
  // back from the published totals until it is confirmed. 0 means nothing
  // is pending. See the defer-then-publish note in addAndFilter().
  private pendingDelta = 0;
  // The value lastFilteredZ held immediately before the pending excursion.
  private pendingAnchorZ: number | null = null;
  totalElevationGain = 0;
  totalElevationLoss = 0;
  // INVARIANT: these two are monotonically non-decreasing running totals.
  // trail_anchor_list.svelte derives per-segment metrics by subtracting
  // consecutive snapshots of them, so any code path that decrements them
  // makes that component render negative elevation gain. Only ever `+=`.
  // The provisional (not-yet-confirmed) excursion lives in pendingDelta and
  // is surfaced through finalElevationGain/finalElevationLoss instead.
  totalElevationGainSmoothed = 0;
  totalElevationLossSmoothed = 0;
  totalDistance = 0;
  // NOT REPORTED as of 2026-08-01: no consumer reads this for a trail's
  // distance anymore — getTotals() reports the raw totalDistance instead.
  // Kept, and thresholdXY_m/GpxMetricsComputation(5, 5) untouched, because
  // lastFilteredPointXY sits in a class whose other anchors are
  // elevation-critical, and a future speed-aware filter would build on it.
  totalDistanceSmoothed = 0;
  cumulativeDistance: number[] = []

  constructor(thresholdXY_m: number, thresholdZ_m: number) {
    this.thresholdXY_m = thresholdXY_m;
    this.thresholdZ_m = thresholdZ_m;
  }

  /**
   * Smoothed elevation gain including a still-pending excursion.
   *
   * This — not `totalElevationGainSmoothed` — is a completed track's reported
   * gain. A track that ends mid-swing (climbs and stops) has its final climb
   * sitting in the pending slot, unconfirmed but real.
   *
   * Consumers that difference successive readings to derive per-segment
   * metrics must use the monotonic `totalElevationGainSmoothed` instead; this
   * getter can decrease when a pending excursion is later discarded as noise.
   */
  get finalElevationGain(): number {
    return this.totalElevationGainSmoothed + Math.max(this.pendingDelta, 0);
  }

  /** Smoothed elevation loss including a still-pending excursion. See {@link finalElevationGain}. */
  get finalElevationLoss(): number {
    return this.totalElevationLossSmoothed + Math.max(-this.pendingDelta, 0);
  }

  /**
   * Move the pending excursion into the published totals. Only ever adds, so
   * the monotonicity invariant on those fields is preserved.
   */
  private publishPending() {
    if (this.pendingDelta > 0) {
      this.totalElevationGainSmoothed += this.pendingDelta;
    } else if (this.pendingDelta < 0) {
      this.totalElevationLossSmoothed -= this.pendingDelta;
    }
    this.pendingDelta = 0;
    this.pendingAnchorZ = null;
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
      // Push once per call, including this first call, so
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
    // Unconditional push keeps cumulativeDistance's entry count equal
    // to the call count even when a hostile coordinate yields a non-finite
    // haversine distance.
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

    // Defer-then-publish noise filter. An above-threshold elevation move is
    // held in a single pending slot rather than added to the published
    // totals straight away. It is *discarded* if the track then returns to
    // the pre-excursion elevation WITHOUT having moved horizontally — the
    // signature of altimeter/GPS noise on a paused or stationary device, and
    // the only thing that distinguishes it from rolling terrain (which
    // produces an identical elevation series). Any other subsequent move
    // confirms it, so it is published then.
    //
    // Deferring rather than committing-then-retracting is what keeps
    // totalElevationGainSmoothed/LossSmoothed monotonic (see the INVARIANT
    // note on those fields): a discarded excursion was never published, so
    // no published total ever decreases. Horizontal stillness is checked
    // only on the discard path, so a monotonic low-horizontal climb is
    // never affected.
    //
    // The pending excursion is not lost when a track ends mid-swing: it is
    // surfaced by the finalElevationGain/finalElevationLoss getters, which
    // are what a completed track's reported totals come from.
    if (elevation !== undefined) {
      if (this.lastFilteredZ === null) {
        // This point establishes the smoothed anchor; no diff to record yet.
        this.lastFilteredZ = elevation;
        this.lastFilteredZPointXY = point;
        this.pendingDelta = 0;
        this.pendingAnchorZ = null;
      } else {
        const elevationDiffSmoothed = elevation - this.lastFilteredZ;

        if (Math.abs(elevationDiffSmoothed) < this.thresholdZ_m) {
          // Below the noise floor — nothing to defer, publish or discard.
        } else {
          // Distance from where the pending excursion left off to here: "has
          // the device moved horizontally while the elevation came back?"
          const returnDistance =
            this.lastFilteredZPointXY !== null
              ? haversineDistance(
                  this.lastFilteredZPointXY.$.lat,
                  this.lastFilteredZPointXY.$.lon,
                  point.$.lat,
                  point.$.lon
                )
              : Infinity;

          const cancelsPending =
            this.pendingDelta !== 0 &&
            this.pendingAnchorZ !== null &&
            Math.sign(elevationDiffSmoothed) === -Math.sign(this.pendingDelta) &&
            Math.abs(elevation - this.pendingAnchorZ) < this.thresholdZ_m &&
            Number.isFinite(returnDistance) &&
            returnDistance < this.thresholdXY_m;

          if (cancelsPending) {
            // Noise round-trip: drop the pending excursion unpublished and
            // rewind the anchor to where the excursion started.
            this.lastFilteredZ = this.pendingAnchorZ;
            this.lastFilteredZPointXY = point;
            this.pendingDelta = 0;
            this.pendingAnchorZ = null;
          } else {
            // This move confirms whatever was pending — publish it — and
            // then becomes the new pending excursion itself.
            this.publishPending();
            this.pendingAnchorZ = this.lastFilteredZ;
            this.pendingDelta = elevationDiffSmoothed;
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
