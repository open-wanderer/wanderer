import { browser } from "$app/environment";
import { APIError } from "$lib/util/api_util";
import { get, writable } from "svelte/store";

export interface TrailPublication {
    id: string;
    trailId: string;
    status: "running" | "completed" | "failed";
    total: number;
    processed: number;
    failed: number;
    error?: string;
    monitoringError?: boolean;
    name?: string;
}

type Fetch = (url: RequestInfo | URL, config?: RequestInit) => Promise<Response>;

export const trailPublications = writable<Record<string, TrailPublication>>({});
const pending = new Map<string, Promise<TrailPublication | null>>();

function remember(job: TrailPublication) {
    // Server-side imports can also publish, but their state must not be shared
    // between requests through the browser's progress store.
    if (browser) trailPublications.update((jobs) => ({ ...jobs, [job.trailId]: { name: jobs[job.trailId]?.name, ...job } }));
    return job;
}

export async function trails_publication_status(id: string, f: Fetch = fetch): Promise<TrailPublication | null> {
    const response = await f(`/api/v1/trail/${id}/publication`, { method: "GET" });
    const data = await response.json();
    if (!response.ok) throw new APIError(response.status, data.message, data.detail);
    return data.status === "idle" ? null : data;
}

async function missingPublication(job: TrailPublication, f: Fetch): Promise<TrailPublication> {
    const response = await f(`/api/v1/trail/${job.trailId}`, { method: "GET" });
    if (!response.ok) throw new APIError(response.status, "asset_publish_status_failed");
    const trail = await response.json();
    return remember({ ...job, monitoringError: false, status: trail.public ? "completed" : "failed", error: trail.public ? undefined : "asset_publish_interrupted" });
}

async function waitForPublication(initial: TrailPublication, f: Fetch): Promise<TrailPublication> {
    let job = remember(initial);
    let delay = 1000;
    let consecutiveErrors = 0;
    while (job.status === "running") {
        await new Promise((resolve) => setTimeout(resolve, delay));
        delay = Math.min(5000, delay * 1.5);
        try {
            const next = await trails_publication_status(job.trailId, f);
            if (!next) {
                // A restart clears job status. Publishing may have completed
                // just before that restart, so check the persisted trail first.
                job = await missingPublication(job, f);
            } else {
                job = remember(next);
            }
            consecutiveErrors = 0;
        } catch (error) {
            if (++consecutiveErrors < 3) continue;
            remember({ ...job, monitoringError: true });
            throw new APIError(503, "asset_publish_status_failed");
        }
    }
    if (job.status === "failed") throw new APIError(400, job.error || "asset_publish_failed");
    return job;
}

function singleMonitor(id: string, operation: () => Promise<TrailPublication | null>): Promise<TrailPublication | null> {
    if (!browser) return operation();
    const existing = pending.get(id);
    if (existing) return existing;
    const promise = operation().finally(() => {
        if (pending.get(id) === promise) pending.delete(id);
    });
    pending.set(id, promise);
    return promise;
}

export function trails_publish(id: string, f: Fetch = fetch, name?: string): Promise<TrailPublication | null> {
    return singleMonitor(id, async () => {
        const response = await f(`/api/v1/trail/${id}/publication`, { method: "POST" });
        const data = await response.json();
        if (!response.ok) throw new APIError(response.status, data.message, data.detail);
        return waitForPublication({ ...data, name: name ?? get(trailPublications)[id]?.name }, f);
    });
}

export async function trails_resume_publication(id: string, f: Fetch = fetch, name?: string): Promise<TrailPublication | null> {
    if (browser && pending.has(id)) return pending.get(id)!;
    const job = await trails_publication_status(id, f);
    if (!job) {
        const previous = get(trailPublications)[id];
        return previous?.status === "running" ? missingPublication(previous, f) : null;
    }
    name ??= get(trailPublications)[id]?.name;
    // Do not resurrect historic completed jobs on every page visit. A tracked
    // job, however, must clear its connection warning when it has finished.
    if (job.status !== "completed" || get(trailPublications)[id]) remember({ ...job, name });
    return job.status === "running" ? singleMonitor(id, () => waitForPublication({ ...job, name }, f)) : job;
}
