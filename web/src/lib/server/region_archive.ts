import type { RequestEvent } from '@sveltejs/kit';

// Archives are replaced by an atomic rename on every rebuild. The conditional
// headers let the backend answer a resumed Range request with the whole new
// file instead of splicing bytes of two different builds together.
const forwardedRequestHeaders = ['Range', 'If-Range', 'If-None-Match', 'If-Modified-Since'];
const forwardedResponseHeaders = ['Content-Type', 'Content-Length', 'Content-Range', 'Accept-Ranges', 'ETag', 'Last-Modified'];

// How long to wait for the backend's response headers. The body itself can
// take much longer for a large archive, so it is not bounded.
const BACKEND_HEADERS_TIMEOUT_MS = 30_000;

/**
 * Streams a region archive from the internal Go backend, forwarding the
 * caller's auth token and the headers resumable downloads depend on. HEAD is
 * forwarded as HEAD so it never transfers the archive.
 */
export async function proxyRegionArchive(
  event: RequestEvent,
  backendPath: string,
  filename: string,
  method: 'GET' | 'HEAD' = 'GET'
): Promise<Response> {
  const headers: Record<string, string> = {
    Authorization: event.locals.pb.authStore.token ? `Bearer ${event.locals.pb.authStore.token}` : '',
  };
  for (const name of forwardedRequestHeaders) {
    const value = event.request.headers.get(name);
    if (value) {
      headers[name] = value;
    }
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), BACKEND_HEADERS_TIMEOUT_MS);
  let response: Response;
  try {
    response = await event.fetch(`${event.locals.pb.baseURL}${backendPath}`, {
      method,
      headers,
      signal: controller.signal,
    });
  } catch (e) {
    if (controller.signal.aborted) {
      return new Response(JSON.stringify({ message: 'Region archive backend did not respond' }), {
        status: 504,
        headers: { 'Content-Type': 'application/json' },
      });
    }
    throw e;
  } finally {
    clearTimeout(timeout);
  }

  const responseHeaders: Record<string, string> = {};
  for (const name of forwardedResponseHeaders) {
    const value = response.headers.get(name);
    if (value) {
      responseHeaders[name] = value;
    }
  }

  if (!response.ok && response.status !== 304) {
    return new Response(response.body, { status: response.status, headers: responseHeaders });
  }

  // A multi-range 206 is a multipart body whose boundary lives in its
  // Content-Type, so only single-part responses are relabelled.
  if (!responseHeaders['Content-Type']?.startsWith('multipart/')) {
    responseHeaders['Content-Type'] = 'application/octet-stream';
  }

  return new Response(response.body, {
    status: response.status,
    headers: {
      ...responseHeaders,
      'Content-Disposition': `attachment; filename="${filename}"`,
      // Revalidate on every use: the archive behind the URL changes when the
      // region is rebuilt.
      'Cache-Control': 'private, no-cache',
    },
  });
}
