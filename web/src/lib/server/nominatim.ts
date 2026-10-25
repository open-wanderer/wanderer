import { version } from "$app/environment";
import { resolveBaseUrl } from "$lib/server/url";
import type { RequestEvent } from "@sveltejs/kit";

const NOMINATIM_RATE_LIMIT_MS = 1000;
const NOMINATIM_MAX_RETRIES = 2;
// Requests are serialized, so a slow response or a long Retry-After holds up
// every other caller. Bound both.
const NOMINATIM_TIMEOUT_MS = 10_000;
const NOMINATIM_MAX_RETRY_AFTER_MS = 30_000;
let lastNominatimCall = 0;
let nominatimRequestQueue: Promise<void> = Promise.resolve();

function getNominatimBaseUrl(): string {
    return resolveBaseUrl("NOMINATIM_URL", "https://nominatim.openstreetmap.org");
}

function needsRateLimiting(baseUrl: string): boolean {
    return baseUrl.includes("nominatim.openstreetmap.org");
}

const waitTimer = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

async function waitForNominatimSlot(baseUrl: string) {
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

async function withNominatimQueue<T>(baseUrl: string, request: () => Promise<T>): Promise<T> {
    if (!needsRateLimiting(baseUrl)) {
        return request();
    }

    const previousRequest = nominatimRequestQueue;
    let releaseRequest!: () => void;
    nominatimRequestQueue = new Promise<void>(resolve => {
        releaseRequest = resolve;
    });

    await previousRequest;
    try {
        return await request();
    } finally {
        releaseRequest();
    }
}

function retryAfterMs(header: string | null, attempt: number): number {
    if (header !== null) {
        const seconds = Number(header);
        if (Number.isFinite(seconds) && seconds >= 0) {
            return seconds * 1000;
        }

        const retryAt = Date.parse(header);
        if (Number.isFinite(retryAt)) {
            return Math.max(0, retryAt - Date.now());
        }
    }

    return NOMINATIM_RATE_LIMIT_MS * 2 ** (attempt - 1);
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

    return withNominatimQueue(baseUrl, async () => {
        let attempt = 0;

        while (true) {
            await waitForNominatimSlot(baseUrl);

            const controller = new AbortController();
            const timeout = setTimeout(() => controller.abort(), NOMINATIM_TIMEOUT_MS);
            try {
                const response = await event.fetch(url.toString(), {
                    method: "GET",
                    headers: {
                        "User-Agent": `wanderer/${version}`,
                    },
                    signal: controller.signal,
                });

                if (response.status === 429 && attempt < NOMINATIM_MAX_RETRIES) {
                    const delay = retryAfterMs(response.headers.get("Retry-After"), attempt + 1);
                    if (delay <= NOMINATIM_MAX_RETRY_AFTER_MS) {
                        await response.body?.cancel();
                        attempt++;
                        await waitTimer(delay);
                        continue;
                    }
                }

                return response;
            } catch (error) {
                if (attempt < NOMINATIM_MAX_RETRIES) {
                    attempt++;
                    continue;
                }
                throw new Error(`Nominatim fetch failed for ${url.toString()}`, { cause: error });
            } finally {
                clearTimeout(timeout);
            }
        }
    });
}
