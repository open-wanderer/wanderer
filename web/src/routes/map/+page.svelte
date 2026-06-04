<script lang="ts">
    import { browser } from "$app/environment";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import Search, {
        type SearchItem,
    } from "$lib/components/base/search.svelte";
    import type { SelectItem } from "$lib/components/base/select.svelte";
    import Select from "$lib/components/base/select.svelte";
    import SkeletonCard from "$lib/components/base/skeleton_card.svelte";
    import EmptyStateSearch from "$lib/components/empty_states/empty_state_search.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import TrailCard from "$lib/components/trail/trail_card.svelte";
    import TrailFilterPanel from "$lib/components/trail/trail_filter_panel.svelte";
    import type { Settings } from "$lib/models/settings";
    import {
        defaultTrailSearchAttributes,
        type Trail,
        type TrailBoundingBox,
        type TrailFilter,
        type TrailSearchResult,
    } from "$lib/models/trail";
    import { categories } from "$lib/stores/category_store";
    import {
        searchMulti,
        type ListSearchResult,
        type LocationSearchResult,
    } from "$lib/stores/search_store";
    import { trails_search_bounding_box } from "$lib/stores/trail_store";
    import { getIconForLocation } from "$lib/util/icon_util";
    import type { Snapshot } from "@sveltejs/kit";
    import type { FeatureCollection } from "geojson";
    import * as M from "maplibre-gl";
    import { _ } from "svelte-i18n";
    import { slide } from "svelte/transition";

    let trails: Trail[] = $state([]);
    let mapTrails: Trail[] = $state([]);
    let clusters: FeatureCollection | undefined = $state();

    let map: M.Map | undefined = $state();
    let mapWithElevation: MapWithElevationMaplibre | undefined = $state();
    let searchDropdownItems: SearchItem[] = $state([]);

    let showFilter: boolean = $state(false);
    let showMap: boolean = $state(true);

    let filter: TrailFilter = $state(page.data.filter);
    const maxBoundingBox: TrailBoundingBox = page.data.boundingBox;
    const settings: Settings = page.data.settings;

    let loading: boolean = $state(true);
    let loadingNextPage: boolean = false;

    let pagination = {
        page: 1,
        totalPages: 1,
    };
    let searchRequestId = 0;

    const sortOptions: SelectItem[] = [
        { text: $_("name"), value: "name" },
        { text: $_("distance"), value: "distance" },
        { text: $_("duration"), value: "duration" },
        { text: $_("difficulty"), value: "difficulty" },
        { text: $_("elevation-gain"), value: "elevation_gain" },
        { text: $_("elevation-loss"), value: "elevation_loss" },
        { text: $_("likes"), value: "like_count" },
        { text: $_("creation-date"), value: "created" },
        { text: $_("date"), value: "date" },
    ];

    export const snapshot: Snapshot<TrailFilter> = {
        capture: () => filter,
        restore: (value) => {
            filter = value;
            handleFilterUpdate();
        },
    };

    async function search(q: string) {
        const r = await searchMulti({
            queries: [
                {
                    indexUid: "trails",
                    attributesToRetrieve: defaultTrailSearchAttributes,
                    q: q,
                    limit: 3,
                },
                {
                    indexUid: "lists",
                    q: q,
                    limit: 3,
                },
                {
                    indexUid: "locations",
                    q: q,
                    limit: 3,
                },
            ],
        });

        const trailItems = (r[0]?.hits || []).map((t: TrailSearchResult) => ({
            text: t.name,
            description: `Trail ${t.location.length ? ", " + t.location : ""}`,
            value: `@${t.author_name}${t.domain ? `@${t.domain}` : ""}/${t.id}`,
            icon: "route",
        }));
        const listItems = (r[1]?.hits || []).map((t: ListSearchResult) => ({
            text: t.name,
            description: `List, ${t.trails} ${$_("trail", { values: { n: t.trails } })}`,
            value: t.id,
            icon: "layer-group",
        }));
        const cityItems = (r[2]?.hits || []).map((c: LocationSearchResult) => ({
            text: c.name,
            description: c.description,
            value: c,
            icon: getIconForLocation(c),
        }));

        searchDropdownItems = [...trailItems, ...listItems, ...cityItems];
    }

    function handleSearchClick(item: SearchItem) {
        if (item.icon == "route") {
            goto(`/map/trail/${item.value}`);
        } else if (item.icon == "layer-group") {
            goto(`/lists?list=${item.value}`);
        } else {
            map?.setCenter([item.value.lon, item.value.lat]);
            map?.setZoom(14);
        }
    }

    async function searchTrails(
        northEast: M.LngLat,
        southWest: M.LngLat,
        reset: boolean = true,
        loadMapData: boolean = true,
    ) {
        const requestId =
            reset || loadMapData ? ++searchRequestId : searchRequestId;

        if (reset) {
            pagination.page = 1;
            loading = true;
        }

        const trailsInBox = await trails_search_bounding_box(
            northEast,
            southWest,
            filter,
            pagination.page,
            map?.getZoom(),
            50,
            loadMapData,
        );

        if (requestId !== searchRequestId) {
            return false;
        }

        pagination.totalPages = trailsInBox.totalPages;
        trails = trailsInBox.trails;
        if (loadMapData) {
            mapTrails = trailsInBox.mapTrails;
            clusters = trailsInBox.clusters;
        }
        loading = false;
        return true;
    }

    function handleTrailCardMouseEnter(trail: Trail) {
        mapWithElevation?.highlightCluster(trail);
    }

    function handleTrailCardMouseLeave(trail: Trail) {
        mapWithElevation?.unHighlightCluster();
    }

    function setSort() {
        if (!filter) {
            return;
        }
        localStorage.setItem("sort", filter.sort);
        handleFilterUpdate();
    }

    function setSortOrder() {
        if (!filter) {
            return;
        }
        if (filter.sortOrder === "+") {
            filter.sortOrder = "-";
        } else {
            filter.sortOrder = "+";
        }
        localStorage.setItem("sort_order", filter.sortOrder);
        handleFilterUpdate();
    }

    async function handleFilterUpdate() {
        if (!map) {
            return;
        }
        const bounds = map.getBounds();
        await searchTrails(bounds.getNorthEast(), bounds.getSouthWest());
    }

    let moveTimeout: ReturnType<typeof setTimeout> | undefined;
    async function handleMapMove() {
        if (!map) {
            return;
        }

        if (moveTimeout) {
            clearTimeout(moveTimeout);
        }
        moveTimeout = setTimeout(async () => {
            const bounds = map!.getBounds();
            const west = bounds.getWest();
            const east = bounds.getEast();
            const north = bounds.getNorth();
            const south = bounds.getSouth();

            let normalizedSW: M.LngLat;
            let normalizedNE: M.LngLat;

            if (east - west >= 360) {
                // Global view
                normalizedSW = new M.LngLat(-180, south);
                normalizedNE = new M.LngLat(180, north);
            } else {
                // Handle wrap-around
                normalizedSW = new M.LngLat(
                    ((((west + 180) % 360) + 360) % 360) - 180,
                    south,
                );
                normalizedNE = new M.LngLat(
                    ((((east + 180) % 360) + 360) % 360) - 180,
                    north,
                );
            }

            const applied = await searchTrails(normalizedNE, normalizedSW);
            if (!applied) {
                return;
            }

            page.url.searchParams.set("tl_lat", north.toString());
            page.url.searchParams.set("tl_lon", east.toString());
            page.url.searchParams.set("br_lat", south.toString());
            page.url.searchParams.set("br_lon", west.toString());

            goto(`?${page.url.searchParams.toString()}`, {
                replaceState: true,
                noScroll: true,
                keepFocus: true,
            });
        }, 200);
    }

    function handleMapInit() {
        if (
            page.url.searchParams.has("tl_lat") &&
            page.url.searchParams.has("tl_lon") &&
            page.url.searchParams.has("br_lat") &&
            page.url.searchParams.has("br_lon")
        ) {
            const boundingBox: M.LngLatBoundsLike = [
                [
                    parseFloat(page.url.searchParams.get("br_lon")!),
                    parseFloat(page.url.searchParams.get("tl_lat")!),
                ],
                [
                    parseFloat(page.url.searchParams.get("tl_lon")!),
                    parseFloat(page.url.searchParams.get("br_lat")!),
                ],
            ];
            map?.fitBounds(boundingBox, { animate: false });
        } else if (
            page.url.searchParams.has("lat") &&
            page.url.searchParams.has("lon")
        ) {
            const lat = page.url.searchParams.get("lat");
            const lon = page.url.searchParams.get("lon");
            map?.setZoom(14);
            map?.setCenter([parseFloat(lon!), parseFloat(lat!)]);
        } else if (
            settings &&
            settings.mapFocus == "trails" &&
            (maxBoundingBox.min_lon != 0 ||
                maxBoundingBox.max_lat != 0 ||
                maxBoundingBox.max_lon != 0 ||
                maxBoundingBox.min_lat != 0)
        ) {
            if (
                maxBoundingBox.min_lon == maxBoundingBox.max_lon &&
                maxBoundingBox.min_lat == maxBoundingBox.max_lat
            ) {
                map?.setZoom(12);
                map?.setCenter([
                    maxBoundingBox.min_lon,
                    maxBoundingBox.min_lat,
                ]);
            } else {
                const boundingBox: M.LngLatBoundsLike = [
                    [maxBoundingBox.min_lon, maxBoundingBox.max_lat],
                    [maxBoundingBox.max_lon, maxBoundingBox.min_lat],
                ];
                map?.fitBounds(boundingBox, { animate: false, padding: 32 });
            }
        } else if (
            settings &&
            settings.mapFocus == "location" &&
            settings.location
        ) {
            map?.setZoom(12);
            map?.setCenter([settings.location.lon, settings.location.lat]);
        } else {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    const lat = position.coords.latitude;
                    const lon = position.coords.longitude;
                    map?.setZoom(12);
                    map?.setCenter([lon, lat]);
                },
                (error) => {
                    console.error("Error getting user location:", error);
                },
            );
        }
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
        if (!map) {
            return;
        }
        pagination.page += 1;
        const bounds = map.getBounds();
        await searchTrails(bounds.getNorthEast(), bounds.getSouthWest(), false, false);
    }
</script>

<svelte:head>
    <title>{$_("map")} | wanderer</title>
</svelte:head>
<main class="grid grid-cols-1 md:grid-cols-[400px_1fr]">
    <div
        id="trail-list"
        class="flex flex-col items-stretch gap-4 px-3 md:px-8 overflow-y-scroll"
        onscroll={onListScroll}
    >
        <div class="sticky top-0 z-10 bg-background pb-4">
            <div class="flex items-center gap-2 md:gap-4">
                <Search
                    extraClasses="w-full"
                    onupdate={search}
                    onclick={handleSearchClick}
                    placeholder="{$_('search-for-trails-places')}..."
                    items={searchDropdownItems}
                ></Search>
                <button
                    aria-label="Toggle map"
                    class="btn-icon md:hidden"
                    onclick={() => (showMap = !showMap)}
                    ><i
                        class="fa-regular fa-{showMap
                            ? 'rectangle-list'
                            : 'map'}"
                    ></i></button
                >
            </div>
            <div class="flex items-center gap-2 mt-2 mb-4">
                <Select
                    bind:value={filter.sort}
                    items={sortOptions}
                    onchange={setSort}
                ></Select>
                <button
                    aria-label="Change sort order"
                    id="sort-order-btn"
                    class="btn-icon"
                    class:rotated={filter.sortOrder == "-"}
                    onclick={() => setSortOrder()}
                    ><i class="fa fa-arrow-up"></i></button
                >
                <div class="basis-full"></div>
                <button
                    aria-label="Open filter"
                    class="btn-icon"
                    onclick={() => (showFilter = !showFilter)}
                    ><i class="fa fa-sliders"></i></button
                >
            </div>
            {#if showFilter}
                <div in:slide out:slide>
                    <TrailFilterPanel
                        categories={$categories}
                        showTrailSearch={false}
                        showCitySearch={false}
                        bind:filter
                        onupdate={handleFilterUpdate}
                    ></TrailFilterPanel>
                </div>
            {/if}
        </div>

        {#if !showFilter && (!showMap || (browser && window.innerWidth >= 768))}
            {#if loading}
                {#each { length: 4 } as _, index}
                    <SkeletonCard></SkeletonCard>
                {/each}
            {:else}
                {#if trails.length == 0}
                    <EmptyStateSearch></EmptyStateSearch>
                {/if}
                {#each trails.filter(t => t.name !== "") as trail, i}
                    <a
                        href="/map/trail/@{trail.author}{trail.domain
                            ? `@${trail.domain}`
                            : ''}/{trail.id}"
                    >
                        <TrailCard
                            {trail}
                            fullWidth={true}
                            hovered={false}
                            selected={false}
                            onmouseenter={() =>
                                handleTrailCardMouseEnter(trail)}
                            onmouseleave={() =>
                                handleTrailCardMouseLeave(trail)}
                        ></TrailCard>
                    </a>
                {/each}
            {/if}
        {/if}
    </div>
    <div
        id="trail-map"
        class:hidden={!showMap && browser && window.innerWidth < 768}
    >
        <MapWithElevationMaplibre
            onmoveend={handleMapMove}
            oninit={handleMapInit}
            trails={mapTrails}
            serverClusters={clusters}
            showElevation={false}
            showTerrain={true}
            showInfoPopup={true}
            activeTrail={-1}
            fitBounds="off"
            clusterTrails={true}
            bind:map

            bind:this={mapWithElevation}
        ></MapWithElevationMaplibre>
    </div>
</main>

<style>
    #trail-map {
        height: calc(100vh - 180px);
    }
    @media only screen and (min-width: 768px) {
        #trail-map,
        #trail-list {
            height: calc(100vh - 124px);
        }
    }

    #sort-order-btn {
        transition: transform 0.5s ease;
    }
    :global(.rotated) {
        transform: rotate(180deg);
    }
</style>
