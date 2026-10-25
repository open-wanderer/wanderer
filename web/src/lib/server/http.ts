import { json } from "@sveltejs/kit";

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
        return json(payload, { status: response.status });
    }
    return json(payload);
}

// Hop-by-hop headers must not be forwarded; Node's fetch rejects some of
// them, e.g. "Connection: upgrade".
const HOP_BY_HOP = new Set([
    "connection", "keep-alive", "proxy-connection", "te", "trailer", "transfer-encoding", "upgrade", "expect",
]);

export function forwardableHeaders(headers: Headers): Record<string, string> {
    const result: Record<string, string> = {};
    headers.forEach((value, key) => {
        if (!HOP_BY_HOP.has(key)) {
            result[key] = value;
        }
    });
    return result;
}
