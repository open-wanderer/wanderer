<script lang="ts">
    import type { SummitLog } from "$lib/models/summit_log";
    import { onMount, tick } from "svelte";
    import Modal from "../base/modal.svelte";
    import SummitLogTableRow from "./summit_log_table_row.svelte";

    import type { Trail } from "$lib/models/trail";
    import { gpx2trail } from "$lib/util/gpx_util";
    import * as M from "maplibre-gl";
    import { _ } from "svelte-i18n";
    import MapWithElevationMaplibre from "../trail/map_with_elevation_maplibre.svelte";
    import { fetchGPX } from "$lib/stores/trail_store";
    import { currentUser } from "$lib/stores/user_store";

    interface Props {
        summitLogs?: SummitLog[];
        handle: string;
        showCategory?: boolean;
        showTrail?: boolean;
        showAuthor?: boolean;
        showRoute?: boolean;
        showPhotos?: boolean;
        showMenu?: boolean;
        ondelete?: (summitLog: SummitLog) => void;
        onedit?: (summitLog: SummitLog) => void;
    }

    let {
        summitLogs = [],
        handle,
        showCategory = false,
        showTrail = false,
        showAuthor = false,
        showRoute = false,
        showPhotos = false,
        showMenu = false,
        ondelete,
        onedit,
    }: Props = $props();

    let mapModal: Modal;
    let textModal: Modal;

    let map: M.Map | undefined = $state();

    let trail: Trail | null = $state(null);

    let currentText: string = $state("");

    onMount(async () => {});

    async function openMap(log: SummitLog) {
        if (!log.expand?.gpx_data) {
            const gpxData: string = await fetchGPX(log as any, fetch);
            log.expand = {
                ...(log.expand ?? {}),
                gpx_data: gpxData,
            };
        }

        trail = (await gpx2trail(log.expand.gpx_data!)).trail;
        trail.id = log.id;
        trail.expand!.gpx_data = log.expand.gpx_data;

        mapModal.openModal();
        await tick();
        return;
    }

    async function openText(log: SummitLog) {
        currentText = log.text ?? "";
        textModal.openModal();
    }
</script>

<table class="w-full table-auto">
    <thead class="text-left text-gray-500">
        <tr class="text-sm">
            {#if showPhotos && summitLogs.some((l) => l.photos.length)}
                <th class="w-24"></th>
            {/if}
            <th>{$_("date")}</th>
            <th>{$_("distance")}</th>
            <th>{$_("elevation-gain")}</th>
            <th>{$_("elevation-loss")}</th>
            <th>{$_("duration")}</th>
            {#if showCategory}
                <th>
                    {$_("category")}
                </th>
            {/if}
            {#if showTrail}
                <th>
                    {$_("trail", { values: { n: 1 } })}
                </th>
            {/if}
            {#if summitLogs.some((l) => l.text?.length)}
                <th>{$_("description")}</th>
            {/if}
            {#if showAuthor && summitLogs.some((l) => l.expand?.author)}
                <th>
                    {$_("author", { values: { n: 1 } })}
                </th>
            {/if}
            {#if showRoute && summitLogs.some((l) => l.gpx)}
                <th>
                    {$_("map")}
                </th>
            {/if}
            {#if showMenu}
                <th> </th>
            {/if}
        </tr>
    </thead>
    <tbody>
        {#each summitLogs as log, i}
            <SummitLogTableRow
                {log}
                {handle}
                onopen={(log) => openMap(log)}
                ontext={(log) => openText(log)}
                {showCategory}
                {showTrail}
                {showAuthor}
                showPhotos={showPhotos &&
                    summitLogs.some((l) => l.photos.length)}
                showDescription={summitLogs.some((l) => l.text?.length)}
                showRoute={showRoute && summitLogs.some((l) => l.gpx)}
                showMenu={showMenu && log.author == $currentUser?.actor}
                {ondelete}
                {onedit}
            ></SummitLogTableRow>
        {/each}
    </tbody>
</table>
{#if !summitLogs.length}
    <p class="text-center w-full my-8 text-gray-500 text-sm">{$_("no-data")}</p>
{/if}
<Modal
    id="summit-log-table-map-modal"
    size="md:min-w-2xl lg:min-w-4xl"
    title=""
    bind:this={mapModal}
>
    {#snippet content()}
        <div id="summit-log-table-map" class="h-[32rem]">
            {#if trail}
                <MapWithElevationMaplibre trails={[trail]} bind:map showTerrain
                ></MapWithElevationMaplibre>
            {/if}
        </div>
    {/snippet}
</Modal>
<Modal
    id="summit-log-table-text-modal"
    size="md:min-w-xl"
    title={$_("description")}
    bind:this={textModal}
>
    {#snippet content()}
        <div class="prose dark:prose-invert">{@html currentText}</div>
    {/snippet}
</Modal>

<style>
    th {
        padding: 0.5rem 0.5rem;
    }
</style>
