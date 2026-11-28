import { POCKETBASE_PROXY_SECRET } from "$app/env/private";
import type { RequestEvent } from "@sveltejs/kit";
import { clientAddress } from "#lib/server/http.ts";

// Leave room for the backend's upstream timeouts, rate limiting, and retries.
const PROXY_TIMEOUT_MS = 120_000;
const REVERSE_NO_RESULT = "Unable to geocode";

function failed(status = 502) {
    return Response.json({ message: "Nominatim request failed" }, { status });
}

function isGeocodingResult(payload: unknown) {
    if (payload === null || typeof payload !== "object" || Array.isArray(payload) || "error" in payload) {
        return false;
    }
    // The Go client normalizes an empty upstream success to {}.
    return Object.keys(payload).length === 0 ||
        ("type" in payload && payload.type === "FeatureCollection" &&
            "features" in payload && Array.isArray(payload.features));
}

function isReverseNoResult(payload: unknown) {
    if (payload === null || typeof payload !== "object" ||
        !("error" in payload) || payload.error !== REVERSE_NO_RESULT) {
        return false;
    }
    const keys = Object.keys(payload);
    return keys.length === 1 && keys[0] === "error";
}

export async function proxyGeocoding(event: RequestEvent, endpoint: "search" | "reverse", params: URLSearchParams) {
    if (!POCKETBASE_PROXY_SECRET) {
        return failed();
    }

    try {
        const url = new URL(event.locals.pb.buildURL(`/geocoding/${endpoint}`));
        url.search = params.toString();
        const signal = AbortSignal.any([event.request.signal, AbortSignal.timeout(PROXY_TIMEOUT_MS)]);
        const headers = new Headers({ "X-Internal-Secret": POCKETBASE_PROXY_SECRET });
        const address = clientAddress(event);
        if (address) {
            headers.set("X-Forwarded-For", address);
        }
        // Unlike the SDK or event.fetch, raw fetch won't inherit visitor credentials or headers.
        const response = await globalThis.fetch(url, {
            method: "GET",
            headers,
            credentials: "omit",
            redirect: "error",
            signal,
        });
        if (!response.ok) {
            void response.body?.cancel().catch(() => {});
            return failed(response.status);
        }

        // Keep the public 200 + JSON behavior without reflecting arbitrary error text
        // or envelopes. A stock reverse lookup may return a successful no-address result.
        const text = await response.text();
        if (!text.length) {
            return Response.json({});
        }
        const payload: unknown = JSON.parse(text);
        if (endpoint === "reverse" && isReverseNoResult(payload)) {
            return Response.json({ error: REVERSE_NO_RESULT });
        }
        return isGeocodingResult(payload) ? Response.json(payload) : failed();
    } catch {
        return failed();
    }
}
