<script lang="ts">
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import {
        asset_merge,
        asset_merge_suggest_groups,
        type AssetMergeAsset,
        type AssetMergeSuggestGroup,
    } from "$lib/stores/asset_merge_api";
    import { APIError } from "$lib/util/api_util";
    import { _ } from "svelte-i18n";

    type SimilarAssetGroupView = AssetMergeSuggestGroup & {
        suggestedTargetAssetId: string;
    };

    let hasScanned = $state(false);
    let loading = $state(false);
    let groups = $state<SimilarAssetGroupView[]>([]);
    let loadError = $state("");
    let mergeError = $state("");
    let mergingGroupId = $state<string | null>(null);
    let pendingMergeGroup = $state<SimilarAssetGroupView | null>(null);

    let confirmModal: ConfirmModal;

    async function loadGroups() {
        hasScanned = true;
        loading = true;
        loadError = "";
        mergeError = "";

        try {
            const response = await asset_merge_suggest_groups();
            groups = response.groups.map((group) => ({
                ...group,
                suggestedTargetAssetId: group.targetAssetId,
            }));
        } catch (error) {
            console.error("Failed to load similar asset groups", error);
            loadError =
                error instanceof APIError
                    ? $_(error.message) || error.message
                    : $_("asset-merge-unknown-error");
        } finally {
            loading = false;
        }
    }

    function updateGroupTarget(groupId: string, targetAssetId: string) {
        groups = groups.map((group) => {
            if (group.groupId !== groupId) {
                return group;
            }
            return {
                ...group,
                targetAssetId,
            };
        });
    }

    function openMergeConfirm(group: SimilarAssetGroupView) {
        pendingMergeGroup = group;
        mergeError = "";
        confirmModal.openModal();
    }

    async function mergePendingGroup() {
        const group = pendingMergeGroup;
        if (!group) {
            return;
        }
        const sourceAssetIds = group.assetIds.filter((assetId) => assetId !== group.targetAssetId);
        if (!sourceAssetIds.length) {
            return;
        }

        mergingGroupId = group.groupId;
        mergeError = "";
        try {
            await asset_merge(sourceAssetIds, group.targetAssetId);
            await loadGroups();
        } catch (error) {
            console.error("Failed to merge asset group", error);
            mergeError =
                error instanceof APIError
                    ? $_(error.message) || error.message
                    : $_("asset-merge-unknown-error");
        } finally {
            mergingGroupId = null;
            pendingMergeGroup = null;
        }
    }

    function sourceCount(group: SimilarAssetGroupView | null) {
        if (!group) {
            return 0;
        }
        return group.assetIds.filter((assetId) => assetId !== group.targetAssetId).length;
    }

    function formatDate(value?: string) {
        if (!value) {
            return "";
        }
        return new Date(value).toLocaleDateString();
    }

    function locationText(asset: AssetMergeAsset) {
        if (asset.lat === undefined || asset.lon === undefined) {
            return "";
        }
        return `${asset.lat.toFixed(5)}, ${asset.lon.toFixed(5)}`;
    }

    function linkSummary(asset: AssetMergeAsset) {
        const parts = [];
        if (asset.links.trails) {
            parts.push($_("asset-merge-links-trails", { values: { n: asset.links.trails } }));
        }
        if (asset.links.waypoints) {
            parts.push(
                $_("asset-merge-links-waypoints", { values: { n: asset.links.waypoints } }),
            );
        }
        if (asset.links.summitLogs) {
            parts.push(
                $_("asset-merge-links-summit-logs", {
                    values: { n: asset.links.summitLogs },
                }),
            );
        }
        return parts.length ? parts.join(" · ") : $_("asset-merge-no-links");
    }

    function providerLabel(asset: AssetMergeAsset) {
        if (asset.externalProvider && asset.externalId) {
            return `${asset.externalProvider} · ${asset.externalId}`;
        }
        if (asset.storageMode === "link_private") {
            return $_("asset-merge-remote-asset");
        }
        return $_("asset-merge-local-asset");
    }
</script>

<div class="space-y-6">
    <div class="flex justify-end">
        <button class="btn-secondary shrink-0" onclick={loadGroups} disabled={loading}>
            <i class="fa fa-magnifying-glass mr-2"></i>
            {$_("similar-assets-scan")}
        </button>
    </div>

    {#if !hasScanned}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {$_("duplicate-maintenance-assets-ready")}
        </div>
    {:else if loading}
        <div class="rounded-lg border border-input-border p-6 flex items-center gap-3">
            <div class="spinner light:spinner-dark"></div>
            <p class="text-sm text-gray-500">{$_("similar-assets-loading")}</p>
        </div>
    {:else if loadError}
        <div class="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300">
            {loadError}
        </div>
    {:else if groups.length === 0}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {$_("similar-assets-empty")}
        </div>
    {:else}
        {#if mergeError}
            <div class="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300">
                {mergeError}
            </div>
        {/if}

        <div class="space-y-6">
            {#each groups as group}
                <section class="rounded-lg border border-input-border overflow-hidden">
                    <div class="p-5 flex flex-col gap-4">
                        <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                            <div class="space-y-2">
                                <h3 class="text-xl font-semibold">
                                    {$_("similar-assets-group-size", {
                                        values: { n: group.assets.length },
                                    })}
                                </h3>
                                <p class="text-sm text-gray-500">
                                    {$_(`similar-assets-match-${group.matchReason}`)}
                                </p>
                                {#if group.targetAssetId === group.suggestedTargetAssetId}
                                    <p class="text-sm text-gray-500">
                                        {$_(`similar-assets-reason-${group.reason}`)}
                                    </p>
                                {/if}
                            </div>
                            <button
                                class="btn-primary shrink-0"
                                disabled={mergingGroupId === group.groupId}
                                onclick={() => openMergeConfirm(group)}
                            >
                                <i class="fa fa-code-merge mr-2"></i>
                                {mergingGroupId === group.groupId
                                    ? $_("loading")
                                    : $_("similar-assets-merge-group")}
                            </button>
                        </div>

                        <div class="divide-y divide-separator overflow-hidden rounded-lg border border-input-border">
                            {#each group.assets as asset}
                                <div
                                    class="grid gap-4 p-3 md:grid-cols-[7rem_minmax(0,1fr)_max-content] md:items-center {asset.id === group.targetAssetId ? 'bg-menu-item-background-hover' : ''}"
                                >
                                    <img
                                        src={asset.thumbnailUrl}
                                        alt=""
                                        class="h-28 w-full rounded-lg bg-input-background object-cover md:h-24 md:w-28"
                                        loading="lazy"
                                    />
                                    <div class="min-w-0 space-y-2">
                                        <div class="flex min-w-0 flex-wrap items-center gap-2">
                                            <h4 class="truncate text-sm font-semibold">
                                                {asset.originalFileName}
                                            </h4>
                                            {#if asset.id === group.targetAssetId}
                                                <span class="rounded-full bg-primary px-2 py-0.5 text-xs font-medium text-white">
                                                    {$_("asset-merge-target")}
                                                </span>
                                            {/if}
                                        </div>
                                        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500">
                                            <span>{providerLabel(asset)}</span>
                                            {#if formatDate(asset.takenAt || asset.created)}
                                                <span>{formatDate(asset.takenAt || asset.created)}</span>
                                            {/if}
                                            {#if locationText(asset)}
                                                <span>{locationText(asset)}</span>
                                            {/if}
                                            <span>{linkSummary(asset)}</span>
                                        </div>
                                    </div>
                                    <button
                                        type="button"
                                        class={`flex h-9 w-9 items-center justify-center rounded-full border shadow-sm transition-all ${
                                            asset.id === group.targetAssetId
                                                ? "bg-primary text-white border-primary"
                                                : "bg-background/95 text-gray-500 border-input-border hover:border-primary hover:text-primary"
                                        }`}
                                        onclick={() => updateGroupTarget(group.groupId, asset.id)}
                                        aria-label={$_("similar-assets-set-target")}
                                        title={$_("similar-assets-set-target")}
                                    >
                                        <i class="fa fa-thumbtack"></i>
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

<ConfirmModal
    bind:this={confirmModal}
    id="asset-merge-confirm-modal"
    title={$_("asset-merge-confirm-title")}
    text=""
    action="asset-merge-confirm-action"
    onconfirm={mergePendingGroup}
>
    <p>
        {$_("asset-merge-confirm-text", {
            values: { n: sourceCount(pendingMergeGroup) },
        })}
    </p>
</ConfirmModal>
