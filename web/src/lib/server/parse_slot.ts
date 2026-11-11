import { ClientResponseError } from "pocketbase";

// A heavy track file takes most of a small server's heap while it is parsed,
// so two at once can kill the process however small each one is.
const MAX_WAITING = 8;
const ELEVATION_TIMEOUT_MS = 30_000;

type Fetch = (url: RequestInfo | URL, config?: RequestInit) => Promise<Response>;

let busy = false;
const waiting: (() => void)[] = [];

/** Runs `task` once no other track file is being parsed. */
export async function withParseSlot<T>(task: () => Promise<T>): Promise<T> {
    if (busy) {
        if (waiting.length >= MAX_WAITING) {
            throw new ClientResponseError({ status: 503, response: { message: "server_busy" } });
        }
        await new Promise<void>((resolve) => waiting.push(resolve));
    }
    busy = true;
    try {
        return await task();
    } finally {
        const next = waiting.shift();
        if (next) {
            next();
        } else {
            busy = false;
        }
    }
}

/** A fetch for the elevation lookup inside a slot, so a hung Valhalla cannot hold it. */
export function elevationFetch(f: Fetch): Fetch {
    return (url, config) => f(url, { ...config, signal: AbortSignal.timeout(ELEVATION_TIMEOUT_MS) });
}
