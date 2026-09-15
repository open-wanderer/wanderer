<script lang="ts">
    import type { Asset } from "$lib/models/asset";
    import { asset_photo_url } from "$lib/stores/asset_store";
    import { extractPhotoExifMetadata, type PhotoExifMetadata } from "$lib/util/exif_util";
    import { _ } from "svelte-i18n";
    import Datepicker from "../../base/datepicker.svelte";
    import TextField from "../../base/text_field.svelte";

    type ItemStatus = "pending" | "updated" | "skipped" | "failed";

    interface PhotoMetadataItem {
        id: string;
        asset: Asset;
        selected: boolean;
        status: ItemStatus;
        metadata: PhotoExifMetadata;
        latInput: string;
        lonInput: string;
        takenAtInput: string;
        scanError?: string;
        error?: string;
    }

    let hasScanned = $state(false);
    let loading = $state(false);
    let repairing = $state(false);
    let items = $state<PhotoMetadataItem[]>([]);
    let error = $state("");
    let total = $state(0);
    let processed = $state(0);
    let updated = $state(0);
    let skipped = $state(0);
    let failed = $state(0);
    let repairableCount = $derived(
        items.filter((item) => item.selected && item.status !== "updated" && item.status !== "skipped").length,
    );

    async function loadCandidates() {
        hasScanned = true;
        loading = true;
        error = "";
        resetProgress();

        try {
            const candidates = await fetchCandidates();
            items = await Promise.all(candidates.map(buildItem));
        } catch (e) {
            console.error("Failed to load photo metadata candidates", e);
            error = e instanceof Error ? e.message : $_("photo-metadata-maintenance-error");
        } finally {
            loading = false;
        }
    }

    async function repairCandidates() {
        const selectedItems = items.filter((item) => item.selected && item.status !== "updated" && item.status !== "skipped");
        if (!selectedItems.length || repairing) {
            return;
        }

        repairing = true;
        error = "";
        resetProgress();
        total = selectedItems.length;

        for (const item of selectedItems) {
            try {
                const patch = metadataPatch(item);
                if (!Object.keys(patch).length) {
                    if (item.scanError) {
                        throw new Error(item.scanError);
                    }
                    await markAssetChecked(item.asset);
                    setItemState(item.id, { selected: false, status: "skipped", error: "" });
                    skipped++;
                    continue;
                }

                const response = await fetch(`/api/v1/assets/${item.asset.id}`, {
                    method: "PATCH",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(patch),
                });
                if (!response.ok) {
                    throw new Error(await responseText(response));
                }
                setItemState(item.id, { selected: false, status: "updated", error: "" });
                updated++;
            } catch (e) {
                console.error("Failed to repair photo metadata", item.asset.id, e);
                setItemState(item.id, {
                    status: "failed",
                    error: e instanceof Error ? e.message : $_("photo-metadata-maintenance-error"),
                });
                failed++;
            } finally {
                processed++;
            }
        }

        repairing = false;
    }

    async function fetchCandidates(): Promise<Asset[]> {
        const response = await fetch("/api/v1/assets/metadata");
        if (!response.ok) {
            throw new Error(await responseText(response));
        }
        return response.json();
    }

    async function buildItem(asset: Asset): Promise<PhotoMetadataItem> {
        let metadata: PhotoExifMetadata = {};
        let scanError: string | undefined;
        try {
            metadata = await readAssetExifMetadata(asset);
        } catch (e) {
            scanError = e instanceof Error ? e.message : $_("photo-metadata-maintenance-error");
        }

        return {
            id: asset.id,
            asset,
            selected: true,
            status: "pending",
            metadata,
            latInput: metadata.coordinates ? formatCoordinate(metadata.coordinates.lat) : "",
            lonInput: metadata.coordinates ? formatCoordinate(metadata.coordinates.lon) : "",
            takenAtInput: dateInputValue(metadata.takenAt),
            scanError,
        };
    }

    async function readAssetExifMetadata(asset: Asset): Promise<PhotoExifMetadata> {
        const response = await fetch(asset_photo_url(asset));
        if (!response.ok) {
            throw new Error(await responseText(response));
        }
        return extractPhotoExifMetadata(await response.blob());
    }

    function metadataPatch(item: PhotoMetadataItem) {
        const patch: {
            lat?: number;
            lon?: number;
            takenAt?: string;
            replaceCoordinates?: boolean;
            markChecked?: boolean;
        } = {};
        const coordinates = coordinatesFromItem(item);
        const canReplaceCoordinates = isLegacyMigratedAsset(item.asset);
        if (
            (!assetHasCoordinates(item.asset) || canReplaceCoordinates) &&
            coordinates
        ) {
            patch.lat = coordinates.lat;
            patch.lon = coordinates.lon;
            patch.replaceCoordinates = canReplaceCoordinates;
        }

        const takenAt = isoFromDateInput(item.takenAtInput);
        if (!item.asset.taken_at && takenAt) {
            patch.takenAt = takenAt;
        }

        if (Object.keys(patch).length) {
            patch.markChecked = true;
        }
        return patch;
    }

    async function markAssetChecked(asset: Asset) {
        const response = await fetch(`/api/v1/assets/${asset.id}`, {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ markChecked: true }),
        });
        if (!response.ok) {
            throw new Error(await responseText(response));
        }
    }

    function updateItem(id: string, values: Partial<PhotoMetadataItem>) {
        items = items.map((item) => (item.id === id ? { ...item, ...values } : item));
    }

    function setItemState(id: string, values: Partial<PhotoMetadataItem>) {
        updateItem(id, values);
    }

    function coordinatesFromItem(item: PhotoMetadataItem) {
        const lat = numberInputValue(item.latInput);
        const lon = numberInputValue(item.lonInput);
        if (lat === undefined || lon === undefined) {
            return undefined;
        }
        return { lat, lon };
    }

    function numberInputValue(value: string): number | undefined {
        const trimmed = value.trim();
        if (trimmed === "") {
            return undefined;
        }
        const parsed = Number(trimmed);
        return Number.isFinite(parsed) ? parsed : undefined;
    }

    function isLegacyMigratedAsset(asset: Asset): boolean {
        return typeof asset.metadata?.source_collection === "string";
    }

    function assetHasCoordinates(asset: Asset): asset is Asset & { lat: number; lon: number } {
        return (
            typeof asset.lat === "number" &&
            typeof asset.lon === "number" &&
            Number.isFinite(asset.lat) &&
            Number.isFinite(asset.lon) &&
            (asset.lat !== 0 || asset.lon !== 0)
        );
    }

    function canEditCoordinates(item: PhotoMetadataItem): boolean {
        return !assetHasCoordinates(item.asset) || isLegacyMigratedAsset(item.asset);
    }

    function canEditTakenAt(item: PhotoMetadataItem): boolean {
        return !item.asset.taken_at;
    }

    function currentTakenAtText(asset: Asset): string {
        return asset.taken_at ? formatDateTime(asset.taken_at) : $_("photo-metadata-maintenance-missing");
    }

    function currentCoordinatesText(asset: Asset): string {
        if (!assetHasCoordinates(asset)) {
            return $_("photo-metadata-maintenance-missing");
        }
        return `${formatCoordinate(asset.lat)}, ${formatCoordinate(asset.lon)}`;
    }

    function statusLabel(item: PhotoMetadataItem): string {
        if (item.scanError && item.status === "pending") {
            return $_("photo-metadata-maintenance-status-no-exif");
        }
        return $_(`photo-metadata-maintenance-status-${item.status}`);
    }

    function fileLabel(asset: Asset): string {
        const sourceFile = typeof asset.metadata?.source_file === "string" ? asset.metadata.source_file : "";
        return sourceFile || asset.file || asset.id;
    }

    function formatCoordinate(value: number): string {
        return value.toFixed(6);
    }

    function formatDateTime(value: string): string {
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value;
        }
        return date.toLocaleString();
    }

    function dateInputValue(value?: string): string {
        if (!value) {
            return "";
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return "";
        }
        const pad = (part: number) => part.toString().padStart(2, "0");
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
    }

    function isoFromDateInput(value: string): string | undefined {
        if (!value.trim()) {
            return undefined;
        }
        const date = new Date(`${value}T00:00:00`);
        return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
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
        updated = 0;
        skipped = 0;
        failed = 0;
    }
</script>

<div class="space-y-6">
    <div class="flex justify-end">
        <div class="flex flex-wrap gap-2">
            <button class="btn-secondary shrink-0" onclick={loadCandidates} disabled={loading || repairing}>
                <i class="fa fa-magnifying-glass mr-2"></i>
                {$_("photo-metadata-maintenance-scan")}
            </button>
            <button
                class="btn-primary shrink-0"
                onclick={repairCandidates}
                disabled={loading || repairing || repairableCount === 0}
            >
                <i class="fa fa-wand-magic-sparkles mr-2"></i>
                {repairing ? $_("loading") : $_("photo-metadata-maintenance-repair")}
            </button>
        </div>
    </div>

    {#if !hasScanned}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {$_("photo-metadata-maintenance-ready")}
        </div>
    {:else if loading}
        <div class="flex items-center gap-3 rounded-lg border border-input-border p-6">
            <div class="spinner light:spinner-dark"></div>
            <p class="text-sm text-gray-500">{$_("photo-metadata-maintenance-loading")}</p>
        </div>
    {:else if error}
        <div class="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300">
            {error}
        </div>
    {:else}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {#if items.length === 0}
                {$_("photo-metadata-maintenance-empty")}
            {:else}
                {$_("photo-metadata-maintenance-candidates", {
                    values: { n: items.length },
                })}
            {/if}
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
                    <div class="min-w-0 space-y-4">
                        <div class="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
                            <div class="min-w-0 space-y-1">
                                <div class="flex min-w-0 flex-wrap items-center gap-2">
                                    <h3 class="truncate text-sm font-semibold">{fileLabel(item.asset)}</h3>
                                    {#if isLegacyMigratedAsset(item.asset)}
                                        <span class="rounded-full bg-menu-item-background-hover px-2 py-0.5 text-xs text-gray-500">
                                            {$_("photo-metadata-maintenance-legacy")}
                                        </span>
                                    {/if}
                                    <span class="rounded-full bg-menu-item-background-hover px-2 py-0.5 text-xs text-gray-500">
                                        {statusLabel(item)}
                                    </span>
                                </div>
                                {#if item.scanError && item.status !== "failed"}
                                    <p class="text-xs text-gray-500">{item.scanError}</p>
                                {/if}
                                {#if item.error}
                                    <p class="text-xs text-red-300">{item.error}</p>
                                {/if}
                            </div>
                            <label class="flex items-center gap-2 text-sm text-gray-500">
                                <input
                                    type="checkbox"
                                    checked={item.selected}
                                    disabled={item.status === "updated" || item.status === "skipped"}
                                    onchange={(e) => updateItem(item.id, { selected: (e.currentTarget as HTMLInputElement).checked })}
                                />
                                {$_("photo-metadata-maintenance-apply")}
                            </label>
                        </div>

                        <div class="grid gap-4 lg:grid-cols-2">
                            <div class="space-y-2">
                                <div class="text-xs text-gray-500">
                                    {$_("photo-metadata-maintenance-current")}: {currentTakenAtText(item.asset)}
                                </div>
                                <Datepicker
                                    name={`${item.id}-taken-at`}
                                    label={$_("photo-metadata-maintenance-taken-at")}
                                    value={item.takenAtInput}
                                    onchange={(e) => updateItem(item.id, { takenAtInput: (e.currentTarget as HTMLInputElement).value })}
                                    disabled={!canEditTakenAt(item)}
                                />
                            </div>

                            <div class="space-y-2">
                                <div class="text-xs text-gray-500">
                                    {$_("photo-metadata-maintenance-current")}: {currentCoordinatesText(item.asset)}
                                </div>
                                <div class="grid grid-cols-2 gap-2">
                                    <TextField
                                        name={`${item.id}-lat`}
                                        label={$_("photo-metadata-maintenance-lat")}
                                        type="number"
                                        step="any"
                                        placeholder={$_("photo-metadata-maintenance-lat")}
                                        value={item.latInput}
                                        disabled={!canEditCoordinates(item)}
                                        oninput={(e) => updateItem(item.id, { latInput: (e.currentTarget as HTMLInputElement).value })}
                                    />
                                    <TextField
                                        name={`${item.id}-lon`}
                                        label={$_("photo-metadata-maintenance-lon")}
                                        type="number"
                                        step="any"
                                        placeholder={$_("photo-metadata-maintenance-lon")}
                                        value={item.lonInput}
                                        disabled={!canEditCoordinates(item)}
                                        oninput={(e) => updateItem(item.id, { lonInput: (e.currentTarget as HTMLInputElement).value })}
                                    />
                                </div>
                            </div>
                        </div>
                    </div>
                </article>
            {/each}
        </div>
    {/if}

    {#if repairing || processed > 0}
        <div class="rounded-lg border border-input-border p-6 text-sm text-gray-500">
            {$_("photo-metadata-maintenance-progress", {
                values: {
                    processed,
                    total,
                    updated,
                    skipped,
                    failed,
                },
            })}
        </div>
    {/if}
</div>
