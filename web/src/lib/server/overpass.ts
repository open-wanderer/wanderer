import { env as privateEnv } from "$env/dynamic/private";
import type { RequestEvent } from "@sveltejs/kit";

const OVERPASS_MAX_RETRIES = 2;
function getOverpassBaseUrl(): string {
    return privateEnv.PRIVATE_OVERPASS_API_URL ?? "";
}

export async function fetchOverpass(event: RequestEvent, params: URLSearchParams): Promise<Response> {
    const baseUrl = getOverpassBaseUrl();
    if (!baseUrl) {
        throw new Error("PRIVATE_OVERPASS_API_URL not set");
    }
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
