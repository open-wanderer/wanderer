<script lang="ts">
    import { browser } from "$app/environment";
    import { beforeNavigate, pushState, replaceState } from "$app/navigation";
    import { page } from "$app/state";
    import TrailSearchBar from "$lib/components/trail/trail_search_bar.svelte";
    import TrailList from "$lib/components/trail/trail_list.svelte";
    import TrailMapResults from "$lib/components/trail/trail_map_results.svelte";
    import type { Trail, TrailFilter } from "$lib/models/trail";
    import { trails_search_filter, trails_search_filter_pages, trails_search_map_data } from "$lib/stores/trail_store";
    import { isTrailSearchBounds, trailSearchBoundsFromUrl, type TrailSearchBounds } from "$lib/util/trail_search_util";
    import { sanitizeTrailFilter } from "$lib/util/trail_filter_util";
    import type { Snapshot } from "@sveltejs/kit";
    import type { FeatureCollection } from "geojson";
    import { LngLat } from "maplibre-gl";
    import { onMount, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import { searchPositionForView } from "$lib/models/search_layout";
    import { loadSearchLayout, searchLayout, updateSearchLayout } from "$lib/stores/search_layout_store";

    const FILTER_STORAGE_KEY = "trailListFilter";
    const BOUNDS_STORAGE_KEY = "trailSearchBounds";
    const VIEW_STORAGE_KEY = "trailSearchView";
    const PAGE_SIZE = 48;
    const views = ["cards", "list", "table", "map"];
    type SearchSnapshot = { filter: TrailFilter; bounds?: TrailSearchBounds; viewport?: TrailSearchBounds };

    function hasLinkFilter() {
        return ["category", "subcategory", "author"].some(key => page.url.searchParams.has(key));
    }
    function restoreFilter(): TrailFilter {
        if (!browser || hasLinkFilter()) return page.data.filter;
        try {
            const value = localStorage.getItem(FILTER_STORAGE_KEY);
            return value ? sanitizeTrailFilter(JSON.parse(value), page.data.filter) : page.data.filter;
        } catch { return page.data.filter; }
    }
    function restoreBounds(): TrailSearchBounds | undefined {
        const linked = trailSearchBoundsFromUrl(page.url.searchParams);
        if (linked || !browser || hasLinkFilter()) return linked;
        if (page.url.searchParams.has("lat") || page.url.searchParams.has("lon")) return;
        try {
            const value = JSON.parse(localStorage.getItem(BOUNDS_STORAGE_KEY) ?? "null");
            return isTrailSearchBounds(value) ? value : undefined;
        } catch { return; }
    }
    function placeKey(value: TrailFilter) {
        return Number.isFinite(value.near.lat) && Number.isFinite(value.near.lon)
            ? `${value.near.lat},${value.near.lon},${value.near.radius}` : "";
    }
    function linkedCenter() {
        const params = (browser ? new URL(window.location.href) : page.url).searchParams;
        if (!params.get("lat")?.trim() || !params.get("lon")?.trim()) return;
        const lat = Number(params.get("lat")), lon = Number(params.get("lon"));
        return Number.isFinite(lat) && Math.abs(lat) <= 90 && Number.isFinite(lon) && Math.abs(lon) <= 180 ? { lat, lon } : undefined;
    }

    const mapCenter = untrack(linkedCenter);
    let filter: TrailFilter = $state(restoreFilter());
    let bounds = $state<TrailSearchBounds | undefined>(restoreBounds());
    let mapBounds = $state<TrailSearchBounds | undefined>(untrack(() => bounds));
    let currentDisplayMode = $state(views.includes(page.url.searchParams.get("view") ?? "") ? page.url.searchParams.get("view")! : "cards");
    let activeDisplayMode: string | undefined;
    let mounted = $state(false);
    let routerReady = $state(false);
    let filterPanelTarget = $state<HTMLDivElement>();
    let wideSearchLayout = $state(false);
    let searchPosition = $derived(searchPositionForView($searchLayout.position, currentDisplayMode, wideSearchLayout));
    let loading = $state(true);
    let loadingMore = $state(false);
    let refreshing = $state(false);
    let mapLoading = $state(false);
    let searchError = $state(false);
    let moreError = $state(false);
    let trails: Trail[] = $state([]);
    let mapTrails: Trail[] = $state([]);
    let clusters: FeatureCollection | undefined = $state();
    let pagination = $state({ page: 1, totalPages: 1, items: PAGE_SIZE });
    let hasMore = $derived(pagination.page < pagination.totalPages);
    let mapResults: TrailMapResults | undefined = $state();
    let mapZoom = 11;
    let searchController: AbortController | undefined;
    let mapController: AbortController | undefined;
    let loadedSearchKey = "";
    let loadedMapKey = "";
    let pendingSearchKey = "";
    let pendingMapKey = "";
    let scheduledMap: { key: string; preservePages: boolean; timer: ReturnType<typeof setTimeout> } | undefined;
    let appliedPlace = untrack(() => placeKey(filter));
    let widthPreferences: Record<string, boolean> = $state({});
    let trailsFullWidth = $derived(widthPreferences[currentDisplayMode] ?? false);

    export const snapshot: Snapshot<SearchSnapshot | TrailFilter> = {
        capture: () => JSON.parse(JSON.stringify({ filter, bounds, viewport: mapBounds })),
        restore: (value) => {
            filter = sanitizeTrailFilter("filter" in value ? value.filter : value, page.data.filter);
            bounds = "filter" in value && isTrailSearchBounds(value.bounds) ? value.bounds : undefined;
            mapBounds = "filter" in value && isTrailSearchBounds(value.viewport) ? value.viewport : bounds;
            appliedPlace = placeKey(filter);
            if (!mounted) return;
            const mode = new URL(window.location.href).searchParams.get("view") ?? "cards";
            selectDisplay(mode, false);
            if (mode === "map" && mapBounds) void tick().then(() => mapResults?.focusBounds(mapBounds!));
            else if (mode !== "map") void searchResults();
        },
    };

    onMount(() => {
        const sort = localStorage.getItem("trailSearchSort") ?? localStorage.getItem("sort");
        const validSorts = ["relevance", "name", "distance", "duration", "difficulty", "elevation_gain", "elevation_loss", "like_count", "created", "date"];
        if (validSorts.includes(sort ?? "")) filter.sort = sort as TrailFilter["sort"];
        else filter.sort = "relevance";
        const order = localStorage.getItem("trailSearchSortOrder") ?? localStorage.getItem("sort_order");
        if (order === "+" || order === "-") filter.sortOrder = order;
        try {
            const stored = JSON.parse(localStorage.getItem("trailWidthPreferences") ?? "{}");
            if (stored && typeof stored === "object") widthPreferences = stored;
        } catch { /* Use the default width for invalid stored preferences. */ }
        const savedView = localStorage.getItem(VIEW_STORAGE_KEY);
        const saved = savedView && views.includes(savedView) && savedView !== "map"
            ? savedView : localStorage.getItem("displayOption");
        const linked = page.state.trailSearchView ?? page.url.searchParams.get("view");
        const mode = linked && views.includes(linked) ? linked : saved && views.includes(saved) && saved !== "map" ? saved : "cards";
        mounted = true;
        selectDisplay(mode, false);
        // Wait until the router has initialized, including when live updates
        // remount this component without triggering a navigation callback.
        loadSearchLayout();
        const searchWidth = window.matchMedia("(min-width: 1024px)");
        const updateSearchWidth = () => { wideSearchLayout = searchWidth.matches; };
        updateSearchWidth();
        searchWidth.addEventListener("change", updateSearchWidth);
        const frame = requestAnimationFrame(() => { routerReady = true; });
        return () => {
            cancelAnimationFrame(frame);
            cancelScheduledMap();
            searchController?.abort();
            searchController = undefined;
            mapController?.abort();
            mapController = undefined;
            searchWidth.removeEventListener("change", updateSearchWidth);
        };
    });

    // Navigation shortcuts and browser history select the same result view.
    $effect(() => {
        const mode = page.state.trailSearchView ?? page.url.searchParams.get("view") ?? untrack(() => currentDisplayMode);
        const restoredBounds = page.state.trailSearchBounds;
        const restoredViewport = page.state.trailSearchViewport;
        if (mounted && routerReady) untrack(() => {
            const scopeChanged = restoredBounds !== undefined && JSON.stringify(restoredBounds ?? undefined) !== JSON.stringify(bounds);
            const viewportChanged = restoredViewport !== undefined && JSON.stringify(restoredViewport ?? undefined) !== JSON.stringify(mapBounds);
            if (scopeChanged) bounds = isTrailSearchBounds(restoredBounds) ? restoredBounds : undefined;
            if (viewportChanged) mapBounds = isTrailSearchBounds(restoredViewport) ? restoredViewport : undefined;
            selectDisplay(mode, false);
            if (scopeChanged && mode !== "map" && loadedSearchKey !== resultKey() && pendingSearchKey !== resultKey()) {
                void searchResults(true, false, { keepVisible: true, preservePages: true });
            }
            else if (viewportChanged && mode === "map" && mapBounds) void tick().then(() => mapResults?.focusBounds(mapBounds!));
            syncUrl(true);
        });
    });

    beforeNavigate(({ to }) => {
        if (!browser || !to?.url || to.url.pathname === "/trails" || to.url.pathname === "/map") return;
        localStorage.removeItem(FILTER_STORAGE_KEY);
        localStorage.removeItem(BOUNDS_STORAGE_KEY);
    });

    function persistSearch() {
        localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify(filter));
        if (bounds) localStorage.setItem(BOUNDS_STORAGE_KEY, JSON.stringify(bounds));
        else localStorage.removeItem(BOUNDS_STORAGE_KEY);
    }
    function syncUrl(replace = false, clearLinkFilters = false) {
        if (!routerReady) return;
        const url = new URL(window.location.href);
        if (clearLinkFilters) for (const key of ["category", "subcategory", "author"]) url.searchParams.delete(key);
        url.searchParams.set("view", currentDisplayMode);
        url.searchParams.delete("page");
        for (const key of ["tl_lat", "tl_lon", "br_lat", "br_lon"]) url.searchParams.delete(key);
        const linkedBounds = currentDisplayMode === "map" ? mapBounds ?? bounds : bounds;
        if (linkedBounds) {
            url.searchParams.delete("lat");
            url.searchParams.delete("lon");
            url.searchParams.set("tl_lat", String(linkedBounds.northEast.lat));
            url.searchParams.set("tl_lon", String(linkedBounds.northEast.lng));
            url.searchParams.set("br_lat", String(linkedBounds.southWest.lat));
            url.searchParams.set("br_lon", String(linkedBounds.southWest.lng));
        }
        const state = { ...page.state, trailSearchView: currentDisplayMode,
            trailSearchBounds: bounds ? JSON.parse(JSON.stringify(bounds)) as TrailSearchBounds : null,
            trailSearchViewport: mapBounds ? JSON.parse(JSON.stringify(mapBounds)) as TrailSearchBounds : null };
        if (url.search === window.location.search && page.state.trailSearchView === currentDisplayMode
            && JSON.stringify(page.state.trailSearchBounds) === JSON.stringify(state.trailSearchBounds)
            && JSON.stringify(page.state.trailSearchViewport) === JSON.stringify(state.trailSearchViewport)) return;
        if (replace) replaceState(url, state);
        else pushState(url, state);
    }
    function normalizedBounds(value?: TrailSearchBounds) {
        if (!value) return;
        const round = (number: number) => Math.round(number * 1e7) / 1e7;
        return [round(value.northEast.lat), round(value.northEast.lng), round(value.southWest.lat), round(value.southWest.lng)];
    }
    function resultKey(value = filter, area = bounds) {
        return JSON.stringify([value, normalizedBounds(area)]);
    }
    function mapKey(value = filter, area = mapBounds, zoom = mapZoom) {
        return JSON.stringify([resultKey(value, area), Math.round(zoom * 1e6) / 1e6]);
    }
    function selectDisplay(mode: string, updateUrl = true) {
        if (!views.includes(mode)) mode = "cards";
        currentDisplayMode = mode;
        localStorage.setItem(VIEW_STORAGE_KEY, mode);
        if (mode !== "map") localStorage.setItem("displayOption", mode);
        if (updateUrl) syncUrl();
        if (activeDisplayMode === mode) return;
        activeDisplayMode = mode;
        // Finish a pending viewport check even when the user changes display
        // before the movement debounce expires.
        if (mode !== "map") flushScheduledMap();
        else cancelScheduledMap();
        // The mounted map reports its viewport on entry. Other display changes
        // reuse both the loaded prefix and any ongoing request for that search.
        const key = resultKey();
        if (mode !== "map" && loadedSearchKey !== key && pendingSearchKey !== key) {
            void searchResults(true, false, { keepVisible: true, preservePages: true });
        }
    }
    function toggleWidth() {
        widthPreferences = { ...widthPreferences, [currentDisplayMode]: !trailsFullWidth };
        localStorage.setItem("trailWidthPreferences", JSON.stringify(widthPreferences));
    }

    async function searchResults(reset = true, loadMapData = currentDisplayMode === "map", options: { keepVisible?: boolean; preservePages?: boolean; pages?: number } = {}) {
        searchController?.abort();
        const controller = new AbortController();
        searchController = controller;
        const fetcher: typeof fetch = (url, options) => fetch(url, { ...options, signal: controller.signal });
        const nextPage = reset ? 1 : pagination.page + 1;
        const pagesToKeep = options.pages ?? (options.preservePages ? pagination.page : 1);
        const currentFilter = JSON.parse(JSON.stringify(filter)) as TrailFilter;
        const currentBounds = bounds;
        const key = resultKey(currentFilter, currentBounds);
        if (reset) {
            pendingSearchKey = key;
            refreshing = true;
            loading = !options.keepVisible || !loadedSearchKey;
            loadingMore = false;
            searchError = false;
        }
        moreError = false;
        persistSearch();
        if (reset && loadMapData) void refreshMapData(options.preservePages);
        try {
            const result = reset
                ? await trails_search_filter_pages(currentFilter, pagesToKeep, PAGE_SIZE, fetcher, currentBounds)
                : await trails_search_filter(currentFilter, nextPage, PAGE_SIZE, fetcher, currentBounds);
            if (controller !== searchController) return;
            if (reset) trails = result.items;
            else {
                const existing = new Set(trails.map(trail => trail.id));
                trails = [...trails, ...result.items.filter(trail => !existing.has(trail.id))];
            }
            pagination = { page: result.page, totalPages: result.totalPages, items: PAGE_SIZE };
            loadedSearchKey = key;
        } catch {
            if (controller === searchController) {
                if (reset) searchError = true;
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
    async function refreshMapData(preservePages = false) {
        const currentBounds = mapBounds;
        if (!currentBounds) return;
        const currentFilter = JSON.parse(JSON.stringify(filter)) as TrailFilter;
        const zoom = mapZoom;
        const key = mapKey(currentFilter, currentBounds, zoom);
        if (key === loadedMapKey || key === pendingMapKey) return;
        mapController?.abort();
        const controller = new AbortController();
        mapController = controller;
        const fetcher: typeof fetch = (url, options) => fetch(url, { ...options, signal: controller.signal });
        pendingMapKey = key;
        mapLoading = true;
        searchError = false;
        try {
            const data = await trails_search_map_data(new LngLat(currentBounds.northEast.lng, currentBounds.northEast.lat),
                new LngLat(currentBounds.southWest.lng, currentBounds.southWest.lat), currentFilter, zoom, fetcher);
            if (controller !== mapController || key !== mapKey()) return;
            mapTrails = data.mapTrails;
            clusters = data.clusters;
            loadedMapKey = key;
            // The camera is retained independently. Only a viewport that leaves
            // matching trails outside becomes a filter in the other views.
            bounds = data.hasOutsideBounds === false ? undefined : currentBounds;
            persistSearch();
            syncUrl(true);
            ensureResults(preservePages);
        } catch {
            if (controller === mapController && key === mapKey()) {
                searchError = true;
                if (!loadedSearchKey) loading = false;
            }
        } finally {
            if (controller === mapController) { pendingMapKey = ""; mapLoading = false; }
        }
    }
    function ensureResults(preservePages: boolean) {
        if (resultKey() !== loadedSearchKey && resultKey() !== pendingSearchKey) {
            void searchResults(true, false, { keepVisible: true, preservePages });
        }
    }
    function loadMore() {
        if (loading || refreshing || loadingMore || !hasMore) return;
        loadingMore = true;
        void searchResults(false, false);
    }
    function cancelScheduledMap() {
        if (!scheduledMap) return;
        clearTimeout(scheduledMap.timer);
        scheduledMap = undefined;
    }
    function flushScheduledMap() {
        if (!scheduledMap) return;
        const { preservePages } = scheduledMap;
        cancelScheduledMap();
        void refreshMapData(preservePages);
    }
    function onViewport(next: TrailSearchBounds, zoom: number, entering = false) {
        if (currentDisplayMode !== "map") return;
        const markerKey = mapKey(filter, next, zoom);
        const needsMap = markerKey !== loadedMapKey && markerKey !== pendingMapKey;
        const firstViewport = !mapBounds;
        mapZoom = zoom;
        if (JSON.stringify(normalizedBounds(mapBounds)) !== JSON.stringify(normalizedBounds(next))) mapBounds = next;
        syncUrl(true);
        if (!needsMap) {
            if (markerKey === loadedMapKey) ensureResults(entering);
            return;
        }
        if (scheduledMap?.key === markerKey) return;
        const preservePages = Boolean(scheduledMap?.preservePages || firstViewport || entering);
        mapController?.abort();
        mapController = undefined;
        pendingMapKey = "";
        mapLoading = false;
        cancelScheduledMap();
        scheduledMap = { key: markerKey, preservePages, timer: setTimeout(flushScheduledMap, 200) };
    }
    function applySearch(next: TrailFilter, resetBounds = false) {
        filter = next;
        const place = placeKey(next);
        const focusOnPlace = Boolean(place && (place !== appliedPlace || resetBounds));
        appliedPlace = place;
        if (focusOnPlace || (resetBounds && currentDisplayMode !== "map")) { bounds = undefined; mapBounds = undefined; }
        persistSearch();
        cancelScheduledMap();
        syncUrl(true, true);
        if (focusOnPlace && currentDisplayMode === "map") {
            searchController?.abort();
            searchController = undefined;
            mapController?.abort();
            mapController = undefined;
            pendingSearchKey = "";
            mapResults?.focusPlace(next.near);
        } else void searchResults(true, currentDisplayMode === "map", { keepVisible: true });
    }
    function clearBounds() {
        bounds = undefined;
        mapBounds = undefined;
        syncUrl(true);
        void searchResults();
    }
</script>

<svelte:head><title>{$_("trail", { values: { n: 2 } })} | wanderer</title></svelte:head>

<main class="combined-search-layout grid grid-cols-1 items-start gap-3 max-w-7xl" class:full-width={trailsFullWidth} class:map-mode={currentDisplayMode === "map"}>
    <TrailList bind:filter bind:trails bind:pagination bind:display={currentDisplayMode} {loading}
        onupdate={() => searchResults()} ondisplaychange={selectDisplay}
        managed incremental compactControls inlineControls fullWidthCards
        fullWidth={trailsFullWidth} onwidthtoggle={toggleWidth} {moreResults}
        bind:filterPanelTarget {searchPosition} searchExpansion={$searchLayout.expansion}
        onsearchposition={currentDisplayMode !== "map" && wideSearchLayout ? position => updateSearchLayout({ position }) : undefined}
        onsearchexpansion={searchPosition === "top" ? expansion => updateSearchLayout({ expansion }) : undefined}>
        {#snippet searchBar()}
            <TrailSearchBar categories={page.data.categories} subcategories={page.data.subcategories}
                bind:filter defaults={page.data.filter} onapply={applySearch} inlineControls {filterPanelTarget}
                layout={searchPosition === "top" ? $searchLayout.expansion : "sidebar"}
                {bounds}
                filterSections={page.data.settings?.behavior?.searchFilterSections}
                onclearbounds={currentDisplayMode === "map" ? undefined : clearBounds} />
            {#if searchError}<p role="alert" class="search-message">{$_("trail-search-error")} <button type="button" onclick={() => searchResults()}>{$_("trail-search-retry")}</button></p>
            {/if}
        {/snippet}
        {#snippet mapView()}
            <TrailMapResults bind:this={mapResults} {trails} {mapTrails} {clusters} {loading} {mapLoading} active={currentDisplayMode === "map"}
                initialBounds={mapBounds} initialCenter={mapCenter} boundingBox={page.data.boundingBox}
                settings={page.data.settings} near={filter.near} onviewport={onViewport}
                {moreResults} />
        {/snippet}
    </TrailList>
</main>

{#snippet moreResults()}
    {#if !loading && hasMore}
        <div class="load-more">
            {#if moreError}<p role="alert">{$_("trail-search-error")}</p>{/if}
            <button type="button" class="load-more-button" disabled={loadingMore || refreshing} onclick={loadMore}>
                {#if loadingMore}<i class="fa fa-spinner fa-spin" aria-hidden="true"></i>{/if}
                {$_(loadingMore ? "trail-results-loading-more" : moreError ? "trail-search-retry" : "trail-results-load-more")}
            </button>
        </div>
    {/if}
{/snippet}

<style>
    .combined-search-layout { --trail-controls-inset: .6rem; --trail-controls-top: .3125rem; margin-inline: auto; min-width: 0; position: relative; width: calc(100% - 3rem); }
    .combined-search-layout.full-width { max-width: none; }
    .load-more { align-items: center; display: flex; flex-direction: column; gap: .5rem; padding: 1rem 0 1.5rem; }
    .load-more-button { border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; display: inline-flex; font-size: .9rem; gap: .6rem; justify-content: center; min-width: 10rem; padding: .7rem 1.3rem; }
    .load-more-button:hover { background: rgba(var(--input-background)); }
    .load-more-button:disabled { cursor: wait; opacity: .6; }
    .search-message { font-size: .8rem; }
    .search-message button { text-decoration: underline; }
    @media (max-width: 639px) {
        .combined-search-layout { --trail-controls-inset: .4rem; width: calc(100% - 1.5rem); }
    }
</style>
