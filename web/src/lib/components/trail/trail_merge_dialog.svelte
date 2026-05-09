<script lang="ts">
    import {
        processMergeQueue,
        mergeStore,
        type Merge,
    } from "$lib/stores/trail_merge_store.svelte";
    import { slide } from "svelte/transition";
    import { _ } from "svelte-i18n";
    import { goto } from "$app/navigation";

    let minimized: boolean = $state(false);

    let visibleMerges = $derived(
        mergeStore.enqueuedMerges
            .concat(mergeStore.completedMerges)
            .sort((a, b) =>
                (a.trailSource.name || a.trailSource.id || "").localeCompare(
                    b.trailSource.name || b.trailSource.id || "",
                ),
            ),
    );

    let remaining = $derived(mergeStore.enqueuedMerges.length);

    let successfulMerges = $derived(
        mergeStore.completedMerges.reduce(
            (sum, u) => (sum += u.status == "success" ? 1 : 0),
            0,
        ),
    );

    let errorMerges = $derived(
        mergeStore.completedMerges.reduce(
            (sum, u) => (sum += u.status == "error" ? 1 : 0),
            0,
        ),
    );

    function dismissMerge(u: Merge) {
        const index = mergeStore.completedMerges.indexOf(u);
        mergeStore.completedMerges.splice(index, 1);
    }

    function dismissAllCompleted() {
        mergeStore.completedMerges = [];
    }

    function cancelMerge(u: Merge) {
        const index = mergeStore.enqueuedMerges.indexOf(u);
        u.status = "cancelled";
        mergeStore.enqueuedMerges.splice(index, 1);
        mergeStore.completedMerges.push(u);
    }

    function reMerge(u: Merge) {
        const index = mergeStore.completedMerges.indexOf(u);
        u.status = "enqueued";
        u.progress = 0;
        u.error = undefined;
        mergeStore.completedMerges.splice(index, 1);
        mergeStore.enqueuedMerges.push(u);
        processMergeQueue(undefined);
    }
</script>

{#if visibleMerges.length}
    <div
        class="fixed bottom-4 right-4 z-10 p-4 bg-background rounded-xl border border-input-border shadow-xl"
        class:cursor-pointer={minimized}
        in:slide
        out:slide
        role="presentation"
        onclick={(e) => {
            e.stopPropagation();
            minimized = false;
        }}
    >
        <div class="flex gap-x-2 items-start justify-between">
            <div>
                <p class="font-medium">
                    {$_("trail-merge-summary", {
                        values: {
                            remaining,
                            processed: mergeStore.completedMerges.length,
                            total:
                                mergeStore.enqueuedMerges.length +
                                mergeStore.completedMerges.length,
                        },
                    })}
                </p>
                <p class="text-sm">
                    {$_("trail-merge-result-summary", {
                        values: {
                            success: successfulMerges,
                            errors: errorMerges,
                        },
                    })}
                </p>
            </div>
            <div class="space-x-2">
                <button title={$_('clear-all')} aria-label={$_('clear-all')} onclick={dismissAllCompleted}
                    ><i class="fa fa-ban"></i></button
                >
                <button
                    aria-label={$_("minimize")}
                    onclick={(e) => {
                        e.stopPropagation();
                        minimized = true;
                    }}><i class="fa fa-minus"></i></button
                >
            </div>
        </div>
        <div
            class="max-h-96 max-w-72 mt-4 overflow-y-auto space-y-2"
            class:hidden={minimized}
        >
            {#each visibleMerges as u}
                <div class="bg-menu-item-background-hover rounded-lg py-2 px-3">
                    <div class="flex items-center gap-2">
                        <div class="w-6 shrink-0">
                            {#if u.status === "enqueued" || u.status == "merging"}
                                <div class="spinner spinner-small"></div>
                            {:else}
                                <i
                                    class={{
                                        fa: true,
                                        "fa-circle-exclamation text-red-400":
                                            u.status == "error",
                                        "fa-circle-check text-emerald-400":
                                            u.status == "success",
                                        "fa-ban text-gray-500":
                                            u.status == "cancelled",
                                    }}
                                ></i>
                            {/if}
                        </div>
                        <p class="text-xs basis-full min-w-0 break-all mr-2">
                            {u.trailSource.name}
                        </p>
                        {#if u.status == "error" || u.status == "cancelled"}
                            <button
                                aria-label={$_("trail-merge-retry")}
                                onclick={() => reMerge(u)}
                                ><i class="fa fa-redo text-sm"></i></button
                            >
                        {/if}
                        {#if u.status == "enqueued"}
                            <button
                                aria-label={$_("trail-merge-cancel")}
                                onclick={() => cancelMerge(u)}
                                ><i class="fa fa-stop text-sm"></i></button
                            >
                        {/if}
                        {#if u.status != "enqueued" && u.status != "merging"}
                            <button
                                aria-label={$_("dismiss")}
                                onclick={() => dismissMerge(u)}
                                ><i class="fa fa-close text-sm"></i></button
                            >
                        {/if}
                    </div>

                    {#if u.status == "merging"}
                        <div
                            class="progress-bar my-1 rounded-md"
                            style="height:2px; width:{u.progress * 100}%; background-color:#3549bb;transition: width 0.5s ease-in-out;"
                        ></div>
                    {:else if u.error}
                        <p class="text-red-400 text-xs">
                            {u.error}
                        </p>
                    {/if}
                </div>
            {/each}
        </div>
    </div>
{/if}
