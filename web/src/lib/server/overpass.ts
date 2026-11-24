import { version } from "$app/env";
import { resolveBaseUrl } from "#lib/server/url.ts";
import type { RequestEvent } from "@sveltejs/kit";

// overpass-api.de answers Node's default "node" User-Agent with 406, and its
// usage policy asks for a descriptive one.
const OVERPASS_USER_AGENT = `wanderer/${version} (+https://github.com/open-wanderer/wanderer)`;
const OVERPASS_TIMEOUT_MS = 30_000;

function getOverpassBaseUrl(): string {
    return resolveBaseUrl("OVERPASS_API_URL", "https://overpass-api.de");
}

/**
 * One attempt, no retries: the browser layer backs off after failures, and
 * retrying here as well multiplied the load on an already overloaded server.
 */
export async function fetchOverpass(event: RequestEvent, params: URLSearchParams): Promise<Response> {
    const baseUrl = getOverpassBaseUrl();
    const base = new URL(baseUrl.endsWith("/") ? baseUrl : `${baseUrl}/`);
    const url = new URL("api/interpreter", base);
    const query = params.toString();
    if (query.length) {
        url.search = query;
    }

    return event.fetch(url.toString(), {
        method: "GET",
        headers: {
            "User-Agent": OVERPASS_USER_AGENT,
        },
        signal: AbortSignal.timeout(OVERPASS_TIMEOUT_MS),
    });
}
