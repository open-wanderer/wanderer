<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import type { DropdownItem } from "$lib/components/base/dropdown.svelte";
    import SkeletonCard from "$lib/components/base/skeleton_card.svelte";
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import EmptyStateSearch from "$lib/components/empty_states/empty_state_search.svelte";
    import ListCard from "$lib/components/list/list_card.svelte";
    import ListSearchBar from "$lib/components/list/list_search_bar.svelte";
    import ListPanel from "$lib/components/list/list_panel.svelte";
    import ListShareModal from "$lib/components/list/list_share_modal.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import TrailInfoPanel from "$lib/components/trail/trail_info_panel.svelte";
    import { List, type ListFilter } from "$lib/models/list";
    import type { Trail } from "$lib/models/trail";
    import {
        lists_delete,
        lists_search_filter,
        lists_show,
    } from "$lib/stores/list_store";
    import { trails_show } from "$lib/stores/trail_store";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util.js";
    import * as M from "maplibre-gl";

    import { onMount, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    const pagination: { page: number; totalPages: number } = $state({
        page: 1,
        totalPages: untrack(() => data.lists.totalPages),
    });

    let lists: List[] = $state(untrack(() => data.lists.items));

    let confirmModal: ConfirmModal | undefined = $state();
    let listShareModal: ListShareModal | undefined = $state();

    let filter: ListFilter = $state(page.data.filter);

    let listsFullWidth = $state(true);
    let resultsElement: HTMLDivElement;
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
    let loadingNextPage: boolean = $state(false);
    let refreshing = $state(false);
    let searchRevision = 0;
    let searchController: AbortController | undefined;

    let loadAllListsOnNextBack = false;

    let selectedTrailIndex = $derived(selectedTrail ? 0 : null);

    let selectedTrailWaypoints = $derived(
        (selectedTrail as Trail | null)?.expand?.waypoints_via_trail,
    );

    onMount(() => {
        listsFullWidth = localStorage.getItem("listsFullWidth") !== "false";
        void tick().then(() => map?.resize());
        if (page.params.handle && page.params.id) {
            // setCurrentList(data.lists[0]);
            // only the requested list has been loaded at this point
            // load all lists the next time the user presses the back button
            loadAllListsOnNextBack = true;
        }
        loading = false;
        resizeLayout();
        const searchSize = new ResizeObserver(resizeLayout);
        const search = resultsElement.closest("main")?.querySelector("[data-search-root]");
        if (search) searchSize.observe(search);
        window.addEventListener("resize", resizeLayout);
        window.visualViewport?.addEventListener("resize", resizeLayout);
        return () => {
            ++searchRevision;
            searchController?.abort();
            searchSize.disconnect();
            window.removeEventListener("resize", resizeLayout);
            window.visualViewport?.removeEventListener("resize", resizeLayout);
        };
    });

    async function toggleWidth() {
        listsFullWidth = !listsFullWidth;
        localStorage.setItem("listsFullWidth", String(listsFullWidth));
        await tick();
        resizeLayout();
        map?.resize();
    }

    function resizeLayout() {
        if (!resultsElement) return;
        if (window.innerWidth < 768) {
            resultsElement.style.height = "";
            return;
        }
        const viewport = window.visualViewport;
        const bottom = (viewport?.offsetTop ?? 0) + (viewport?.height ?? window.innerHeight);
        resultsElement.style.height = `${Math.max(280, bottom - resultsElement.getBoundingClientRect().top - 12)}px`;
    }

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
            map?.flyTo({
                animate: true,
                zoom: 1,
                center: [0, 0],
            });
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
        if (refreshing) return;
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
        const revision = searchRevision;
        const response = await lists_search_filter({ ...filter }, pagination.page + 1);
        if (revision !== searchRevision) return;
        lists = response.items;
        pagination.page = response.page;
        pagination.totalPages = response.totalPages;
    }

    async function updateFilter(resetMap: boolean = true) {
        const revision = ++searchRevision;
        searchController?.abort();
        const controller = new AbortController();
        searchController = controller;
        const fetcher: typeof fetch = (url, options) => fetch(url, { ...options, signal: controller.signal });
        loading = lists.length === 0;
        refreshing = true;

        if ((selectedList || selectedTrail) && resetMap) {
            selectedList = null;
            selectedTrail = null;
            map?.flyTo({
                animate: true,
                zoom: 1,
                center: [0, 0],
            });
        }

        pagination.page = 1;
        try {
            const response = await lists_search_filter({ ...filter }, 1, 30, fetcher);
            if (revision !== searchRevision) return;
            lists = response.items;
            pagination.page = response.page;
            pagination.totalPages = response.totalPages;
        } catch (error) {
            if (!controller.signal.aborted) throw error;
        } finally {
            if (revision === searchRevision) { loading = false; refreshing = false; }
        }
    }
</script>

<svelte:head>
    <title>{$_("list", { values: { n: 2 } })} | wanderer</title>
</svelte:head>
<main class="lists-layout grid grid-cols-1 items-start gap-3 max-w-7xl" class:full-width={listsFullWidth}>
    <ListSearchBar bind:filter wide={listsFullWidth} onwidthtoggle={toggleWidth} onapply={() => updateFilter()} />
    <div class="list-results" aria-busy={loading || refreshing || loadingNextPage} bind:this={resultsElement}>
        <div class="list-list order-1 md:order-none" class:selected-list={Boolean(selectedList)}>
            <div
                id="list-container"
                class="overflow-y-auto overflow-x-clip"
                onscroll={onListScroll}
            >
                {#if !selectedList}
                    <div class="list-card-results">
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
                        onback={back}
                        onmouseenter={(data) => highlightTrail(data.trail)}
                        onmouseleave={(data) => unHighlightTrail(data.trail)}
                        onchange={(item) => handleDropdownClick(item)}
                        onclick={(data) => selectTrail(data.trail)}
                    ></ListPanel>
                {:else if selectedList && selectedTrail}
                    <TrailInfoPanel
                        initTrail={selectedTrail}
                        mode="list"
                        onback={back}
                        {markers}
                        handle={handleFromRecordWithIRI(selectedTrail)}
                    ></TrailInfoPanel>
                {/if}
            </div>
        </div>
        <div id="trail-map">
            <MapWithElevationMaplibre
                trails={selectedTrail
                    ? [selectedTrail]
                    : (selectedList?.expand?.trails ?? [])}
                waypoints={selectedTrailWaypoints}
                bind:map
                bind:this={mapWithElevation}
                bind:markers
                activeTrail={selectedTrailIndex}
                fitBounds="animate"
                onselect={(trail) => {
                    selectedTrail = trail;
                }}
                showInfoPopup={true}
                showTerrain={true}
            ></MapWithElevationMaplibre>
        </div>
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
    .lists-layout { grid-template-columns: minmax(0, 1fr); margin-inline: auto; min-width: 0; width: calc(100% - 3rem); }
    .lists-layout.full-width { max-width: none; }
    .list-results { display: grid; gap: 1rem; grid-template-columns: minmax(0, 1fr); margin-top: .25rem; min-width: 0; }
    .list-list { display: flex; flex-direction: column; min-height: 0; min-width: 0; }
    .selected-list { border: 1px solid rgba(var(--input-border)); border-radius: .9rem; overflow: hidden; }
    #list-container { flex: 1; min-height: 0; }
    .list-card-results { display: flex; flex-direction: column; gap: 1rem; padding-right: .25rem; }
    .list-list-item { border: 1px solid rgba(var(--input-border)); border-radius: 1rem; overflow: hidden; }
    #trail-map { border: 1px solid rgba(var(--input-border)); border-radius: .9rem; height: 20rem; min-height: 0; min-width: 0; overflow: hidden; }
    @media (min-width: 768px) {
        .list-results { grid-template-columns: clamp(360px, 32vw, 440px) minmax(0, 1fr); height: calc(100dvh - 225px); min-height: 280px; }
        #trail-map { height: 100%; }
    }
    @media (max-width: 639px) {
        .lists-layout { width: calc(100% - 1.5rem); }
    }
</style>
