import { env as privateEnv } from "$env/dynamic/private";
import { env as publicEnv } from "$env/dynamic/public";
import type { RequestEvent } from "@sveltejs/kit";

const OVERPASS_MAX_RETRIES = 2;

function normalizeBaseUrl(url: string): string {
    if (!/^https?:\/\//i.test(url)) {
        return `https://${url}`;
    }
    return url;
}

function getOverpassBaseUrl(): string {
    const rawUrl =
        privateEnv.PRIVATE_OVERPASS_API_URL ??
        publicEnv.PUBLIC_OVERPASS_API_URL ??
        "https://overpass-api.de";
    return normalizeBaseUrl(rawUrl);
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
