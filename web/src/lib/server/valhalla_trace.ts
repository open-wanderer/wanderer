import type { NavigateManeuver, NavigateResponse } from '#lib/models/api/valhalla_navigate_schema.ts';
import { decodePolyline } from '#lib/util/polyline_util.ts';

type ValhallaManeuver = {
  type?: number;
  instruction: string;
  length: number;
  begin_shape_index: number;
  bearing_before?: number;
  bearing_after?: number;
};

type ValhallaLeg = {
  shape: string;
  maneuvers?: ValhallaManeuver[];
};

export type ValhallaTrip = {
  legs: ValhallaLeg[];
};

type ValhallaTraceResponse = {
  trip?: ValhallaTrip;
  alternates?: { trip?: ValhallaTrip }[];
};

// Valhalla maneuver types kDestination, kDestinationRight and kDestinationLeft.
export const DESTINATION_MANEUVER_TYPES: ReadonlySet<number> = new Set([4, 5, 6]);

/**
 * Collects every matched piece of a trace_route answer. When the map match has a
 * discontinuity, Valhalla returns the first continuous piece in `trip` and every later
 * piece, in trace order, in `alternates`. No `topk` is sent, so the alternates are
 * pieces, not alternative matchings. Reading only `trip` made navigation arrive at the
 * end of the first piece (issue #1534).
 */
export function traceRouteTrips(data: ValhallaTraceResponse | null | undefined): ValhallaTrip[] {
  const trip = data?.trip;
  if (!trip || !Array.isArray(trip.legs)) {
    return [];
  }

  const trips: ValhallaTrip[] = [trip];
  if (Array.isArray(data?.alternates)) {
    for (const alternate of data.alternates) {
      const piece = alternate?.trip;
      if (piece && Array.isArray(piece.legs)) {
        trips.push(piece);
      }
    }
  }
  return trips;
}

/**
 * Concatenates the shape and maneuvers of every piece in trace order. Only the last
 * maneuver of the whole list may be a destination, so the app's last maneuver is the
 * real end of the trail. Start maneuvers of later pieces are kept; a gap between pieces
 * stays one straight segment of the shape.
 */
export function stitchNavigateTrips(trips: ValhallaTrip[]): NavigateResponse {
  const shape: [number, number][] = [];
  const maneuvers: NavigateManeuver[] = [];

  for (const trip of trips) {
    for (const leg of trip.legs) {
      const offset = shape.length;

      for (const [lng, lat] of decodePolyline(leg.shape)) {
        shape.push([lat, lng]);
      }

      for (const m of leg.maneuvers ?? []) {
        maneuvers.push({
          instruction: m.instruction,
          length: m.length,
          begin_shape_index: offset + m.begin_shape_index,
          bearing: m.bearing_after ?? m.bearing_before ?? 0,
          type: m.type ?? 0,
        });
      }
    }
  }

  const last = maneuvers.length - 1;
  return {
    maneuvers: maneuvers.filter((m, i) => i === last || !DESTINATION_MANEUVER_TYPES.has(m.type)),
    shape,
  };
}

/**
 * Valhalla's default pedestrian max_hiking_difficulty is 1, so trail edges with sac_scale
 * 2 and up get no match candidates and the trace splits into pieces. 6 matches the web
 * route editor's default.
 */
export function traceCostingOptions(
  costing: "pedestrian" | "bicycle"
): { pedestrian: { max_hiking_difficulty: number } } | undefined {
  return costing === "pedestrian" ? { pedestrian: { max_hiking_difficulty: 6 } } : undefined;
}
