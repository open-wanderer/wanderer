import type { RequestEvent } from '@sveltejs/kit';
import { ClientResponseError } from 'pocketbase';
import { describe, expect, it, vi } from 'vitest';
import { GET } from './+server';

function buildEvent(
  id = 'canada.alberta.south',
  record?: { path: string; polygon: object; bbox: number[] },
  getFirstListItemImpl?: (filter: string) => Promise<unknown>
): RequestEvent {
  const filterMock = vi.fn(
    (expr: string, params: Record<string, unknown>) =>
      `${expr}::${JSON.stringify(params)}`
  );

  const getFirstListItem =
    getFirstListItemImpl ??
    vi.fn(async () => {
      if (!record) {
        throw new ClientResponseError({ status: 404, response: {} });
      }
      return record;
    });

  const collectionMock = vi.fn(() => ({ getFirstListItem }));

  return {
    params: { id },
    locals: {
      pb: {
        filter: filterMock,
        collection: collectionMock,
      },
    },
    fetch: vi.fn(),
  } as unknown as RequestEvent;
}

describe('GET /api/v1/regions/[id]/geometry', () => {
  it('returns 200 with { path, polygon, bbox } from the cached row', async () => {
    const record = {
      path: 'canada.alberta.south',
      polygon: { type: 'Polygon', coordinates: [] },
      bbox: [-115, 50, -114, 51],
    };
    const event = buildEvent('canada.alberta.south', record);

    const response = await GET(event);

    expect(response.status).toBe(200);
    const body = await response.json();
    expect(body).toEqual(record);
  });

  it('binds the path via pb.filter, never string concatenation', async () => {
    const record = {
      path: 'canada.alberta.south',
      polygon: { type: 'Polygon', coordinates: [] },
      bbox: [-115, 50, -114, 51],
    };
    const event = buildEvent('canada.alberta.south', record);

    await GET(event);

    const filterMock = event.locals.pb.filter as unknown as ReturnType<
      typeof vi.fn
    >;
    expect(filterMock).toHaveBeenCalledWith('path = {:path}', {
      path: 'canada.alberta.south',
    });
  });

  it('rejects an id containing a path-traversal sequence with a 400', async () => {
    const event = buildEvent('canada..south');

    const response = await GET(event);

    expect(response.status).toBe(400);
    expect(event.locals.pb.collection).not.toHaveBeenCalled();
  });

  it('rejects an id containing an uppercase/slash character with a 400', async () => {
    const event = buildEvent('Canada/Alberta');

    const response = await GET(event);

    expect(response.status).toBe(400);
    expect(event.locals.pb.collection).not.toHaveBeenCalled();
  });

  it('returns 404 when no cached row exists', async () => {
    const event = buildEvent('canada.alberta.south', undefined);

    const response = await GET(event);

    expect(response.status).toBe(404);
  });

  it('never proxies upstream — event.fetch is never called in any case', async () => {
    const happyEvent = buildEvent('canada.alberta.south', {
      path: 'canada.alberta.south',
      polygon: { type: 'Polygon', coordinates: [] },
      bbox: [-115, 50, -114, 51],
    });
    const missingEvent = buildEvent('canada.alberta.south', undefined);
    const badEvent = buildEvent('canada..south');

    await GET(happyEvent);
    await GET(missingEvent);
    await GET(badEvent);

    expect(happyEvent.fetch).not.toHaveBeenCalled();
    expect(missingEvent.fetch).not.toHaveBeenCalled();
    expect(badEvent.fetch).not.toHaveBeenCalled();
  });
});
