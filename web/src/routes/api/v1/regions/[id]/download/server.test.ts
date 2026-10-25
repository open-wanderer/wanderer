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

  it('forwards If-Range and returns the archive ETag and Last-Modified', async () => {
    const event = buildEvent('bytes=100-');
    event.request.headers.set('If-Range', '"c8-1873"');
    event.fetch = vi.fn(
      async () =>
        new Response('whole new build', {
          status: 200,
          headers: {
            'Content-Length': '15',
            ETag: '"d2-1875"',
            'Last-Modified': 'Thu, 08 Oct 2026 03:00:00 GMT',
          },
        })
    );

    const response = await GET(event);

    const fetchMock = event.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][1].headers['If-Range']).toBe('"c8-1873"');
    expect(response.status).toBe(200);
    expect(response.headers.get('ETag')).toBe('"d2-1875"');
    expect(response.headers.get('Last-Modified')).toBe('Thu, 08 Oct 2026 03:00:00 GMT');
  });

  it('passes a 304 Not Modified through with its ETag', async () => {
    const event = buildEvent();
    event.request.headers.set('If-None-Match', '"c8-1873"');
    event.fetch = vi.fn(async () => new Response(null, { status: 304, headers: { ETag: '"c8-1873"' } }));

    const response = await GET(event);

    const fetchMock = event.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][1].headers['If-None-Match']).toBe('"c8-1873"');
    expect(response.status).toBe(304);
    expect(response.headers.get('ETag')).toBe('"c8-1873"');
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
