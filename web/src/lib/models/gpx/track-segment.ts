import { extractPointMetrics, POINT_METRIC_KEYS, type PointMetricKey } from './extensions';
import type Track from './track';
import Waypoint from './waypoint';

export default class TrackSegment {
  trkpt?: Waypoint[];
  extensions?: string;
  constructor(object: { trkpt?: Waypoint[], extensions?: string }) {
    if (object.trkpt) {
      if (!Array.isArray(object.trkpt)) {
        object.trkpt = [object.trkpt];
      }
      this.trkpt = object.trkpt.map(trkpt => new Waypoint(trkpt))
    }
    this.extensions = object.extensions;
  }

  toGeoJSON(
    track: Track,
    segmentId: number,
    featureId: number
  ): GeoJSON.Feature {
    const points = this.trkpt || [];

    const coordinates = points.map(pt => [
      pt.$.lon ?? 0,
      pt.$.lat ?? 0,
      pt.ele ?? 0,
    ]);

    const times = points.map(pt => pt.time?.toISOString() ?? null);

    const coordinateProperties: Record<string, (number | string | null)[]> = { times };

    // Sensor metrics (HR/cadence/power/temperature) live in each point's
    // <extensions>. Build one index-aligned series per metric and attach it only
    // if at least one point actually carries that metric.
    const metricSeries: Record<PointMetricKey, (number | null)[]> = {
      heartRate: [],
      cadence: [],
      power: [],
      temperature: [],
    };
    const metricPresent: Record<PointMetricKey, boolean> = {
      heartRate: false,
      cadence: false,
      power: false,
      temperature: false,
    };

    for (const pt of points) {
      const metrics = extractPointMetrics(pt.extensions);
      for (const key of POINT_METRIC_KEYS) {
        const value = metrics[key];
        if (value !== undefined) {
          metricPresent[key] = true;
        }
        metricSeries[key].push(value ?? null);
      }
    }

    for (const key of POINT_METRIC_KEYS) {
      if (metricPresent[key]) {
        coordinateProperties[key] = metricSeries[key];
      }
    }

    return {
      type: "Feature",
      geometry: {
        type: "LineString",
        coordinates,
      },
      properties: {
        name: track.name,
        desc: track.desc,
        type: track.type,
        number: track.number,
        featureId,
        segmentId,
        coordinateProperties
      }
    };
  }

}