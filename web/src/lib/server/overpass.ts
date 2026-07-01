import { resolveBaseUrl } from "$lib/server/url";
import type { RequestEvent } from "@sveltejs/kit";

const OVERPASS_MAX_RETRIES = 2;

function getOverpassBaseUrl(): string {
    return resolveBaseUrl("OVERPASS_API_URL", "https://overpass-api.de");
}

export async function fetchOverpass(event: RequestEvent, params: URLSearchParams): Promise<Response> {
    const baseUrl = getOverpassBaseUrl();
    const base = new URL(baseUrl.endsWith("/") ? baseUrl : `${baseUrl}/`);
    const url = new URL("api/interpreter", base);
    const query = params.toString();
    if (query.length) {
        url.search = query;
    }

    let attempt = 0;

    while (true) {
        try {
            return await event.fetch(url.toString(), {
                method: "GET",
            });
        } catch (error) {
            if (attempt < OVERPASS_MAX_RETRIES) {
                attempt++;
                continue;
            }
            throw error;
        }
    }
}
