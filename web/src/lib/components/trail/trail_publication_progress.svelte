<script lang="ts">
    import type { Trail } from "$lib/models/trail";
    import { trailPublications, trails_publish, trails_resume_publication } from "$lib/stores/trail_publication_store";
    import { trails_show, trailSaveErrorKey } from "$lib/stores/trail_store";
    import { APIError } from "$lib/util/api_util";
    import { untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import Button from "../base/button.svelte";

    let { trailId, name, publicTrail = false, onpublished, onretry }: {
        trailId?: string;
        name?: string;
        publicTrail?: boolean;
        onpublished?: (trail: Trail) => void;
        onretry?: () => void;
    } = $props();
    let retrying = $state(false);
    let localError = $state("");
    const job = $derived(trailId ? $trailPublications[trailId] : undefined);

    $effect(() => {
        const id = trailId;
        if (!id || publicTrail) return;
        let disposed = false;
        untrack(() => {
            void trails_resume_publication(id, fetch, name).then(async (result) => {
                if (result?.status === "completed" && !disposed) {
                    const saved = await trails_show(id);
                    if (!disposed) onpublished?.(saved);
                }
            }).catch(() => { /* Job failures are displayed below and in background tasks. */ });
        });
        return () => { disposed = true; };
    });

    async function retry() {
        if (!trailId) return;
        if (onretry && !job?.monitoringError) {
            onretry();
            return;
        }
        retrying = true;
        localError = "";
        try {
            const result = job?.monitoringError
                ? await trails_resume_publication(trailId, fetch, name)
                : await trails_publish(trailId, fetch, name);
            if (result?.status === "completed") onpublished?.(await trails_show(trailId));
        } catch (error) {
            localError = trailSaveErrorKey(error);
        } finally {
            retrying = false;
        }
    }
</script>

{#if job && !publicTrail && job.status !== "completed"}
    <div class="rounded-xl border border-input-border p-4 space-y-2" role="status" aria-live="polite">
        <p class="font-medium">{$_("trail-publication-title")}</p>
        {#if job.status === "running" && !job.monitoringError}
            <p>{$_("trail-publication-progress", { values: { processed: job.processed, total: job.total } })}</p>
            {#if job.total > 0}
                <progress class="w-full" max={job.total} value={job.processed}></progress>
            {/if}
            <p class="text-sm text-content/75">{$_("trail-publication-background")}</p>
        {:else}
            <p>{$_(localError || (job.monitoringError ? "asset_publish_status_failed" : trailSaveErrorKey(new APIError(400, job.error || "asset_publish_failed"))))}</p>
            <Button secondary type="button" loading={retrying} onclick={retry}>
                {$_(job.monitoringError ? "trail-publication-check-status" : "trail-publication-retry")}
            </Button>
        {/if}
    </div>
{/if}
