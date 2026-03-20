import type { MergeSettings } from "$lib/components/trail/trail_merge_modal.svelte";
import type { Trail } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";
import { show_toast } from "./toast_store.svelte";
import { get } from "svelte/store";
import { _ } from "svelte-i18n";

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

function getMergeErrorMessage(error: unknown): string {
    if (error instanceof APIError) {
        return error.message;
    }

    if (error instanceof Error) {
        return error.message;
    }

    if (typeof error === "string") {
        return error;
    }

    return "Unknown merge error";
}

export async function processMergeQueue(batchSize: number = 3) {
    if (mergeStore.merging) {
        return;
    }
    mergeStore.merging = true;
    const completedBeforeRun = mergeStore.completedMerges.length;

    try {
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
                if (r instanceof Error || typeof r === "string") {
                    u.status = "error";
                    u.error = getMergeErrorMessage(r);
                } else {
                    u.status = "success";
                    u.error = undefined;
                }
                mergeStore.completedMerges.push(u);
            });
            mergeStore.enqueuedMerges.splice(0, batchSize)
        }
    } finally {
        mergeStore.merging = false;
    }

    const completedThisRun = mergeStore.completedMerges.slice(completedBeforeRun);
    if (completedThisRun.length === 0) {
        return;
    }

    const successfulCount = completedThisRun.filter((merge) => merge.status === "success").length;
    const errorCount = completedThisRun.filter((merge) => merge.status === "error").length;

    if (errorCount === 0) {
        show_toast({
            type: "success",
            icon: "check",
            text: get(_)("trail-merge-complete", {
                values: { success: successfulCount },
            }),
        });
        return;
    }

    show_toast({
        type: successfulCount > 0 ? "warning" : "error",
        icon: successfulCount > 0 ? "triangle-exclamation" : "close",
        text: get(_)("trail-merge-complete-with-errors", {
            values: { success: successfulCount, errors: errorCount },
        }),
    }, 5000);
}
