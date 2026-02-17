import { haversineDistance } from './utils';

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
      // Initialize raw and smoothed anchors with the first point.
      this.lastPointXY = point;
      this.lastFilteredPointXY = point;
      this.lastFilteredZ = point.ele ?? 0;
      this.lastZ = point.ele ?? 0;
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

    const elevation = point.ele ?? 0;
    // @ts-ignore I know this.lastZ is not null
    const elevationDiff = elevation - this.lastZ;
    this.lastZ = elevation;
    if (elevationDiff > 0) {
      this.totalElevationGain += elevationDiff;
    }
    if (elevationDiff < 0) {
      this.totalElevationLoss -= elevationDiff;
    }

    if (smoothedDistance < this.thresholdXY_m) {
      return;
    }

    this.totalDistanceSmoothed += smoothedDistance;
    this.lastFilteredPointXY = point;

    // @ts-ignore: I know this.lastFilteredZ is not null
    const elevationDiffSmoothed = elevation - this.lastFilteredZ;

    if (Math.abs(elevationDiffSmoothed) < this.thresholdZ_m) {
      return;
    }

    this.lastFilteredZ = elevation;
    if (elevationDiffSmoothed > 0) {
      this.totalElevationGainSmoothed += elevationDiffSmoothed;
    } else {
      this.totalElevationLossSmoothed -= elevationDiffSmoothed;
    }
  }
}

export default GpxMetricsComputation;
