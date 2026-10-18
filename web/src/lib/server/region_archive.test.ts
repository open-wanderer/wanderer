import type { RequestEvent } from '@sveltejs/kit';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { proxyRegionArchive } from './region_archive';

function buildEvent(upstream: (url: string, init: RequestInit) => Promise<Response>, headers: Record<string, string> = {}) {
  return {
    request: new Request('http://localhost/api/v1/regions/monaco/download', { headers }),
    locals: { pb: { baseURL: 'http://internal', authStore: { token: 't' } } },
    fetch: vi.fn(upstream),
  } as unknown as RequestEvent;
}

describe('proxyRegionArchive', () => {
  afterEach(() => vi.useRealTimers());

  it('forwards HEAD as HEAD so no archive is transferred', async () => {
    const event = buildEvent(async () => new Response(null, {
      status: 200,
      headers: { 'Content-Length': '1536520', ETag: '"177208-1"', 'Content-Type': 'application/octet-stream' },
    }));

    const response = await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles', 'HEAD');

    const fetchMock = event.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][1].method).toBe('HEAD');
    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Length')).toBe('1536520');
    expect(response.headers.get('ETag')).toBe('"177208-1"');
  });

  it('keeps the boundary of a multi-range response', async () => {
    const contentType = 'multipart/byteranges; boundary=abc123';
    const event = buildEvent(async () => new Response('--abc123...', { status: 206, headers: { 'Content-Type': contentType } }),
      { Range: 'bytes=0-9,100-109' });

    const response = await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');

    expect(response.status).toBe(206);
    expect(response.headers.get('Content-Type')).toBe(contentType);
  });

  it('labels single-part archives and tells caches to revalidate', async () => {
    const event = buildEvent(async () => new Response('pmtiles', { status: 200, headers: { 'Content-Type': 'text/plain; charset=utf-8' } }));

    const response = await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');

    expect(response.headers.get('Content-Type')).toBe('application/octet-stream');
    expect(response.headers.get('Cache-Control')).toBe('private, no-cache');
    expect(response.headers.get('Content-Disposition')).toBe('attachment; filename="monaco-vector.pmtiles"');
  });

  it('passes errors through with their content type', async () => {
    const event = buildEvent(async () => new Response(JSON.stringify({ message: 'Region archive not ready yet' }), {
      status: 404,
      headers: { 'Content-Type': 'application/json' },
    }));

    const response = await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');

    expect(response.status).toBe(404);
    expect(response.headers.get('Content-Type')).toBe('application/json');
    expect(response.headers.get('Content-Disposition')).toBeNull();
  });

  it('answers 504 when the backend sends no headers', async () => {
    vi.useFakeTimers();
    const event = buildEvent((_url, init) => new Promise((_resolve, reject) => {
      init.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')));
    }));

    const pending = proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');
    await vi.advanceTimersByTimeAsync(30_000);
    const response = await pending;

    expect(response.status).toBe(504);
  });
});
