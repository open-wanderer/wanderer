import { describe, expect, it } from 'vitest';
import { decodePolyline, encodePolyline } from '#lib/util/polyline_util.ts';
import {
  stitchNavigateTrips,
  traceRouteTrips,
  type ValhallaTrip,
} from './valhalla_trace.ts';

type Coord = [number, number];

function line(count: number, start: number): Coord[] {
  return Array.from({ length: count }, (_, i) => [47 + (start + i) * 0.001, 11 + (start + i) * 0.0005] as Coord);
}

type Leg = ValhallaTrip['legs'][number];

function leg(coords: Coord[], maneuvers: NonNullable<Leg['maneuvers']>): Leg {
  return { shape: encodePolyline(coords), maneuvers };
}

function maneuver(
  type: number | undefined,
  begin: number,
  extra: { bearing_before?: number; bearing_after?: number } = {}
) {
  return { type, instruction: `m${type}`, length: 1, begin_shape_index: begin, ...extra };
}

describe('traceRouteTrips', () => {
  const trip = (n: number): ValhallaTrip => ({ legs: [leg(line(2, n), [])] });

  it('returns only the trip when there are no alternates', () => {
    const t = trip(0);
    expect(traceRouteTrips({ trip: t })).toEqual([t]);
  });

  it('returns the trip followed by the alternates in order', () => {
    const [a, b, c] = [trip(0), trip(1), trip(2)];
    expect(traceRouteTrips({ trip: a, alternates: [{ trip: b }, { trip: c }] })).toEqual([a, b, c]);
  });

  it('skips alternates without a legs array', () => {
    const [a, b] = [trip(0), trip(1)];
    const data = {
      trip: a,
      alternates: [{}, { trip: {} }, { trip: { legs: 'x' } }, null, { trip: b }],
    };
    expect(traceRouteTrips(data as never)).toEqual([a, b]);
  });

  it('ignores a non-array alternates', () => {
    const a = trip(0);
    expect(traceRouteTrips({ trip: a, alternates: { trip: trip(1) } } as never)).toEqual([a]);
  });

  it('returns nothing when trip.legs is missing, even with alternates', () => {
    expect(traceRouteTrips({ alternates: [{ trip: trip(1) }] })).toEqual([]);
    expect(traceRouteTrips({ trip: {}, alternates: [{ trip: trip(1) }] } as never)).toEqual([]);
  });

  it('returns nothing for null and undefined', () => {
    expect(traceRouteTrips(null)).toEqual([]);
    expect(traceRouteTrips(undefined)).toEqual([]);
  });
});

describe('stitchNavigateTrips', () => {
  it('keeps a single piece exactly as the handler used to map it', () => {
    const coords = line(3, 0);
    const result = stitchNavigateTrips([
      {
        legs: [
          leg(coords, [
            maneuver(1, 0, { bearing_before: 10, bearing_after: 20 }),
            maneuver(15, 1, { bearing_before: 30 }),
            maneuver(4, 2),
          ]),
        ],
      },
    ]);

    expect(result.shape).toEqual(coords);
    expect(result.maneuvers.map((m) => m.begin_shape_index)).toEqual([0, 1, 2]);
    expect(result.maneuvers.map((m) => m.bearing)).toEqual([20, 30, 0]);
    expect(result.maneuvers.map((m) => m.type)).toEqual([1, 15, 4]);
    expect(result.maneuvers[0]).toEqual({
      instruction: 'm1',
      length: 1,
      begin_shape_index: 0,
      bearing: 20,
      type: 1,
    });
  });

  it('falls back to type 0 when a maneuver has no type', () => {
    const result = stitchNavigateTrips([{ legs: [leg(line(2, 0), [maneuver(undefined, 0)])] }]);
    expect(result.maneuvers[0].type).toBe(0);
  });

  it('concatenates pieces in trace order and keeps only the final destination', () => {
    const pieces = [line(3, 0), line(2, 10), line(2, 20)];
    const trips: ValhallaTrip[] = [
      { legs: [leg(pieces[0], [maneuver(1, 0), maneuver(4, 2)])] },
      { legs: [leg(pieces[1], [maneuver(2, 0), maneuver(5, 1)])] },
      { legs: [leg(pieces[2], [maneuver(3, 0), maneuver(6, 1)])] },
    ];

    const result = stitchNavigateTrips(trips);

    expect(result.shape).toEqual(pieces.flat());
    expect(result.shape).toHaveLength(7);
    expect(result.maneuvers.map((m) => m.type)).toEqual([1, 2, 3, 6]);
    expect(result.maneuvers.map((m) => m.begin_shape_index)).toEqual([0, 3, 5, 6]);
  });

  it('offsets the second leg of a piece by the first leg point count', () => {
    const first = line(3, 0);
    const second = line(2, 10);
    const result = stitchNavigateTrips([
      {
        legs: [
          leg(first, [maneuver(1, 0), maneuver(15, 2)]),
          leg(second, [maneuver(15, 0), maneuver(4, 1)]),
        ],
      },
    ]);

    expect(result.shape).toEqual([...first, ...second]);
    expect(result.maneuvers.map((m) => m.begin_shape_index)).toEqual([0, 2, 3, 4]);
  });

  it('decodes the shapes it was given', () => {
    const coords = line(2, 0);
    expect(decodePolyline(encodePolyline(coords)).map(([lng, lat]) => [lat, lng])).toEqual(coords);
  });
});
