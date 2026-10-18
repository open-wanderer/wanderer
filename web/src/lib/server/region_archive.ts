import type { RequestEvent } from '@sveltejs/kit';

// Archives are replaced by an atomic rename on every rebuild. The conditional
// headers let the backend answer a resumed Range request with the whole new
// file instead of splicing bytes of two different builds together.
const forwardedRequestHeaders = ['Range', 'If-Range', 'If-None-Match', 'If-Modified-Since'];
const forwardedResponseHeaders = ['Content-Length', 'Content-Range', 'Accept-Ranges', 'ETag', 'Last-Modified'];

/**
 * Streams a region archive from the internal Go backend, forwarding the
 * caller's auth token and the headers resumable downloads depend on.
 */
export async function proxyRegionArchive(
  event: RequestEvent,
  backendPath: string,
  filename: string
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

  const response = await event.fetch(`${event.locals.pb.baseURL}${backendPath}`, { headers });

  const responseHeaders: Record<string, string> = {};
  for (const name of forwardedResponseHeaders) {
    const value = response.headers.get(name);
    if (value) {
      responseHeaders[name] = value;
    }
  }

  if (!response.ok) {
    return new Response(response.body, { status: response.status, headers: responseHeaders });
  }

  return new Response(response.body, {
    status: response.status,
    headers: {
      ...responseHeaders,
      'Content-Type': 'application/octet-stream',
      'Content-Disposition': `attachment; filename="${filename}"`,
    },
  });
}
