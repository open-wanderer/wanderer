<script lang="ts">
    import { goto } from "$app/navigation";
    import { browser } from "$app/environment";
    import { page } from "$app/state";
    import ActorSearch from "$lib/components/actor_search.svelte";
    import type { DropdownItem } from "$lib/components/base/dropdown.svelte";
    import Search, {
        type SearchItem,
    } from "$lib/components/base/search.svelte";
    import type { SelectItem } from "$lib/components/base/select.svelte";
    import Select from "$lib/components/base/select.svelte";
    import SkeletonCard from "$lib/components/base/skeleton_card.svelte";
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import EmptyStateSearch from "$lib/components/empty_states/empty_state_search.svelte";
    import ListCard from "$lib/components/list/list_card.svelte";
    import ListPanel from "$lib/components/list/list_panel.svelte";
    import ListShareModal from "$lib/components/list/list_share_modal.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import TrailInfoPanel from "$lib/components/trail/trail_info_panel.svelte";
    import { List, type ListFilter } from "$lib/models/list";
    import { Trail } from "$lib/models/trail";
    import {
        lists_delete,
        lists_map_preview,
        lists_search_filter,
        lists_show,
    } from "$lib/stores/list_store";
    import { trails_get_bounding_box, trails_show } from "$lib/stores/trail_store";
    import { currentUser } from "$lib/stores/user_store";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util.js";
    import type { PreviewListBounds } from "$lib/util/list_map_preview_util";
    import * as M from "maplibre-gl";

    import { onMount, untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import { slide } from "svelte/transition";

    let { data } = $props();

    const sortOptions: SelectItem[] = [
        { text: $_("alphabetical"), value: "name" },
        { text: $_("creation-date"), value: "created" },
    ];

    const pagination: { page: number; totalPages: number } = $state({
        page: 1,
        totalPages: untrack(() => data.lists.totalPages),
    });

    let lists: List[] = $state(untrack(() => data.lists.items));

    let confirmModal: ConfirmModal | undefined = $state();
    let listShareModal: ListShareModal | undefined = $state();

    let filter: ListFilter = $state(page.data.filter);

    let map: M.Map | undefined = $state();
    let mapWithElevation: MapWithElevationMaplibre | undefined = $state();
    let markers: M.Marker[] = $state([]);

    let selectedList: List | null = $state(
        untrack(() =>
            page.params.handle && page.params.id ? data.lists.items[0] : null,
        ),
    );
    let selectedTrail: Trail | null = $state(null);

    let loading: boolean = $state(true);
    let loadingNextPage: boolean = false;
    let filterExpanded: boolean = $state(false);

    let loadAllListsOnNextBack = false;

    let userQuery = $state("");
    let overviewFitKey = $state("");
    let overviewTrails: Trail[] = $state([]);
    let overviewPreviewKey = $state("");
    let overviewPreviewRequest = 0;

    let selectedTrailIndex = $derived(selectedTrail ? 0 : null);

    let selectedTrailWaypoints = $derived(
        (selectedTrail as Trail | null)?.expand?.waypoints_via_trail,
    );

    let mapTrails = $derived(
        selectedTrail
            ? [selectedTrail]
            : (selectedList?.expand?.trails ?? overviewTrails),
    );

    function applyBounds(trail: Trail, bounds?: PreviewListBounds) {
        if (!bounds) {
            return;
        }
        trail.min_lat = bounds.min_lat;
        trail.max_lat = bounds.max_lat;
        trail.min_lon = bounds.min_lon;
        trail.max_lon = bounds.max_lon;
    }

    function listIdFromOverviewTrail(trail: Trail) {
        return trail.id?.split("#")[0] ?? trail.id;
    }

    async function refreshOverviewPreview(sourceLists: List[]) {
        const listIds = sourceLists
            .map((item) => item.id)
            .filter((id): id is string => !!id);
        const key = listIds.join(",");
        if (key === overviewPreviewKey && overviewTrails.length > 0) {
            return;
        }
        overviewPreviewKey = key;
        const request = ++overviewPreviewRequest;

        if (listIds.length === 0) {
            overviewTrails = [];
            return;
        }

        try {
            const preview = await lists_map_preview(listIds);
            if (request !== overviewPreviewRequest) {
                return;
            }
            const nextTrails: Trail[] = [];

            for (const listPreview of preview.lists) {
                const list = sourceLists.find((item) => item.id === listPreview.id);
                if (!list) {
                    continue;
                }

                if (listPreview.trails.length === 0) {
                    continue;
                }

                for (const geometry of listPreview.trails) {
                    const hasPoint =
                        geometry.lat != null &&
                        geometry.lon != null &&
                        !(geometry.lat === 0 && geometry.lon === 0);
                    if (!geometry.polyline && !hasPoint) {
                        continue;
                    }

                    const trail = new Trail(list.name, {
                        id: `${list.id}#${geometry.id}`,
                        lat: geometry.lat,
                        lon: geometry.lon,
                    });
                    trail.polyline = geometry.polyline;
                    trail.author = list.author;
                    trail.min_lat = geometry.min_lat;
                    trail.max_lat = geometry.max_lat;
                    trail.min_lon = geometry.min_lon;
                    trail.max_lon = geometry.max_lon;
                    applyBounds(trail, listPreview.bounds);
                    nextTrails.push(trail);
                }
            }

            overviewTrails = nextTrails;
        } catch {
            if (request === overviewPreviewRequest) {
                overviewTrails = [];
            }
        }
    }

    async function fitOverviewOrFallback() {
        if (overviewTrails.length) {
            mapWithElevation?.fitToBounds();
            return;
        }
        try {
            const bbox = await trails_get_bounding_box();
            if (
                bbox.has_trails ??
                (bbox.min_lon != 0 ||
                    bbox.max_lat != 0 ||
                    bbox.max_lon != 0 ||
                    bbox.min_lat != 0)
            ) {
                map?.fitBounds(
                    [
                        [bbox.min_lon, bbox.min_lat],
                        [bbox.max_lon, bbox.max_lat],
                    ],
                    { animate: true, padding: 64, maxZoom: 12 },
                );
            }
        } catch {
            // Keep the default map view if the bounding box is unavailable.
        }
    }

    $effect(() => {
        if (!browser || selectedList || selectedTrail) {
            return;
        }
        const sourceLists = lists;
        untrack(() => {
            void refreshOverviewPreview(sourceLists);
        });
    });

    $effect(() => {
        if (!browser) {
            return;
        }
        if (selectedList || selectedTrail) {
            overviewFitKey = "";
            return;
        }
        if (!map) {
            return;
        }
        const key = [
            filter.q,
            filter.author ?? "",
            String(filter.public),
            String(filter.shared),
            filter.sort ?? "",
            filter.sortOrder ?? "",
            overviewPreviewKey,
            String(overviewTrails.length),
        ].join("|");
        if (overviewTrails.length === 0 && loading) {
            return;
        }
        if (overviewFitKey === key) {
            return;
        }
        overviewFitKey = key;
        untrack(() => {
            fitOverviewOrFallback();
        });
    });

    onMount(() => {
        if (page.params.handle && page.params.id) {
            // setCurrentList(data.lists[0]);
            // only the requested list has been loaded at this point
            // load all lists the next time the user presses the back button
            loadAllListsOnNextBack = true;
        }
        loading = false;
    });

    async function handleDropdownClick(item: DropdownItem) {
        if (!selectedList) {
            return;
        }
        if (item.value == "share") {
            listShareModal?.openModal();
        } else if (item.value == "edit") {
            goto("/lists/edit/" + selectedList.id);
        } else if (item.value == "delete") {
            confirmModal?.openModal();
        }
    }

    async function deleteList() {
        if (!selectedList) {
            return;
        }
        await lists_delete(selectedList);
        await updateFilter();
        selectedList = null;
    }

    async function setCurrentList(item: List) {
        const fullList = await lists_show(
            item.id!,
            handleFromRecordWithIRI(item),
            fetch,
        );
        selectedList = fullList;
        document.getElementById("list-container")?.scrollTo({ top: 0 });
    }

    async function back() {
        if (selectedTrail) {
            selectedTrail = null;
        } else if (selectedList) {
            selectedList = null;
        }
        if (loadAllListsOnNextBack) {
            await updateFilter(false);
            loadAllListsOnNextBack = false;
        }
    }

    async function selectTrail(trail: Trail) {
        const fullTrail = await trails_show(
            trail.iri ? trail.iri.substring(trail.iri.length - 15) : trail.id!,
            handleFromRecordWithIRI(trail),
            undefined,
            true,
        );
        selectedTrail = fullTrail;

        mapWithElevation?.unHighlightTrail(trail.id!);
        window.scrollTo({ top: 0 });
    }

    function highlightTrail(trail: Trail) {
        mapWithElevation?.highlightTrail(trail.id!);
    }

    function unHighlightTrail(trail: Trail) {
        mapWithElevation?.unHighlightTrail(trail.id!);
    }

    async function onListScroll(e: Event) {
        const container = e.target as HTMLDivElement;
        const scrollTop = container.scrollTop;
        const scrollHeight = container.scrollHeight;
        const clientHeight = container.clientHeight;

        if (
            scrollTop + clientHeight >= scrollHeight * 0.8 &&
            pagination.page !== pagination.totalPages &&
            !loadingNextPage
        ) {
            loadingNextPage = true;
            await loadNextPage();
            loadingNextPage = false;
        }
    }

    async function loadNextPage() {
        pagination.page += 1;
        const response = await lists_search_filter(filter, pagination.page);
        overviewPreviewKey = "";
        lists = response.items;
        pagination.page = response.page;
        pagination.totalPages = response.totalPages;
    }

    async function updateFilter(resetMap: boolean = true) {
        loading = true;

        if ((selectedList || selectedTrail) && resetMap) {
            selectedList = null;
            selectedTrail = null;
        }

        pagination.page = 1;
        const response = await lists_search_filter(filter, pagination.page);
        overviewPreviewKey = "";
        lists = response.items;
        pagination.page = response.page;
        pagination.totalPages = response.totalPages;
        loading = false;
    }

    async function setSort(value: "name" | "created") {
        filter.sort = value;
        await updateFilter();
    }

    async function setSortOrder() {
        if (filter.sortOrder === "+") {
            filter.sortOrder = "-";
        } else {
            filter.sortOrder = "+";
        }
        await updateFilter();
    }

    async function setPublicFilter(e: Event) {
        filter.public = (e.target as HTMLInputElement).checked;
        updateFilter();
    }

    async function setAuthorFilter(item: SearchItem) {
        filter.author = item.value.id;
        await updateFilter();
    }

    async function clearAuthorFilter() {
        filter.author = "";
        await updateFilter();
    }

    async function setSharedFilter(e: Event) {
        filter.shared = (e.target as HTMLInputElement).checked;
        updateFilter();
    }
</script>

<svelte:head>
    <title>{$_("list", { values: { n: 2 } })} | wanderer</title>
</svelte:head>
<main class="grid grid-cols-1 md:grid-cols-[430px_1fr] gap-4 lg:gap-4 md:mx-4">
    <div
        class="list-list relative md:mx-auto rounded-xl border border-input-border max-h-full w-full order-1 md:order-none"
    >
        <div
            class="flex gap-x-3 items-center px-3 py-4 bg-background z-50 rounded-xl"
        >
            <button
                aria-label="Back"
                class="btn-icon"
                class:btn-disabled={!selectedList}
                disabled={!selectedList}
                onclick={back}><i class="fa fa-arrow-left"></i></button
            >
            <Search bind:value={filter.q} onupdate={() => updateFilter()}
            ></Search>
            <button
                aria-label="Toggle filter"
                class="btn-icon"
                onclick={() => (filterExpanded = !filterExpanded)}
                ><i class="fa fa-sliders"></i></button
            >
            {#if $currentUser}
                <a
                    aria-label="New list"
                    class="btn-primary tooltip"
                    data-title={$_("new-list")}
                    href="/lists/edit/new"><i class="fa fa-plus"></i></a
                >
            {/if}
        </div>
        {#if filterExpanded}
            <div
                class="absolute bg-background z-10 shadow-lg border-b border-input-border rounded-b-xl w-full px-14"
                in:slide
                out:slide
            >
                <p class="text-sm font-medium pb-1">{$_("sort")}</p>
                <div class="flex items-center gap-2" class:mb-6={!$currentUser}>
                    <Select
                        bind:value={filter.sort}
                        items={sortOptions}
                        onchange={setSort}
                    ></Select>
                    <button
                        aria-label="Set sort order"
                        id="sort-order-btn"
                        class="btn-icon"
                        class:rotated={filter.sortOrder == "-"}
                        onclick={() => setSortOrder()}
                        ><i class="fa fa-arrow-up"></i></button
                    >
                </div>
                {#if $currentUser}
                    <hr class="my-4 border-separator" />

                    <ActorSearch
                        onclick={setAuthorFilter}
                        onclear={clearAuthorFilter}
                        bind:value={userQuery}
                        clearAfterSelect={false}
                        label={$_("author")}
                    ></ActorSearch>
                    <div class="flex items-center my-4">
                        <input
                            id="public-checkbox"
                            type="checkbox"
                            checked={filter.public}
                            class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                            onchange={setPublicFilter}
                        />
                        <label for="public-checkbox" class="ms-2 text-sm"
                            >{$_("public")}</label
                        >
                    </div>
                    <div class="flex items-center my-4">
                        <input
                            id="shared-checkbox"
                            type="checkbox"
                            checked={filter.shared}
                            class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                            onchange={setSharedFilter}
                        />
                        <label for="shared-checkbox" class="ms-2 text-sm"
                            >{$_("shared")}</label
                        >
                    </div>
                {/if}
            </div>
        {/if}
        <hr class="border-separator" />

        <div
            id="list-container"
            class="overflow-y-scroll overflow-x-clip max-h-full"
            onscroll={onListScroll}
        >
            {#if !selectedList}
                <div class="px-4 mt-2" class:space-y-3={loading}>
                    {#if loading}
                        {#each { length: 3 } as _, index}
                            <SkeletonCard></SkeletonCard>
                        {/each}
                    {:else if lists.length == 0}
                        <EmptyStateSearch width={356}></EmptyStateSearch>
                    {:else}
                        {#each lists as item, i}
                            <div
                                class="list-list-item"
                                onclick={() => setCurrentList(item)}
                                role="presentation"
                            >
                                <ListCard list={item}></ListCard>
                            </div>
                        {/each}
                    {/if}
                </div>
            {:else if selectedList && !selectedTrail}
                <ListPanel
                    list={selectedList}
                    onmouseenter={(data) => highlightTrail(data.trail)}
                    onmouseleave={(data) => unHighlightTrail(data.trail)}
                    onchange={(item) => handleDropdownClick(item)}
                    onclick={(data) => selectTrail(data.trail)}
                ></ListPanel>
            {:else if selectedList && selectedTrail}
                <TrailInfoPanel
                    initTrail={selectedTrail}
                    mode="list"
                    {markers}
                    handle={handleFromRecordWithIRI(selectedTrail)}
                ></TrailInfoPanel>
            {/if}
        </div>
    </div>
    <div id="trail-map">
        <MapWithElevationMaplibre
            trails={mapTrails}
            waypoints={selectedTrailWaypoints}
            bind:map
            bind:this={mapWithElevation}
            bind:markers
            activeTrail={selectedTrailIndex}
            fitBounds={selectedList || selectedTrail ? "animate" : "off"}
            clusterTrails={!selectedList && !selectedTrail}
            onUnclusteredClick={(_, trail) => {
                const listId = listIdFromOverviewTrail(trail);
                const list = lists.find((item) => item.id === listId);
                if (list) {
                    setCurrentList(list);
                }
            }}
            onselect={(trail) => {
                if (selectedList && !selectedTrail) {
                    selectTrail(trail);
                }
            }}
            oninit={() => {
                if (!selectedList && !selectedTrail) {
                    fitOverviewOrFallback();
                }
            }}
            showInfoPopup={true}
            showTerrain={true}
        ></MapWithElevationMaplibre>
    </div>

    <ConfirmModal
        text={$_("delete-list-confirm")}
        bind:this={confirmModal}
        onconfirm={deleteList}
    ></ConfirmModal>
    {#if selectedList}
        <ListShareModal bind:this={listShareModal} list={selectedList}
        ></ListShareModal>
    {/if}
</main>

<style>
    @media only screen and (min-width: 768px) {
        #trail-map {
            height: calc(100vh - 124px);
        }

        #list-container {
            height: calc(100vh - 204px);
        }
    }
</style>
