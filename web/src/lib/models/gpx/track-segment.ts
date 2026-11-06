import type Track from './track';
import Waypoint, { hasRouteAnchor, hasUsablePosition } from './waypoint';

type FileTimes = { durationMs: number, start?: Date, end?: Date };

/**
 * The summed time of a segment's legs, as the app measures it: a marked anchor
 * opens a new leg, so the hop into it belongs to neither leg.
 */
function legsDurationMs(points: Waypoint[]): number {
  let total = 0;
  let legStart = 0;
  const addLeg = (legEnd: number) => {
    const start = points[legStart].time;
    const end = points[legEnd].time;
    if (legEnd > legStart && start && end) {
      total += end.getTime() - start.getTime();
    }
  };

  for (let i = 1; i < points.length - 1; i++) {
    if (hasRouteAnchor(points[i])) {
      addLeg(i - 1);
      legStart = i;
    }
  }
  if (points.length > 0) {
    addLeg(points.length - 1);
  }
  return total;
}

/** A function, as the route store's segments are not always TrackSegment instances. */
export function segmentDurationMs(segment: Pick<TrackSegment, 'trkpt' | 'fileTimes'>): number {
  return segment.fileTimes?.durationMs ?? legsDurationMs(segment.trkpt ?? []);
}

export default class TrackSegment {
  trkpt?: Waypoint[];
  extensions?: string;
  /**
   * The times of the segment as the file has them, when a point without a
   * position was skipped. The app still reads the time of such a point.
   * Not enumerable, so that it is not written to GPX, and gone once the segment
   * is copied for an edit, which is when it stops describing the file.
   */
  declare fileTimes?: FileTimes;
  constructor(object: { trkpt?: Waypoint[], extensions?: string }) {
    if (object.trkpt) {
      if (!Array.isArray(object.trkpt)) {
        object.trkpt = [object.trkpt];
      }
      // Points without a position carry no geometry; the app's reader skips
      // them too, so both compute the same metrics for such a file.
      const points = object.trkpt.map(trkpt => new Waypoint(trkpt));
      this.trkpt = points.filter(hasUsablePosition);
      if (this.trkpt.length !== points.length) {
        const fileTimes: FileTimes = { durationMs: legsDurationMs(points), start: points[0].time, end: points[points.length - 1].time };
        Object.defineProperty(this, 'fileTimes', { value: fileTimes });
      }
    }
    this.extensions = object.extensions;
  }

  toGeoJSON(
    track: Track,
    segmentId: number,
    featureId: number
  ): GeoJSON.Feature {
    const coordinates = (this.trkpt || []).map(pt => [
      pt.$.lon ?? 0,
      pt.$.lat ?? 0,
      pt.ele ?? 0,
    ]);

    const times = (this.trkpt || []).map(pt => pt.time?.toISOString() ?? null);

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
        coordinateProperties: {
          times
        }
      }
    };
  }

}