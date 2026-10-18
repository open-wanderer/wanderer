import type { RequestEvent } from '@sveltejs/kit';
import { describe, expect, it, vi } from 'vitest';
import { GET } from './+server';

function buildEvent(range?: string, id = 'de-nrw'): RequestEvent {
  const url = `http://localhost/api/v1/regions/${id}/download`;
  const request = new Request(url, {
    headers: range ? { Range: range } : undefined,
  });

  return {
    request,
    params: { id },
    locals: {
      pb: {
        baseURL: 'http://internal',
        authStore: { token: 't' },
      },
    },
    fetch: vi.fn(),
  } as unknown as RequestEvent;
}

describe('GET /api/v1/regions/[id]/download', () => {
  it('forwards a Range header and returns the upstream 206 + Content-Range', async () => {
    const event = buildEvent('bytes=100-');
    event.fetch = vi.fn(
      async () =>
        new Response('partial', {
          status: 206,
          headers: {
            'Content-Range': 'bytes 100-199/200',
            'Accept-Ranges': 'bytes',
            'Content-Length': '100',
          },
        })
    );

    const response = await GET(event);

    expect(response.status).toBe(206);
    expect(response.headers.get('Content-Range')).toBe('bytes 100-199/200');
    const fetchMock = event.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][1].headers.Range).toBe('bytes=100-');
  });

  it('returns 200 for a fresh request with no Range header', async () => {
    const event = buildEvent();
    event.fetch = vi.fn(
      async () =>
        new Response('full', {
          status: 200,
          headers: { 'Content-Length': '200' },
        })
    );

    const response = await GET(event);

    expect(response.status).toBe(200);
  });

  it('accepts a seeded materialized-path id containing a dot', async () => {
    const event = buildEvent(undefined, 'algeria.algeria_central');
    event.fetch = vi.fn(
      async () =>
        new Response('full', {
          status: 200,
          headers: { 'Content-Length': '200' },
        })
    );

    const response = await GET(event);

    expect(response.status).toBe(200);
    const fetchMock = event.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][0]).toContain('algeria.algeria_central');
  });

  it('rejects a region id containing a path-traversal sequence with a 400', async () => {
    const event = buildEvent(undefined, 'algeria..central');
    event.fetch = vi.fn();

    const response = await GET(event);

    expect(response.status).toBe(400);
    expect(event.fetch).not.toHaveBeenCalled();
  });
});
