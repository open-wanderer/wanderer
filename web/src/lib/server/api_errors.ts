import { json, type Handle } from "@sveltejs/kit";

const encoder = new TextEncoder();

export function isApiRequest(url: URL) {
  return url.pathname.startsWith("/api/");
}

/**
 * Makes error responses under /api/ parseable for API clients.
 *
 * - No route matched: 404 JSON, before the CSRF and auth hooks run.
 * - 405 from an existing route: SvelteKit's text/plain body is replaced with
 *   JSON, keeping every header of the original response (Allow, the cookies
 *   the auth hook refreshed, anything set via event.setHeaders()).
 */
export const apiErrorsAsJson: Handle = async ({ event, resolve }) => {
  if (!isApiRequest(event.url)) {
    return resolve(event);
  }

  if (event.route.id === null) {
    return json({ message: "not_found" }, { status: 404 });
  }

  const response = await resolve(event);
  if (response.status === 405) {
    return withJsonBody(response, { message: "method_not_allowed" });
  }
  return response;
}

// Rebuilds a response around a JSON body while preserving its status and
// headers. The Response body is immutable, so a new one has to be constructed.
function withJsonBody(response: Response, body: unknown) {
  const text = JSON.stringify(body);
  const headers = new Headers(response.headers);
  headers.set("content-type", "application/json");
  headers.set("content-length", encoder.encode(text).byteLength.toString());
  return new Response(text, { status: response.status, statusText: response.statusText, headers });
}
