<script lang="ts">
    import { goto } from "$app/navigation";
    import MergeDialog from "$lib/components/trail/trail_merge_dialog.svelte";
    import TrailListItem from "$lib/components/trail/trail_list_item.svelte";
    import TrailMergeModal, {
        type MergeSelection,
        type MergeSettings,
    } from "$lib/components/trail/trail_merge_modal.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import type { Trail } from "$lib/models/trail";
    import {
        type TrailMergeSuggestGroup,
        trail_merge,
        trail_merge_suggest_groups,
    } from "$lib/stores/trail_merge_api";
    import { translateTrailMergeError } from "$lib/stores/trail_merge_i18n";
    import {
        mergeStore,
        processMergeQueue,
        type Merge,
    } from "$lib/stores/trail_merge_store.svelte";
    import { trails_show } from "$lib/stores/trail_store";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import { APIError } from "$lib/util/api_util";
    import { _ } from "svelte-i18n";
    import { onMount } from "svelte";

    type SimilarTrailGroupView = TrailMergeSuggestGroup & {
        trails: Trail[];
        targetTrail?: Trail;
        suggestedTargetTrailId: string;
    };

    let loading = $state(true);
    let groups = $state<SimilarTrailGroupView[]>([]);
    let loadError = $state("");
    let openMapGroupId = $state<string | null>(null);
    let mapLoadingGroupId = $state<string | null>(null);
    let mapTrailsByGroupId = $state<Record<string, Trail[]>>({});
    let trailById = $state<Record<string, Trail>>({});
    let trailWithGpxById = $state<Record<string, Trail>>({});

    let trailMergeModal: TrailMergeModal;

    onMount(async () => {
        await loadGroups();
    });

    async function loadGroups() {
        loading = true;
        loadError = "";
        openMapGroupId = null;

        try {
            const response = await trail_merge_suggest_groups();
            const uniqueTrailIds = Array.from(
                new Set(response.groups.flatMap((group) => group.trailIds)),
            );
            const missingTrailIds = uniqueTrailIds.filter((trailId) => !trailById[trailId]);

            if (missingTrailIds.length > 0) {
                const loadedTrails = await Promise.all(
                    missingTrailIds.map((trailId) => trails_show(trailId)),
                );

                trailById = {
                    ...trailById,
                    ...Object.fromEntries(
                        loadedTrails
                            .filter((trail) => Boolean(trail.id))
                            .map((trail) => [trail.id!, trail]),
                    ),
                };
            }

            groups = await Promise.all(
                response.groups.map(async (group) => {
                    const trails = group.trailIds
                        .map((trailId) => trailById[trailId])
                        .filter((trail): trail is Trail => Boolean(trail));

                    return {
                        ...group,
                        trails,
                        targetTrail: trails.find((trail) => trail.id === group.targetTrailId),
                        suggestedTargetTrailId: group.targetTrailId,
                    } satisfies SimilarTrailGroupView;
                }),
            );
        } catch (error) {
            console.error("Failed to load similar trail groups", error);
            loadError =
                error instanceof APIError
                    ? translateTrailMergeError(error.message)
                    : $_("trail-merge-unknown-error");
        } finally {
            loading = false;
        }
    }

    async function toggleMap(group: SimilarTrailGroupView) {
        if (openMapGroupId === group.groupId) {
            openMapGroupId = null;
            return;
        }

        openMapGroupId = group.groupId;
        if (mapTrailsByGroupId[group.groupId]) {
            return;
        }

        mapLoadingGroupId = group.groupId;
        try {
            const trailIdsToLoad = group.trails
                .map((trail) => trail.id)
                .filter((trailId): trailId is string => Boolean(trailId))
                .filter((trailId) => !trailWithGpxById[trailId]);

            if (trailIdsToLoad.length > 0) {
                const loadedTrails = await Promise.all(
                    trailIdsToLoad.map((trailId) =>
                        trails_show(trailId, undefined, undefined, true),
                    ),
                );

                trailWithGpxById = {
                    ...trailWithGpxById,
                    ...Object.fromEntries(
                        loadedTrails
                            .filter((trail) => Boolean(trail.id))
                            .map((trail) => [trail.id!, trail]),
                    ),
                };
            }

            const groupMapTrails = group.trails
                .map((trail) => trail.id)
                .filter((trailId): trailId is string => Boolean(trailId))
                .map((trailId) => trailWithGpxById[trailId])
                .filter((trail): trail is Trail => Boolean(trail));

            mapTrailsByGroupId = {
                ...mapTrailsByGroupId,
                [group.groupId]: groupMapTrails,
            };
        } finally {
            mapLoadingGroupId = null;
        }
    }

    async function openMergeGroupModal(group: SimilarTrailGroupView) {
        await trailMergeModal.openModal(group.trails, {
            fixedTargetTrailId: group.targetTrailId,
        });
    }

    function updateGroupTarget(groupId: string, targetTrailId: string) {
        groups = groups.map((group) => {
            if (group.groupId !== groupId) {
                return group;
            }

            return {
                ...group,
                targetTrailId,
                targetTrail: group.trails.find((trail) => trail.id === targetTrailId),
            };
        });
    }

    async function mergeGroup(settings: MergeSettings, selection: MergeSelection) {
        let trailTarget = selection.targetTrail;
        if (!trailTarget.id) {
            return;
        }

        if (!trailTarget.expand) {
            trailTarget = await trails_show(trailTarget.id);
        }

        for (const sourceTrail of selection.sourceTrails) {
            if (sourceTrail.id === trailTarget.id) {
                continue;
            }

            const mergeJob: Merge = {
                trailTarget,
                trailSource: sourceTrail,
                progress: 0,
                status: "enqueued",
                settings,
                function: async (target, source, mergeSettings, onProgress) => {
                    if (!target.id || !source.id) {
                        throw new Error($_("error-merging-trail"));
                    }

                    onProgress?.(0.2);
                    await trail_merge(source.id, target.id, mergeSettings);
                    onProgress?.(1);
                },
            };

            mergeStore.enqueuedMerges.push(mergeJob);
        }

        const completedBeforeRun = mergeStore.completedMerges.length;
        await processMergeQueue();
        const completedThisRun = mergeStore.completedMerges.slice(completedBeforeRun);
        const successfulThisRun = completedThisRun.filter(
            (merge) => merge.status === "success",
        ).length;

        if (successfulThisRun > 0 && settings.delete) {
            await loadGroups();
        }
    }
</script>

<svelte:head>
    <title>{$_("similar-trails-maintenance-title")} | wanderer</title>
</svelte:head>

<div class="space-y-6">
    <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div class="space-y-2">
            <h1 class="text-3xl font-bold">{$_("similar-trails-maintenance-title")}</h1>
            <p class="text-sm text-gray-500 max-w-3xl">
                {$_("similar-trails-maintenance-description")}
            </p>
        </div>
        <button class="btn-secondary shrink-0" onclick={loadGroups} disabled={loading}>
            {$_("similar-trails-scan")}
        </button>
    </div>

    {#if loading}
        <div class="rounded-2xl border border-input-border p-6 flex items-center gap-3">
            <div class="spinner light:spinner-dark"></div>
            <p class="text-sm text-gray-500">{$_("similar-trails-loading")}</p>
        </div>
    {:else if loadError}
        <div class="rounded-2xl border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300">
            {loadError}
        </div>
    {:else if groups.length === 0}
        <div class="rounded-2xl border border-input-border p-6 text-sm text-gray-500">
            {$_("similar-trails-empty")}
        </div>
    {:else}
        <div class="space-y-6">
            {#each groups as group}
                <section class="rounded-2xl border border-input-border overflow-hidden">
                    <div class="p-5 flex flex-col gap-4">
                        <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                            <div class="space-y-2">
                                <div class="flex flex-wrap items-center gap-3">
                                    <h2 class="text-xl font-semibold">
                                        {$_("similar-trails-group-size", {
                                            values: { n: group.trails.length },
                                        })}
                                    </h2>
                                </div>
                                {#if group.targetTrailId === group.suggestedTargetTrailId}
                                    <p class="text-sm text-gray-500">
                                        {$_(`similar-trails-reason-${group.reason}`)}
                                    </p>
                                {/if}
                                {#if group.indirect}
                                    <div class="rounded-xl border border-yellow-500/40 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-800 dark:text-yellow-200">
                                        {$_("similar-trails-indirect-hint")}
                                    </div>
                                {/if}
                            </div>
                            <div class="flex flex-col items-end gap-3 shrink-0">
                                <button class="btn-secondary" onclick={() => toggleMap(group)}>
                                    {openMapGroupId === group.groupId
                                        ? $_("similar-trails-hide-map")
                                        : $_("similar-trails-show-map")}
                                </button>
                                <button class="btn-primary" onclick={() => openMergeGroupModal(group)}>
                                    {$_("similar-trails-merge-group")}
                                </button>
                            </div>
                        </div>

                        {#if openMapGroupId === group.groupId}
                            <div class="border-t border-input-border p-5 space-y-3">
                                <h3 class="text-lg font-semibold">{$_("similar-trails-map-title")}</h3>
                                {#if mapLoadingGroupId === group.groupId}
                                    <div class="flex items-center gap-3 py-6">
                                        <div class="spinner light:spinner-dark"></div>
                                        <p class="text-sm text-gray-500">{$_("similar-trails-map-loading")}</p>
                                    </div>
                                {:else if mapTrailsByGroupId[group.groupId]}
                                    <div class="h-[26rem] rounded-2xl overflow-hidden border border-input-border">
                                        <MapWithElevationMaplibre
                                            trails={mapTrailsByGroupId[group.groupId]}
                                            showTerrain={true}
                                        />
                                    </div>
                                {/if}
                            </div>
                        {/if}

                        <div class="space-y-2">
                            {#each group.trails as trail}
                                <div class="relative group">
                                    <a
                                        class="block"
                                        href={`/trail/view/${handleFromRecordWithIRI(trail)}/${trail.id}`}
                                        onclick={(event) => {
                                            event.preventDefault();
                                            goto(`/trail/view/${handleFromRecordWithIRI(trail)}/${trail.id}`);
                                        }}
                                    >
                                        <TrailListItem
                                            {trail}
                                            selected={false}
                                            hovered={false}
                                            showDescription={false}
                                        />
                                    </a>
                                    <button
                                        type="button"
                                        class={`absolute bottom-4 right-4 z-10 flex h-9 w-9 items-center justify-center rounded-full border shadow-sm transition-all ${
                                            trail.id === group.targetTrailId
                                                ? "bg-primary text-white border-primary opacity-100"
                                                : "bg-background/95 text-gray-500 border-input-border opacity-0 group-hover:opacity-100 hover:border-primary hover:text-primary"
                                        }`}
                                        onclick={(event) => {
                                            event.preventDefault();
                                            event.stopPropagation();
                                            if (trail.id) {
                                                updateGroupTarget(group.groupId, trail.id);
                                            }
                                        }}
                                        aria-label={$_("similar-trails-set-target")}
                                        title={$_("similar-trails-set-target")}
                                    >
                                        <i class="fa fa-flag-checkered"></i>
                                    </button>
                                </div>
                            {/each}
                        </div>
                    </div>
                </section>
            {/each}
        </div>
    {/if}
</div>

<TrailMergeModal
    bind:this={trailMergeModal}
    onmerge={(settings, selection) => mergeGroup(settings, selection)}
/>
<MergeDialog />
