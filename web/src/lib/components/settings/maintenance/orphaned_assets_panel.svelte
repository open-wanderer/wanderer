<script lang="ts">
    import type { Asset } from "$lib/models/asset";
    import { asset_photo_url } from "$lib/stores/asset_store";
    import { _ } from "svelte-i18n";

    type ItemStatus = "pending" | "deleted" | "skipped" | "failed";

    interface OrphanedAssetItem {
        id: string;
        asset: Asset;
        selected: boolean;
        status: ItemStatus;
        error?: string;
    }

    interface DeleteResponse {
        deleted: string[];
        skipped: string[];
        failed: { id: string; error: string }[];
    }

    let hasScanned = $state(false);
    let loading = $state(false);
    let deleting = $state(false);
    let items = $state<OrphanedAssetItem[]>([]);
    let error = $state("");
    let total = $state(0);
    let processed = $state(0);
    let deleted = $state(0);
    let skipped = $state(0);
    let failed = $state(0);

    let selectableCount = $derived(items.filter((item) => item.status === "pending").length);
    let deletableCount = $derived(
        items.filter((item) => item.selected && item.status === "pending").length,
    );
    let allSelected = $derived(
        selectableCount > 0 &&
            items.every((item) => item.status !== "pending" || item.selected),
    );

    async function loadCandidates() {
        hasScanned = true;
        loading = true;
        error = "";
        resetProgress();

        try {
            const assets = await fetchCandidates();
            items = assets.map((asset) => ({
                id: asset.id,
                asset,
                selected: true,
                status: "pending",
            }));
        } catch (e) {
            console.error("Failed to load orphaned assets", e);
            error = e instanceof Error ? e.message : $_("orphaned-assets-maintenance-error");
        } finally {
            loading = false;
        }
    }

    async function deleteSelected() {
        const selectedItems = items.filter((item) => item.selected && item.status === "pending");
        if (!selectedItems.length || deleting) {
            return;
        }

        deleting = true;
        error = "";
        resetProgress();
        total = selectedItems.length;

        try {
            const result = await deleteAssets(selectedItems.map((item) => item.id));
            const deletedIds = new Set(result.deleted);
            const skippedIds = new Set(result.skipped);
            const failures = new Map(result.failed.map((failure) => [failure.id, failure.error]));

            items = items.map((item) => {
                if (!item.selected || item.status !== "pending") {
                    return item;
                }
                if (deletedIds.has(item.id)) {
                    return { ...item, selected: false, status: "deleted", error: "" };
                }
                if (skippedIds.has(item.id)) {
                    return { ...item, selected: false, status: "skipped", error: "" };
                }
                const failure = failures.get(item.id);
                if (failure) {
                    return { ...item, status: "failed", error: failure };
                }
                return { ...item, status: "failed", error: $_("orphaned-assets-maintenance-error") };
            });

            deleted = result.deleted.length;
            skipped = result.skipped.length;
            failed = result.failed.length + Math.max(0, total - result.deleted.length - result.skipped.length - result.failed.length);
            processed = total;
        } catch (e) {
            console.error("Failed to delete orphaned assets", e);
            error = e instanceof Error ? e.message : $_("orphaned-assets-maintenance-error");
        } finally {
            deleting = false;
        }
    }

    async function fetchCandidates(): Promise<Asset[]> {
        const response = await fetch("/api/v1/assets/orphans");
        if (!response.ok) {
            throw new Error(await responseText(response));
        }
        return response.json();
    }

    async function deleteAssets(assetIds: string[]): Promise<DeleteResponse> {
        const response = await fetch("/api/v1/assets/orphans", {
            method: "DELETE",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ assetIds }),
        });
        if (!response.ok) {
            throw new Error(await responseText(response));
        }
        return response.json();
    }

    function updateItem(id: string, values: Partial<OrphanedAssetItem>) {
        items = items.map((item) => (item.id === id ? { ...item, ...values } : item));
    }

    function toggleAll(event: Event) {
        const checked = (event.currentTarget as HTMLInputElement).checked;
        items = items.map((item) =>
            item.status === "pending" ? { ...item, selected: checked } : item,
        );
    }

    function fileLabel(asset: Asset): string {
        const remote = asset.metadata?.remote as { filename?: string } | undefined;
        const sourceFile = typeof asset.metadata?.source_file === "string" ? asset.metadata.source_file : "";
        return asset.file || remote?.filename || sourceFile || asset.external_id || asset.id;
    }

    function storageLabel(asset: Asset): string {
        if (asset.storage_mode === "link_private") {
            return $_("orphaned-assets-maintenance-storage-link");
        }
        return $_("orphaned-assets-maintenance-storage-copy");
    }

    function createdLabel(asset: Asset): string {
        if (!asset.created) {
            return "";
        }
        const date = new Date(asset.created);
        return Number.isNaN(date.getTime()) ? asset.created : date.toLocaleString();
    }

    function statusLabel(item: OrphanedAssetItem): string {
        return $_(`orphaned-assets-maintenance-status-${item.status}`);
    }

    async function responseText(response: Response) {
        try {
            const data = await response.json();
            return data.message ?? response.statusText;
        } catch {
            return response.statusText;
        }
    }

    function resetProgress() {
        total = 0;
        processed = 0;
        deleted = 0;
        skipped = 0;
        failed = 0;
    }
</script>

<div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        {#if items.length > 0}
            <label class="flex items-center gap-2 text-sm text-gray-500">
                <input
                    type="checkbox"
                    class="h-4 w-4 rounded border-input-border text-primary"
                    checked={allSelected}
                    disabled={loading || deleting || selectableCount === 0}
                    onchange={toggleAll}
                />
                {$_("orphaned-assets-maintenance-select-all")}
            </label>
        {:else}
            <span></span>
        {/if}

        <div class="flex flex-wrap gap-2 sm:justify-end">
            <button class="btn-secondary shrink-0" onclick={loadCandidates} disabled={loading || deleting}>
                <i class="fa fa-magnifying-glass mr-2"></i>
                {$_("orphaned-assets-maintenance-scan")}
            </button>
            <button
                class="btn-primary shrink-0"
                onclick={deleteSelected}
                disabled={loading || deleting || deletableCount === 0}
            >
                <i class="fa fa-trash mr-2"></i>
                {deleting ? $_("loading") : $_("orphaned-assets-maintenance-delete")}
            </button>
        </div>
    </div>

    {#if !hasScanned}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {$_("orphaned-assets-maintenance-ready")}
        </div>
    {:else if loading}
        <div class="flex items-center gap-3 rounded-lg border border-input-border p-6">
            <div class="spinner light:spinner-dark"></div>
            <p class="text-sm text-gray-500">{$_("orphaned-assets-maintenance-loading")}</p>
        </div>
    {:else if error}
        <div class="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300">
            {error}
        </div>
    {:else}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {#if items.length === 0}
                {$_("orphaned-assets-maintenance-empty")}
            {:else}
                {$_("orphaned-assets-maintenance-candidates", {
                    values: { n: items.length },
                })}
            {/if}
        </div>
    {/if}

    {#if total > 0}
        <div class="rounded-lg border border-input-border p-4 text-sm text-gray-500">
            {$_("orphaned-assets-maintenance-progress", {
                values: { processed, total, deleted, skipped, failed },
            })}
        </div>
    {/if}

    {#if items.length > 0}
        <div class="divide-y divide-separator overflow-hidden rounded-lg border border-input-border">
            {#each items as item (item.id)}
                <article class="grid gap-4 p-4 md:grid-cols-[7rem_minmax(0,1fr)]">
                    <img
                        src={asset_photo_url(item.asset)}
                        alt=""
                        class="h-28 w-full rounded-lg bg-input-background object-cover md:h-24 md:w-28"
                        loading="lazy"
                    />
                    <div class="min-w-0 space-y-3">
                        <div class="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
                            <div class="min-w-0 space-y-1">
                                <div class="flex min-w-0 flex-wrap items-center gap-2">
                                    <h3 class="truncate text-sm font-semibold">{fileLabel(item.asset)}</h3>
                                    <span class="rounded-full bg-menu-item-background-hover px-2 py-0.5 text-xs text-gray-500">
                                        {storageLabel(item.asset)}
                                    </span>
                                    {#if item.asset.remote_status}
                                        <span class="rounded-full bg-menu-item-background-hover px-2 py-0.5 text-xs text-gray-500">
                                            {item.asset.remote_status}
                                        </span>
                                    {/if}
                                    <span class="rounded-full bg-menu-item-background-hover px-2 py-0.5 text-xs text-gray-500">
                                        {statusLabel(item)}
                                    </span>
                                </div>
                                <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
                                    <span>{item.asset.id}</span>
                                    {#if createdLabel(item.asset)}
                                        <span>{createdLabel(item.asset)}</span>
                                    {/if}
                                </div>
                                {#if item.error}
                                    <p class="text-xs text-red-300">{item.error}</p>
                                {/if}
                            </div>
                            <label class="flex items-center gap-2 text-sm text-gray-500">
                                <input
                                    type="checkbox"
                                    checked={item.selected}
                                    disabled={item.status !== "pending" || deleting}
                                    onchange={(e) => updateItem(item.id, { selected: (e.currentTarget as HTMLInputElement).checked })}
                                />
                                {$_("orphaned-assets-maintenance-select")}
                            </label>
                        </div>
                    </div>
                </article>
            {/each}
        </div>
    {/if}
</div>
