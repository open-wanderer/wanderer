<script lang="ts">
    import { trailPublications, trails_resume_publication } from "$lib/stores/trail_publication_store";
    import { upsertBackgroundTask } from "$lib/stores/background_task_store.svelte";
    import { trailSaveErrorKey } from "$lib/stores/trail_store";
    import { APIError } from "$lib/util/api_util";
    import { untrack } from "svelte";
    import { _ } from "svelte-i18n";

    const dismissed = new Set<string>();
    $effect(() => {
        for (const job of Object.values($trailPublications)) {
            if (dismissed.has(job.id)) continue;
            const detail = job.monitoringError
                ? $_("asset_publish_status_failed")
                : job.status === "failed"
                  ? $_(trailSaveErrorKey(new APIError(400, job.error || "asset_publish_failed")))
                  : job.status === "completed"
                    ? $_("trail-publication-completed")
                    : `${$_("trail-publication-progress", { values: { processed: job.processed, total: job.total } })} ${$_("trail-publication-background")}`;
            const task = {
                id: `publication-${job.trailId}`,
                title: `${$_("trail-publication-title")}${job.name ? `: ${job.name}` : ""}`,
                status: job.monitoringError ? "warning" as const : job.status === "failed" ? "error" as const : job.status === "completed" ? "success" as const : "running" as const,
                detail,
                progress: job.status === "completed" ? 100 : job.total ? 100 * job.processed / job.total : 0,
                actions: [
                    { label: $_("edit"), icon: "external-link", href: `/trail/edit/${job.trailId}` },
                    ...(job.monitoringError ? [{ label: $_("trail-publication-check-status"), icon: "refresh", run: async () => { await trails_resume_publication(job.trailId).catch(() => {}); } }] : []),
                ],
                onDismiss: () => { dismissed.add(job.id); },
            };
            untrack(() => upsertBackgroundTask(task));
        }
    });
</script>
