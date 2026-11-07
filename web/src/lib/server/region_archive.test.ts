import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';
import type { AddressInfo } from 'node:net';
import type { RequestEvent } from '@sveltejs/kit';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { proxyRegionArchive } from './region_archive';

// Tests swap request-scoped functions on the event.
type TestEvent = { -readonly [K in keyof RequestEvent]: RequestEvent[K] };

function buildEvent(upstream: (url: string, init: RequestInit) => Promise<Response>, headers: Record<string, string> = {}) {
  return {
    request: new Request('http://localhost/api/v1/regions/monaco/download', { headers }),
    locals: { pb: { baseURL: 'http://internal', authStore: { token: 't' } } },
    fetch: vi.fn(upstream),
    getClientAddress: () => '192.0.2.1',
  } as unknown as TestEvent;
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

  it('answers 504 even if the abort never reaches the fetch', async () => {
    vi.useFakeTimers();
    const event = buildEvent(() => new Promise(() => {}));

    const pending = proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');
    await vi.advanceTimersByTimeAsync(30_000);

    expect((await pending).status).toBe(504);
  });

  it('forwards the resolved client address, not a client-supplied one', async () => {
    const event = buildEvent(async () => new Response('pmtiles'), { 'X-Forwarded-For': '198.51.100.1' });

    await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');

    const fetchMock = event.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][1].headers['X-Forwarded-For']).toBe('192.0.2.1');
  });

  it('sends no client address when none can be resolved', async () => {
    const event = buildEvent(async () => new Response('pmtiles'), { 'X-Forwarded-For': '198.51.100.1' });
    event.getClientAddress = () => {
      throw new Error('ADDRESS_HEADER was specified, but the incoming request does not include it');
    };

    await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');

    const fetchMock = event.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][1].headers).not.toHaveProperty('X-Forwarded-For');
  });

  it('answers a JSON 503 without the raw error when the backend cannot be reached', async () => {
    const event = buildEvent(async () => {
      throw new TypeError('fetch failed', { cause: new Error('connect ECONNREFUSED 10.0.0.1:8090') });
    });

    const response = await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');

    expect(response.status).toBe(503);
    expect(response.headers.get('Content-Type')).toContain('application/json');
    expect(await response.json()).toEqual({ message: 'backend_unavailable' });
  });
});

describe('proxyRegionArchive streaming from a real backend', () => {
  let server: Server | undefined;

  afterEach(async () => {
    server?.closeAllConnections();
    await new Promise(resolve => server ? server.close(resolve) : resolve(undefined));
    server = undefined;
  });

  async function backend(handler: (req: IncomingMessage, res: ServerResponse) => void) {
    server = createServer(handler);
    await new Promise<void>(resolve => server!.listen(0, '127.0.0.1', resolve));
    const baseURL = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
    return {
      request: new Request('http://localhost/api/v1/regions/monaco/download'),
      locals: { pb: { baseURL, authStore: { token: 't' } } },
      fetch,
      getClientAddress: () => '192.0.2.1',
    } as unknown as RequestEvent;
  }

  it('cancels the upstream transfer when the client disconnects', async () => {
    const upstreamClosed = new Promise<boolean>(resolve => {
      void backend((_req, res) => {
        res.writeHead(200, { 'Content-Length': String(1024 * 1024 * 1024) });
        res.write('first chunk');
        res.on('close', () => resolve(res.writableFinished));
      }).then(event => proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles'))
        .then(async response => {
          const reader = response.body!.getReader();
          await reader.read();
          // adapter-node cancels the body like this when the client goes away.
          await reader.cancel();
        });
    });

    expect(await upstreamClosed).toBe(false);
  });

  it('fails the client download when the backend dies mid-body', async () => {
    const event = await backend((_req, res) => {
      res.writeHead(200, { 'Content-Length': '1000' });
      res.write('partial', () => res.destroy());
    });

    const response = await proxyRegionArchive(event, '/regions/monaco/download', 'monaco-vector.pmtiles');

    expect(response.status).toBe(200);
    expect(response.headers.get('Content-Length')).toBe('1000');
    // A clean end here would let the client keep a truncated archive.
    await expect(response.arrayBuffer()).rejects.toThrow();
  });
});
