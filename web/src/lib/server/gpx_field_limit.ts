import { BODY_SIZE_LIMIT } from "$app/env/private";
import { PUBLIC_POCKETBASE_URL } from "$app/env/public";
import { DEFAULT_GPX_FIELD_LIMIT, setGpxFieldLimit } from "#lib/util/track_limits.ts";
import { uploadBodyLimit } from "./upload_body_limit.js";

const REFRESH_MS = 60_000;
const RETRY_MS = 10_000;
const TIMEOUT_MS = 3_000;
const MULTIPART_OVERHEAD = 64 * 1024;

let maxSize = DEFAULT_GPX_FIELD_LIMIT;
let nextRefresh = 0;

async function fetchMaxSize(): Promise<number | undefined> {
    try {
        const response = await fetch(`${PUBLIC_POCKETBASE_URL}/trail/gpx-limit`, { signal: AbortSignal.timeout(TIMEOUT_MS) });
        if (!response.ok) return undefined;
        const { maxSize } = await response.json();
        return Number.isSafeInteger(maxSize) && maxSize > 0 ? maxSize : undefined;
    } catch {
        return undefined;
    }
}

/** What the admin set on the gpx field of the trails collection. Keeps the last known value while the backend cannot say. */
export async function gpxFieldLimit(): Promise<number> {
    if (Date.now() < nextRefresh) return maxSize;
    const current = await fetchMaxSize();
    nextRefresh = Date.now() + (current === undefined ? RETRY_MS : REFRESH_MS);
    if (current !== undefined) {
        maxSize = current;
        setGpxFieldLimit(maxSize);
    }
    return maxSize;
}

/** The largest request that can carry a track file, which BODY_SIZE_LIMIT caps for the whole server. */
export async function trackBodyLimit(): Promise<number> {
    return Math.min(await gpxFieldLimit() + MULTIPART_OVERHEAD, uploadBodyLimit(BODY_SIZE_LIMIT));
}
