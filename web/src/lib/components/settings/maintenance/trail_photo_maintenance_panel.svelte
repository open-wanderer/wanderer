<script lang="ts">
    import TrailListItem from "$lib/components/trail/trail_list_item.svelte";
    import type { Trail } from "$lib/models/trail";
    import { _ } from "svelte-i18n";

    type ItemStatus = "pending" | "imported" | "empty" | "failed";

    interface TrailPhotoMaintenanceCandidate {
        id: string;
        name: string;
        location?: string;
        date?: string;
        created?: string;
        updated?: string;
        completed: boolean;
        public?: boolean;
        distance?: number;
        elevation_gain?: number;
        elevation_loss?: number;
        duration?: number;
        difficulty?: "easy" | "moderate" | "difficult";
        thumbnail?: string;
    }

    interface TrailPhotoMaintenanceListResponse {
        assetPluginActive: boolean;
        trails: TrailPhotoMaintenanceCandidate[];
    }

    interface TrailPhotoMaintenanceAttachResponse {
        ok: boolean;
        trailId: string;
        imported: number;
        plugins: {
            pluginId: string;
            instanceId: string;
            imported: number;
            error?: string;
        }[];
    }

    interface TrailPhotoMaintenanceItem {
        id: string;
        trail: TrailPhotoMaintenanceCandidate;
        trailView: Trail;
        selected: boolean;
        status: ItemStatus;
        imported: number;
        error?: string;
    }

    let hasScanned = $state(false);
    let loading = $state(false);
    let attaching = $state(false);
    let assetPluginActive = $state(true);
    let items = $state<TrailPhotoMaintenanceItem[]>([]);
    let error = $state("");
    let total = $state(0);
    let processed = $state(0);
    let imported = $state(0);
    let empty = $state(0);
    let failed = $state(0);
    let hoveredTrailId = $state<string | null>(null);

    let attachableCount = $derived(
        items.filter((item) => item.selected && item.status === "pending").length,
    );
    let selectableCount = $derived(items.filter((item) => item.status === "pending").length);
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
            const response = await fetchCandidates();
            assetPluginActive = response.assetPluginActive;
            items = response.trails.map((trail) => ({
                id: trail.id,
                trail,
                trailView: trailToViewModel(trail),
                selected: true,
                status: "pending",
                imported: 0,
            }));
        } catch (e) {
            console.error("Failed to load trail photo maintenance candidates", e);
            error = e instanceof Error ? e.message : $_("trail-photo-maintenance-error");
        } finally {
            loading = false;
        }
    }

    async function attachSelected() {
        const selectedItems = items.filter(
            (item) => item.selected && item.status === "pending",
        );
        if (!selectedItems.length || attaching) {
            return;
        }

        attaching = true;
        error = "";
        resetProgress();
        total = selectedItems.length;

        for (const item of selectedItems) {
            try {
                const response = await attachTrail(item.trail.id);
                const pluginError = firstPluginError(response);
                if (pluginError && response.imported <= 0) {
                    throw new Error(pluginError);
                }

                const status: ItemStatus = response.imported > 0 ? "imported" : "empty";
                updateItem(item.id, {
                    selected: false,
                    status,
                    imported: response.imported,
                    error: pluginError,
                });
                if (response.imported > 0) {
                    imported++;
                } else {
                    empty++;
                }
            } catch (e) {
                console.error("Failed to attach trail photos", item.trail.id, e);
                updateItem(item.id, {
                    status: "failed",
                    error: e instanceof Error ? e.message : $_("trail-photo-maintenance-error"),
                });
                failed++;
            } finally {
                processed++;
            }
        }

        attaching = false;
    }

    async function fetchCandidates(): Promise<TrailPhotoMaintenanceListResponse> {
        const response = await fetch("/api/v1/plugins/assets/maintenance/trails");
        if (!response.ok) {
            throw new Error(await responseText(response));
        }
        return response.json();
    }

    async function attachTrail(trailId: string): Promise<TrailPhotoMaintenanceAttachResponse> {
        const response = await fetch("/api/v1/plugins/assets/maintenance/attach", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ trailId }),
        });
        if (!response.ok) {
            throw new Error(await responseText(response));
        }
        return response.json();
    }

    function updateItem(id: string, values: Partial<TrailPhotoMaintenanceItem>) {
        items = items.map((item) => (item.id === id ? { ...item, ...values } : item));
    }

    function toggleAll(event: Event) {
        const checked = (event.currentTarget as HTMLInputElement).checked;
        items = items.map((item) =>
            item.status === "pending" ? { ...item, selected: checked } : item,
        );
    }

    function firstPluginError(response: TrailPhotoMaintenanceAttachResponse) {
        return response.plugins.find((plugin) => plugin.error)?.error ?? "";
    }

    function trailToViewModel(candidate: TrailPhotoMaintenanceCandidate) {
        const trail: Trail = {
            collectionId: "trails",
            collectionName: "trails",
            id: candidate.id,
            name: candidate.name,
            location: candidate.location ?? "",
            date: candidate.date,
            public: candidate.public ?? false,
            completed: candidate.completed,
            distance: candidate.distance ?? 0,
            elevation_gain: candidate.elevation_gain ?? 0,
            elevation_loss: candidate.elevation_loss ?? 0,
            duration: candidate.duration ?? 0,
            difficulty: candidate.difficulty ?? "easy",
            photos: candidate.thumbnail ? [candidate.thumbnail] : [],
            thumbnail: 0,
            tags: [],
            category: "",
            subcategory: "",
            created: candidate.created,
            updated: candidate.updated,
            description: "",
            author: "",
            like_count: 0,
            expand: {},
        } as Trail;
        return trail;
    }

    function toggleItemSelection(id: string) {
        const item = items.find((candidate) => candidate.id === id);
        if (!item || item.status !== "pending") {
            return;
        }
        updateItem(id, { selected: !item.selected });
    }

    function itemResultText(item: TrailPhotoMaintenanceItem) {
        if (item.status === "imported") {
            return $_("trail-photo-maintenance-imported-count", {
                values: { n: item.imported },
            });
        }
        if (item.status === "empty") {
            return $_("trail-photo-maintenance-status-empty");
        }
        if (item.status === "failed") {
            return $_("trail-photo-maintenance-status-failed");
        }
        return "";
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
        imported = 0;
        empty = 0;
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
                    disabled={attaching || selectableCount === 0}
                    onchange={toggleAll}
                />
                {$_("trail-photo-maintenance-select-all")}
            </label>
        {:else}
            <span></span>
        {/if}

        <div class="flex flex-wrap gap-2 sm:justify-end">
            <button class="btn-secondary shrink-0" onclick={loadCandidates} disabled={loading || attaching}>
                <i class="fa fa-magnifying-glass mr-2"></i>
                {$_("trail-photo-maintenance-scan")}
            </button>
            <button
                class="btn-primary shrink-0"
                onclick={attachSelected}
                disabled={loading || attaching || attachableCount === 0}
            >
                <i class="fa fa-images mr-2"></i>
                {attaching ? $_("loading") : $_("trail-photo-maintenance-attach")}
            </button>
        </div>
    </div>

    {#if !hasScanned}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {$_("trail-photo-maintenance-ready")}
        </div>
    {:else if loading}
        <div class="flex items-center gap-3 rounded-lg border border-input-border p-6">
            <div class="spinner light:spinner-dark"></div>
            <p class="text-sm text-gray-500">{$_("trail-photo-maintenance-loading")}</p>
        </div>
    {:else if error}
        <div class="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300">
            {error}
        </div>
    {:else if !assetPluginActive}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {$_("trail-photo-maintenance-no-plugin")}
        </div>
    {:else}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {#if items.length === 0}
                {$_("trail-photo-maintenance-empty")}
            {:else}
                {$_("trail-photo-maintenance-candidates", {
                    values: { n: items.length },
                })}
            {/if}
        </div>
    {/if}

    {#if total > 0}
        <div class="rounded-lg border border-input-border p-4 text-sm text-gray-500">
            {$_("trail-photo-maintenance-progress", {
                values: { processed, total, imported, empty, failed },
            })}
        </div>
    {/if}

    {#if items.length > 0}
        <div class="space-y-3" role="list">
            {#each items as item (item.id)}
                <article class="space-y-2">
                    <a
                        class="block max-w-full"
                        href={`/trail/edit/${item.trail.id}`}
                        onmouseenter={() => (hoveredTrailId = item.id)}
                        onmouseleave={() => (hoveredTrailId = null)}
                    >
                        <TrailListItem
                            trail={item.trailView}
                            selected={item.status === "pending" && item.selected}
                            hovered={item.status === "pending" && hoveredTrailId === item.id}
                            showDescription={false}
                            onTrailSelect={() => toggleItemSelection(item.id)}
                        />
                    </a>

                    {#if item.status !== "pending" || item.error}
                        <div class="flex flex-wrap items-center gap-2 pl-1 text-xs text-gray-500">
                            {#if itemResultText(item)}
                                <span>{itemResultText(item)}</span>
                            {/if}
                            {#if item.error}
                                <span class="text-red-700 dark:text-red-300">{item.error}</span>
                            {/if}
                        </div>
                    {/if}
                </article>
            {/each}
        </div>
    {/if}
</div>
