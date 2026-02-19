import { version } from "$app/environment";
import { resolveBaseUrl } from "$lib/server/url";
import type { RequestEvent } from "@sveltejs/kit";

const NOMINATIM_RATE_LIMIT_MS = 1000;
const NOMINATIM_MAX_RETRIES = 2;
let lastNominatimCall = 0;

function getNominatimBaseUrl(): string {
    return resolveBaseUrl("NOMINATIM_URL", "https://nominatim.openstreetmap.org");
}

function needsRateLimiting(baseUrl: string): boolean {
    return baseUrl.includes("nominatim.openstreetmap.org");
}

const waitTimer = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

async function nominatimRateLimiter(baseUrl: string) {
    if (!needsRateLimiting(baseUrl)) {
        return;
    }

    const elapsedTimeMs = Date.now() - lastNominatimCall;
    const waitTime = NOMINATIM_RATE_LIMIT_MS - elapsedTimeMs;
    if (waitTime > 0) {
        await waitTimer(waitTime);
    }

    lastNominatimCall = Date.now();
}

export async function fetchNominatim(event: RequestEvent, path: string, params: URLSearchParams): Promise<Response> {
    const baseUrl = getNominatimBaseUrl();
    const base = new URL(baseUrl.endsWith("/") ? baseUrl : `${baseUrl}/`);
    const cleanPath = path.replace(/^\/+/, "");
    const url = new URL(cleanPath, base);
    const query = params.toString();
    if (query.length) {
        url.search = query;
    }

    let attempt = 0;

    while (true) {
        await nominatimRateLimiter(baseUrl);

        try {
            return await event.fetch(url.toString(), {
                method: "GET",
                headers: {
                    "User-Agent": `wanderer/${version}`,
                },
            });
        } catch (error) {
            if (attempt < NOMINATIM_MAX_RETRIES) {
                attempt++;
                continue;
            }
            throw new Error(`Nominatim fetch failed for ${url.toString()}`, { cause: error });
        }
    }
}
