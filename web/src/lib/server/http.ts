
function safeJson(text: string): any {
    try {
        return JSON.parse(text);
    } catch {
        return { message: text };
    }
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
