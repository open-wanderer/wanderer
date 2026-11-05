import type { RequestEvent } from '@sveltejs/kit';

function safeJson(text: string): any {
    try {
        return JSON.parse(text);
    } catch {
        return { message: text };
    }
}

/** The address to send PocketBase as X-Forwarded-For, never a client-supplied header. */
export function clientAddress(event: RequestEvent): string | undefined {
    try {
        return event.getClientAddress();
    } catch {
        // ADDRESS_HEADER is set but this request has none (e.g. a healthcheck).
        return undefined;
    }
}

/**
 * Rejects when the signal aborts, even if the fetch never settles. event.fetch
 * wraps the call in Request objects, and undici stops forwarding the abort once
 * a garbage collection reclaims them, so its signal alone is not reliable.
 */
export function settleOnAbort<T>(pending: Promise<T>, signal: AbortSignal): Promise<T> {
    return new Promise((resolve, reject) => {
        const onAbort = () => reject(signal.reason);
        if (signal.aborted) {
            onAbort();
            return;
        }
        signal.addEventListener("abort", onAbort, { once: true });
        pending.then(resolve, reject).finally(() => signal.removeEventListener("abort", onAbort));
    });
}

export async function proxyJsonResponse(response: Response) {
    const text = await response.text();
    const payload = text.length ? safeJson(text) : {};
    if (!response.ok) {
        return Response.json(payload, { status: response.status });
    }
    return Response.json(payload);
}

// Hop-by-hop headers must not be forwarded; Node's fetch rejects some of
// them, e.g. "Connection: upgrade". Content-Length is set by fetch for the
// body it sends; a forwarded copy doubles the value, which undici rejects.
const NOT_FORWARDED = new Set([
    "connection", "keep-alive", "proxy-connection", "te", "trailer", "transfer-encoding", "upgrade", "expect",
    "content-length",
]);

export function forwardableHeaders(headers: Headers): Record<string, string> {
    const result: Record<string, string> = {};
    headers.forEach((value, key) => {
        if (!NOT_FORWARDED.has(key)) {
            result[key] = value;
        }
    });
    return result;
}
