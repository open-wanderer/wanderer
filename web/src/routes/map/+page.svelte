<script lang="ts">
    import { replaceState } from "$app/navigation";
    import { page } from "$app/state";
    import TrailSearchBar from "#lib/components/trail/trail_search_bar.svelte";
    import TrailResultControls from "#lib/components/trail/trail_result_controls.svelte";
    import TrailMapResults, { type MapCamera } from "#lib/components/trail/trail_map_results.svelte";
    import type { Trail, TrailFilter } from "#lib/models/trail.ts";
    import { loadSearchLayout, searchLayout, updateSearchLayout } from "#lib/stores/search_layout_store.ts";
    import { trails_search_filter, trails_search_map_data } from "#lib/stores/trail_store.ts";
    import { sanitizeTrailFilter } from "#lib/util/trail_filter_util.ts";
    import { isTrailSearchBounds, trailSearchBoundsFromUrl, type TrailSearchBounds } from "#lib/util/trail_search_util.ts";
    import type { Snapshot } from "@sveltejs/kit";
    import type { FeatureCollection } from "geojson";
    import { LngLat } from "maplibre-gl";
    import { onMount, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";

    const PAGE_SIZE = 48;
    type MapSnapshot = { filter: TrailFilter; viewport?: TrailSearchBounds; zoom?: number; camera?: MapCamera };
    let filter: TrailFilter = $state(untrack(() => page.state.mapSearchFilter
        ? { ...sanitizeTrailFilter(page.state.mapSearchFilter, page.data.filter), ...(page.state.mapSearchFilter.shared === undefined ? { shared: undefined } : {}) } : page.data.filter));
    let viewport = $state<TrailSearchBounds | undefined>(trailSearchBoundsFromUrl(page.url.searchParams));
    let zoom = 11;
    let restoredCamera = $state<MapCamera>();
    let renderedViewport: TrailSearchBounds | undefined;
    const initialCenter = untrack(() => {
        const params = page.url.searchParams;
        if (!params.get("lat")?.trim() || !params.get("lon")?.trim()) return;
        const lat = Number(params.get("lat"));
        const lon = Number(params.get("lon"));
        if (Number.isFinite(lat) && Math.abs(lat) <= 90 && Number.isFinite(lon) && Math.abs(lon) <= 180) return { lat, lon };
    });
    let mapResults: TrailMapResults | undefined = $state();
    let searchBar: TrailSearchBar | undefined = $state();
    let resultDisplay = $state<"cards" | "list" | "table">("cards");
    let trails: Trail[] = $state([]);
    let mapTrails: Trail[] = $state([]);
    let clusters = $state<FeatureCollection>();
    let pagination = $state({ page: 1, totalPages: 1 });
    let loading = $state(true);
    let refreshing = $state(false);
    let loadingMore = $state(false);
    let mapLoading = $state(false);
    let resultError = $state(false);
    let markerError = $state(false);
    let moreError = $state(false);
    let scheduled = $state<ReturnType<typeof setTimeout>>();
    let scheduledRender: ReturnType<typeof setTimeout> | undefined;
    let searchController: AbortController | undefined;
    let mapController: AbortController | undefined;
    let loadedSearchKey = "";
    let pendingSearchKey = "";
    let loadedMapKey = "";
    let pendingMapKey = "";
    let mounted = false;
    let routerReady = false;
    let appliedPlace = untrack(() => placeKey(filter));
    let hasMore = $derived(pagination.page < pagination.totalPages);
    const sortKeys = ["name", "distance", "duration", "difficulty", "elevation_gain", "elevation_loss", "like_count", "created", "date"] as const;
    let displayOptions = $derived([
        { text: $_("card", { values: { n: 2 } }), value: "cards" },
        { text: $_("list", { values: { n: 1 } }), value: "list" },
        { text: $_("table"), value: "table" },
    ]);
    let sortOptions = $derived([
        ...(filter.q.trim() ? [{ text: $_("trail-results-relevance"), value: "relevance" }] : []),
        ...sortKeys.map(key => ({ value: key, text: $_(({ elevation_gain: "elevation-gain", elevation_loss: "elevation-loss", like_count: "likes", created: "creation-date" } as Record<string, string>)[key] ?? key) })),
    ]);

    export const snapshot: Snapshot<MapSnapshot | TrailFilter> = {
        capture: () => $state.snapshot({ filter, viewport, zoom, camera: mapResults?.captureView() }),
        restore: value => {
            const savedFilter = "filter" in value ? value.filter : value;
            filter = { ...sanitizeTrailFilter(savedFilter, page.data.filter), ...(savedFilter.shared === undefined ? { shared: undefined } : {}) };
            searchBar?.restoreFilter(filter);
            if ("filter" in value) {
                if (isTrailSearchBounds(value.viewport)) viewport = value.viewport;
                if (value.zoom !== undefined && Number.isFinite(value.zoom)) zoom = value.zoom;
                restoredCamera = value.camera;
            }
            renderedViewport = undefined;
            appliedPlace = placeKey(filter);
            if (mounted) {
                cancelScheduled();
                void tick().then(() => {
                    const positioned = restoredCamera ? mapResults?.focusCamera(restoredCamera)
                        : viewport ? mapResults?.focusBounds(viewport) : false;
                    // Before oninit, initializeViewport owns the fitted bounds
                    // and its single search. Never query the unfitted snapshot.
                    if (positioned) refresh();
                });
            }
        },
    };
    onMount(() => {
        mounted = true;
        loadSearchLayout();
        try {
            const stored = localStorage.getItem("mapResultsDisplay");
            if (stored === "cards" || stored === "list" || stored === "table") resultDisplay = stored;
        } catch { /* Keep the default view when storage is unavailable. */ }
        const frame = requestAnimationFrame(() => { routerReady = true; syncUrl(); });
        return () => {
            mounted = false;
            cancelAnimationFrame(frame);
            cancelScheduled();
            abortRequests();
        };
    });

    function placeKey(value: TrailFilter) {
        return Number.isFinite(value.near.lat) && Number.isFinite(value.near.lon)
            ? `${value.near.lat},${value.near.lon},${value.near.radius}` : "";
    }
    function setAreaParams(params: URLSearchParams, area: TrailSearchBounds) {
        params.set("tl_lat", String(area.northEast.lat));
        params.set("tl_lon", String(area.northEast.lng));
        params.set("br_lat", String(area.southWest.lat));
        params.set("br_lon", String(area.southWest.lng));
    }
    function syncUrl(clearLinkFilters = false) {
        if (!routerReady || !viewport) return;
        const url = new URL(window.location.href);
        for (const key of ["lat", "lon", "view", "page", ...(clearLinkFilters ? ["author", "category", "subcategory"] : [])]) url.searchParams.delete(key);
        const linkedBounds = trailSearchBoundsFromUrl(url.searchParams);
        // Map fits can differ by floating-point noise. Keep the existing URL
        // when it already represents the same search area.
        if (!linkedBounds || searchKey(filter, linkedBounds) !== searchKey()) setAreaParams(url.searchParams, viewport);
        if (url.search !== window.location.search) replaceState(url, $state.snapshot(page.state));
    }
    function searchKey(value = filter, area = viewport) {
        const rounded = area && [area.northEast.lat, area.northEast.lng, area.southWest.lat, area.southWest.lng]
            .map(coordinate => Math.round(coordinate * 1e7) / 1e7);
        return JSON.stringify([value, rounded]);
    }
    function mapKey() {
        return JSON.stringify([searchKey(filter, renderedViewport ?? viewport), Math.round(zoom * 1e6) / 1e6]);
    }
    function cancelScheduled() {
        if (scheduled !== undefined) clearTimeout(scheduled);
        scheduled = undefined;
    }
    function abortRequests() {
        clearTimeout(scheduledRender);
        scheduledRender = undefined;
        searchController?.abort();
        searchController = undefined;
        mapController?.abort();
        mapController = undefined;
        pendingSearchKey = "";
        pendingMapKey = "";
        refreshing = false;
        loadingMore = false;
        mapLoading = false;
    }
    function pauseViewportRefresh() {
        cancelScheduled();
        clearTimeout(scheduledRender);
        scheduledRender = undefined;
    }
    function onViewport(next: TrailSearchBounds, nextZoom: number, _entering = false, settled = false) {
        clearTimeout(scheduledRender);
        scheduledRender = undefined;
        const previous = searchKey();
        const previousMap = mapKey();
        viewport = next;
        renderedViewport = next;
        zoom = nextZoom;
        syncUrl();
        if (previous !== searchKey()) {
            searchController?.abort();
            searchController = undefined;
            pendingSearchKey = "";
            loadingMore = false;
            if (loadedSearchKey === searchKey()) refreshing = false;
            loading = !loadedSearchKey;
        }
        const key = mapKey();
        if (pendingMapKey && pendingMapKey !== key) {
            mapController?.abort();
            mapController = undefined;
            pendingMapKey = "";
            mapLoading = false;
        }
        if (key === loadedMapKey || key === pendingMapKey) {
            cancelScheduled();
            if (loadedSearchKey !== searchKey() && pendingSearchKey !== searchKey()) void searchResults();
            return;
        }
        if (!settled && previousMap === key && scheduled !== undefined) return;
        mapController?.abort();
        mapController = undefined;
        pendingMapKey = "";
        cancelScheduled();
        if (settled) refresh();
        else scheduled = setTimeout(refresh, 200);
    }
    function onMapRender(nextZoom: number, bounds?: TrailSearchBounds, settled = false) {
        zoom = nextZoom;
        if (bounds) renderedViewport = bounds;
        if (pendingMapKey && pendingMapKey !== mapKey()) {
            mapController?.abort();
            mapController = undefined;
            pendingMapKey = "";
            mapLoading = false;
        }
        clearTimeout(scheduledRender);
        scheduledRender = undefined;
        if (settled) {
            // Also consume a preceding pan timer, using the final rendered bounds.
            refresh();
            return;
        }
        scheduledRender = setTimeout(() => {
            scheduledRender = undefined;
            void refreshMapData();
        }, 200);
    }
    function refresh() {
        cancelScheduled();
        if (!viewport) return;
        if (loadedSearchKey !== searchKey() && pendingSearchKey !== searchKey()) void searchResults();
        void refreshMapData();
    }
    async function searchResults(reset = true) {
        if (!viewport) return;
        searchController?.abort();
        const controller = new AbortController();
        searchController = controller;
        const key = searchKey();
        pendingSearchKey = key;
        const currentFilter = JSON.parse(JSON.stringify(filter)) as TrailFilter;
        const area = viewport;
        const nextPage = reset ? 1 : pagination.page + 1;
        refreshing = reset;
        loadingMore = !reset;
        loading = !loadedSearchKey;
        resultError = false;
        moreError = false;
        try {
            const result = await trails_search_filter(currentFilter, nextPage, PAGE_SIZE,
                (url, options) => fetch(url, { ...options, signal: controller.signal }), area);
            if (controller !== searchController || key !== searchKey()) return;
            if (reset) trails = result.items;
            else {
                const ids = new Set(trails.map(trail => trail.id));
                trails = [...trails, ...result.items.filter(trail => !ids.has(trail.id))];
            }
            pagination = { page: result.page, totalPages: result.totalPages };
            loadedSearchKey = key;
        } catch {
            if (controller === searchController) {
                if (reset) resultError = true;
                else moreError = true;
            }
        } finally {
            if (controller === searchController) {
                pendingSearchKey = "";
                loading = false;
                refreshing = false;
                loadingMore = false;
            }
        }
    }
    async function refreshMapData() {
        if (!viewport) return;
        const key = mapKey();
        if (key === loadedMapKey || key === pendingMapKey) return;
        mapController?.abort();
        const controller = new AbortController();
        mapController = controller;
        pendingMapKey = key;
        mapLoading = true;
        markerError = false;
        const area = renderedViewport ?? viewport;
        const currentFilter = JSON.parse(JSON.stringify(filter)) as TrailFilter;
        try {
            const data = await trails_search_map_data(new LngLat(area.northEast.lng, area.northEast.lat),
                new LngLat(area.southWest.lng, area.southWest.lat), currentFilter, zoom,
                (url, options) => fetch(url, { ...options, signal: controller.signal }));
            if (controller !== mapController || key !== mapKey()) return;
            mapTrails = data.mapTrails;
            clusters = data.clusters;
            loadedMapKey = key;
        } catch {
            if (controller === mapController) markerError = true;
        } finally {
            if (controller === mapController) { pendingMapKey = ""; mapLoading = false; }
        }
    }
    function applySearch(next: TrailFilter, resetBounds = false) {
        filter = next;
        const place = placeKey(next);
        const focus = Boolean(place && (place !== appliedPlace || resetBounds));
        appliedPlace = place;
        cancelScheduled();
        abortRequests();
        syncUrl(true);
        if (focus) mapResults?.focusPlace(next.near);
        else refresh();
    }
    function handleTrailsUpdated(updatedTrails: Trail[] = []) {
        if (updatedTrails.length) {
            const updates = new Map(updatedTrails.filter(trail => trail.id).map(trail => [trail.id, trail]));
            const mergeUpdated = (trail: Trail) => {
                const updated = updates.get(trail.id);
                return updated ? { ...trail, ...updated, author: trail.author, expand: { ...trail.expand, ...updated.expand } } : trail;
            };
            // Meili indexes asynchronously. Keep successful bulk edits locally
            // so an immediate search cannot overwrite them with stale records.
            trails = trails.map(mergeUpdated);
            mapTrails = mapTrails.map(mergeUpdated);
            return;
        }
        cancelScheduled();
        abortRequests();
        loadedSearchKey = "";
        loadedMapKey = "";
        refresh();
    }
    function loadMore() {
        if (loading || refreshing || loadingMore || scheduled !== undefined || !hasMore) return;
        void searchResults(false);
    }
    function selectResultDisplay(display: string) {
        if (display !== "cards" && display !== "list" && display !== "table") return;
        resultDisplay = display;
        try { localStorage.setItem("mapResultsDisplay", display); } catch { /* Keep the view usable without storage. */ }
    }
    function sortTable(sort: string) {
        const sortOrder = sort === filter.sort && filter.sortOrder === "+" ? "-" : "+";
        applySearch({ ...filter, sort: sort as TrailFilter["sort"], sortOrder });
    }
</script>

<svelte:head><title>{$_("map")} | wanderer</title></svelte:head>

<main class="map-page">
    <div class="map-search-header">
        <div class="map-search-toolbar">
            <TrailSearchBar bind:this={searchBar} categories={page.data.categories} subcategories={page.data.subcategories}
                bind:filter defaults={page.data.filter} onapply={applySearch} inlineControls
                onplaceselect={location => mapResults?.focusLocation(location)} trailSuggestions
                layout={$searchLayout.expansion} filterSections={page.data.settings?.behavior?.searchFilterSections} />
            <TrailResultControls buttonsOnly sort={filter.sort === "relevance" && !filter.q.trim() ? "created" : filter.sort}
                order={filter.sortOrder} {sortOptions} onsort={(sort, sortOrder) => applySearch({ ...filter, sort: sort as TrailFilter["sort"], sortOrder })}
                display={resultDisplay} {displayOptions} ondisplay={selectResultDisplay}
                searchExpansion={$searchLayout.expansion} onsearchexpansion={expansion => updateSearchLayout({ expansion })} />
        </div>
    </div>
    {#if resultError || markerError}
        <p class="map-error" role="alert">{$_("trail-search-error")} <button type="button" onclick={refresh}>{$_("trail-search-retry")}</button></p>
    {/if}
    <div class="map-content">
        <div class="map-results-container" aria-busy={loading || refreshing}>
            <TrailMapResults bind:this={mapResults}
                {trails} {mapTrails} {clusters} {loading} {mapLoading}
                initialBounds={viewport} initialCamera={restoredCamera} {initialCenter} boundingBox={page.data.boundingBox}
                settings={page.data.settings} display={resultDisplay} near={filter.near} onviewport={onViewport}
                onzoom={onMapRender} onwindowresize={pauseViewportRefresh}
                onupdate={handleTrailsUpdated} {filter} tableHeader={sortOptions.filter(option => option.value !== "elevation_loss" && option.value !== "relevance")} onsort={sortTable}>
                {#snippet moreResults()}
                    {#if hasMore}
                        <div class="load-more">
                            {#if moreError}<p role="alert">{$_("trail-search-error")}</p>{/if}
                            <button type="button" disabled={loadingMore || refreshing || scheduled !== undefined} onclick={loadMore}>
                                {$_(loadingMore ? "trail-results-loading-more" : moreError ? "trail-search-retry" : "trail-results-load-more")}
                            </button>
                        </div>
                    {/if}
                {/snippet}
            </TrailMapResults>
        </div>
    </div>
</main>

<style>
    .map-page { --trail-controls-inset: .6rem; --trail-controls-top: .3125rem; display: grid; gap: 1rem; margin-inline: auto; min-width: 0; width: calc(100% - 3rem); }
    .map-search-header { min-width: 0; }
    .map-search-toolbar { min-width: 0; position: relative; }
    .map-content { align-items: start; display: grid; gap: 1rem; min-width: 0; }
    .map-results-container { min-width: 0; }
    .map-results-container[aria-busy="true"] :global(.trail-results) { opacity: .55; }
    .load-more { display: grid; justify-content: center; padding: .5rem 0; }
    .load-more button { border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; padding: .7rem 1.3rem; }
    .load-more button:disabled { cursor: wait; opacity: .6; }
    @media (max-width: 639px) {
        .map-page { --trail-controls-inset: .4rem; width: calc(100% - 1.5rem); }
    }
</style>
