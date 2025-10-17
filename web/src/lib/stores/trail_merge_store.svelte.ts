import type { MergeSettings } from "$lib/components/trail/trail_merge_modal.svelte";
import type { Trail } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";

export type Merge = {
    trailTarget: Trail,
    trailSource: Trail;
    status: "enqueued" | "merging" | "cancelled" | "success" | "error";
    error?: string;
    progress: number;
    settings: MergeSettings;
    function: (t: Trail, t2: Trail, settings: MergeSettings, onProgress?: (p: number) => void) => Promise<unknown>
};

class MergeStore {
    enqueuedMerges: Merge[] = $state([]);
    completedMerges: Merge[] = $state([]);
    merging: boolean = $state(false);
}

export const mergeStore = new MergeStore();


export async function processMergeQueue(batchSize: number = 3) {
    if (mergeStore.merging) {
        return;
    }
    mergeStore.merging = true;

    while (mergeStore.enqueuedMerges.length > 0) {
        const batch = mergeStore.enqueuedMerges.slice(0, batchSize);
        const mergePromises: Promise<unknown>[] = [];
        for (const b of batch) {
            b.status = "merging";
            mergePromises.push(
                b.function(b.trailTarget, b.trailSource, b.settings, (p: number) => {
                    b.progress = p
                })
            );
        }
        const results = await Promise.all(
            mergePromises.map((p) => p.catch((e) => e)),
        );
        results.forEach((r, i) => {
            const u = batch[i];
            if (r instanceof APIError) {
                u.status = "error"
                u.error = r.message
            } else {
                u.status = "success"
            }
            mergeStore.completedMerges.push(u);
        });
        mergeStore.enqueuedMerges.splice(0, batchSize)
    }
    mergeStore.merging = false;
}