import type { RequestEvent } from '@sveltejs/kit';
import { describe, expect, it, vi } from 'vitest';
import { encodePolyline } from '#lib/util/polyline_util.ts';

vi.mock('#lib/server/valhalla.ts', () => ({ getValhallaUrl: () => 'http://valhalla.test' }));

import { POST as navigate } from './navigate/+server';
import { POST as traceRoute } from './trace-route/+server';

type Coord = [number, number];

function line(count: number, start: number): Coord[] {
  return Array.from({ length: count }, (_, i) => [47 + (start + i) * 0.001, 11 + (start + i) * 0.0005] as Coord);
}

function maneuver(type: number, begin: number) {
  return { type, instruction: `m${type}`, length: 1, begin_shape_index: begin };
}

function trip(coords: Coord[], maneuvers: ReturnType<typeof maneuver>[]) {
  return { legs: [{ shape: encodePolyline(coords), maneuvers }] };
}

function event(body: unknown, valhallaData: unknown): RequestEvent {
  return {
    request: new Request('http://localhost/api/v1/valhalla', { method: 'POST', body: JSON.stringify(body) }),
    locals: { pb: { authStore: { record: { id: 'user' }, token: 't' } }, user: { id: 'user' } },
    fetch: vi.fn(async () =>
      new Response(JSON.stringify(valhallaData), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    ),
  } as unknown as RequestEvent;
}

const requestBody = {
  shape: [
    { lat: 47, lon: 11 },
    { lat: 47.01, lon: 11.01 },
  ],
  costing: 'pedestrian',
};

describe('navigate', () => {
  it('stitches the trip and its alternates into one response', async () => {
    const first = line(3, 0);
    const second = line(2, 10);
    const e = event(requestBody, {
      trip: trip(first, [maneuver(1, 0), maneuver(4, 2)]),
      alternates: [{ trip: trip(second, [maneuver(2, 0), maneuver(4, 1)]) }],
    });

    const response = await navigate(e);

    expect(response.status).toBe(200);
    const data = await response.json();
    expect(data.shape).toEqual([...first, ...second]);
    expect(data.maneuvers.map((m: { type: number }) => m.type)).toEqual([1, 2, 4]);
    expect(data.maneuvers.map((m: { begin_shape_index: number }) => m.begin_shape_index)).toEqual([0, 3, 4]);
    expect((e.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe('http://valhalla.test/trace_route');
  });

  it('answers 502 when Valhalla returned only alternates', async () => {
    const e = event(requestBody, { alternates: [{ trip: trip(line(2, 0), [maneuver(1, 0), maneuver(4, 1)]) }] });

    const response = await navigate(e);

    expect(response.status).toBe(502);
    expect(await response.json()).toEqual({ message: 'valhalla_error', detail: 'unexpected response shape' });
  });
});

describe('trace-route', () => {
  it('returns only the first matched piece and ignores the alternates', async () => {
    const first = line(3, 0);
    const second = line(2, 10);
    const e = event(requestBody, {
      trip: trip(first, [maneuver(1, 0), maneuver(4, 2)]),
      alternates: [{ trip: trip(second, [maneuver(2, 0), maneuver(4, 1)]) }],
    });

    const response = await traceRoute(e);

    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({
      shape: first.map(([lat, lon]) => ({ lat, lon })),
    });
  });

  it('answers 502 when Valhalla returned only alternates', async () => {
    const e = event(requestBody, { alternates: [{ trip: trip(line(2, 0), [maneuver(1, 0), maneuver(4, 1)]) }] });

    const response = await traceRoute(e);

    expect(response.status).toBe(502);
    expect(await response.json()).toEqual({ message: 'valhalla_error', detail: 'unexpected response shape' });
  });
});

describe('request body costing options', () => {
  const answer = { trip: trip(line(2, 0), [maneuver(1, 0), maneuver(4, 1)]) };
  const hiking = { pedestrian: { max_hiking_difficulty: 6 } };

  function sentBody(e: RequestEvent) {
    return JSON.parse((e.fetch as ReturnType<typeof vi.fn>).mock.calls[0][1].body);
  }

  it('navigate sends max_hiking_difficulty 6 for pedestrian', async () => {
    const e = event(requestBody, answer);
    await navigate(e);
    expect(sentBody(e)).toEqual({
      shape: requestBody.shape,
      costing: 'pedestrian',
      directions_type: 'instructions',
      shape_match: 'map_snap',
      costing_options: hiking,
    });
  });

  it('navigate sends no costing_options for bicycle', async () => {
    const e = event({ ...requestBody, costing: 'bicycle' }, answer);
    await navigate(e);
    expect(sentBody(e)).toEqual({
      shape: requestBody.shape,
      costing: 'bicycle',
      directions_type: 'instructions',
      shape_match: 'map_snap',
    });
  });

  it('navigate defaults to pedestrian costing options when costing is absent', async () => {
    const e = event({ shape: requestBody.shape }, answer);
    await navigate(e);
    expect(sentBody(e).costing_options).toEqual(hiking);
  });

  it('trace-route sends max_hiking_difficulty 6 for pedestrian', async () => {
    const e = event(requestBody, answer);
    await traceRoute(e);
    expect(sentBody(e)).toEqual({
      shape: requestBody.shape,
      costing: 'pedestrian',
      shape_match: 'map_snap',
      costing_options: hiking,
    });
  });

  it('trace-route sends no costing_options for bicycle', async () => {
    const e = event({ ...requestBody, costing: 'bicycle' }, answer);
    await traceRoute(e);
    expect(sentBody(e)).toEqual({
      shape: requestBody.shape,
      costing: 'bicycle',
      shape_match: 'map_snap',
    });
  });
});
