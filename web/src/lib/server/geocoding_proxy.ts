import { NOMINATIM_URL, POCKETBASE_PROXY_SECRET, PUBLIC_NOMINATIM_URL } from "$app/env/private";
import type { RequestEvent } from "@sveltejs/kit";
import { clientAddress } from "#lib/server/http.ts";
import { normalizeBaseUrl } from "#lib/server/url.ts";

// Leave room for the backend's upstream timeouts, rate limiting, and retries.
const PROXY_TIMEOUT_MS = 120_000;
const REVERSE_NO_RESULT = "Unable to geocode";
// Only db can add this internal marker; upstream response headers are discarded.
const DB_FAILURE_HEADER = "X-Wanderer-Geocoding-Failure";
const DB_PROXY_AUTH_FAILURE = "internal_proxy_auth";
const DB_PRIVACY_HEADER = "X-Wanderer-Geocoding-Privacy";
const PUBLIC_NOMINATIM_HOST = "nominatim.openstreetmap.org";

function requiresPrivateNominatim() {
    // Match the old precedence, including Vite's dotenv settings. Only the flag goes to db.
    const value = NOMINATIM_URL ?? PUBLIC_NOMINATIM_URL;
    if (value === undefined) {
        return false;
    }
    const base = normalizeBaseUrl(value);
    // Only the public root URL counts as the shipped default. Reject alternate
    // paths, credentials and malformed spellings that URL parsing can normalize.
    if (!/^https?:\/\/[a-z0-9.-]+(?::[0-9]+)?\/?$/i.test(base)) {
        return true;
    }
    try {
        const url = new URL(base);
        return url.port !== "" || url.hostname.replace(/\.$/, "") !== PUBLIC_NOMINATIM_HOST;
    } catch {
        return true;
    }
}

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
        console.warn("Geocoding disabled: POCKETBASE_PROXY_SECRET is not configured on web");
        return failed();
    }

    try {
        const url = new URL(event.locals.pb.buildURL(`/geocoding/${endpoint}`));
        url.search = params.toString();
        const signal = AbortSignal.any([event.request.signal, AbortSignal.timeout(PROXY_TIMEOUT_MS)]);
        const headers = new Headers({ "X-Internal-Secret": POCKETBASE_PROXY_SECRET });
        if (requiresPrivateNominatim()) {
            headers.set(DB_PRIVACY_HEADER, "private-required");
        }
        const address = clientAddress(event);
        if (address) {
            headers.set("X-Forwarded-For", address);
        }
        // Unlike the SDK or event.fetch, raw fetch won't inherit visitor credentials or headers.
        // Manual mode on the server rejects redirects below without following them.
        const response = await globalThis.fetch(url, {
            method: "GET",
            headers,
            credentials: "omit",
            redirect: "manual",
            signal,
        });
        if (!response.ok) {
            void response.body?.cancel().catch(() => {});
            const { status } = response;
            if (status >= 300 && status < 400) {
                console.warn("Geocoding proxy blocked a redirect from db");
                return failed();
            }
            if (status === 401 && response.headers.get(DB_FAILURE_HEADER) === DB_PROXY_AUTH_FAILURE) {
                console.warn("Geocoding proxy failed: db rejected the internal proxy secret");
                return failed();
            }
            return failed(status >= 400 ? status : 502);
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
