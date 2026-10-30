<script lang="ts">
    import { browser } from "$app/env";
    import { replaceState } from "$app/navigation";
    import { page } from "$app/state";
    import LogoText from "#lib/components/logo/logo_text.svelte";
    import LogoTextLight from "#lib/components/logo/logo_text_light.svelte";
    import TrailSearchBar from "#lib/components/trail/trail_search_bar.svelte";
    import TrailResultControls from "#lib/components/trail/trail_result_controls.svelte";
    import TrailMapResults, { type MapCamera } from "#lib/components/trail/trail_map_results.svelte";
    import type { Trail, TrailFilter } from "#lib/models/trail.ts";
    import { loadSearchLayout, searchLayout, updateSearchLayout } from "#lib/stores/search_layout_store.ts";
    import { trails_search_filter, trails_search_map_data } from "#lib/stores/trail_store.ts";
    import { theme } from "#lib/stores/theme_store.ts";
    import { sanitizeTrailFilter } from "#lib/util/trail_filter_util.ts";
    import { isTrailSearchBounds, trailSearchBoundsFromUrl, type TrailSearchBounds } from "#lib/util/trail_search_util.ts";
    import type { Snapshot } from "@sveltejs/kit";
    import type { FeatureCollection } from "geojson";
    import { LngLat } from "maplibre-gl";
    import { onDestroy, onMount, tick, untrack } from "svelte";
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
    let fullscreen = $state(false);
    let fullscreenSearchVisible = $state(false);
    let mapPage: HTMLElement;
    let resultDisplay = $state<"cards" | "list" | "table">("cards");
    let toolbar: HTMLDivElement;
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

    const fullscreenDisplay = browser ? window.matchMedia("(display-mode: fullscreen)") : undefined;
    function updateFullscreenLayout(event?: Event) {
        const value = fullscreenDisplay?.matches ?? false;
        if (value !== fullscreen) void mapResults?.setFullscreen(value, Boolean(event), Boolean(event));
    }
    // Register before child components mount and install their resize listeners.
    if (fullscreenDisplay) {
        fullscreenDisplay.addEventListener("change", updateFullscreenLayout);
        window.addEventListener("resize", updateFullscreenLayout);
        onDestroy(() => {
            fullscreenDisplay.removeEventListener("change", updateFullscreenLayout);
            window.removeEventListener("resize", updateFullscreenLayout);
        });
    }

    onMount(() => {
        mounted = true;
        window.addEventListener("keydown", guardFullscreenShortcut);
        updateFullscreenLayout();
        loadSearchLayout();
        try {
            const stored = localStorage.getItem("mapResultsDisplay");
            if (stored === "cards" || stored === "list" || stored === "table") resultDisplay = stored;
        } catch { /* Keep the default view when storage is unavailable. */ }
        const toolbarSize = new ResizeObserver(() => {
            if (toolbar.offsetHeight) mapPage.style.setProperty("--map-toolbar-height", `${Math.ceil(toolbar.getBoundingClientRect().height)}px`);
        });
        toolbarSize.observe(toolbar);
        const frame = requestAnimationFrame(() => { routerReady = true; syncUrl(); });
        return () => {
            mounted = false;
            cancelAnimationFrame(frame);
            toolbarSize.disconnect();
            window.removeEventListener("keydown", guardFullscreenShortcut);
            cancelScheduled();
            abortRequests();
        };
    });

    $effect(() => {
        fullscreenSearchVisible = false;
        if (!fullscreen) return;
        untrack(() => searchBar?.dismiss());
        const overflow = document.body.style.overflow;
        const background = new Map<HTMLElement, boolean>();
        const remember = (element: HTMLElement) => {
            if (!background.has(element)) background.set(element, element.inert);
        };
        const overlays = 'dialog, [aria-live], [role="status"], [role="alert"], [data-fullscreen-overlay]';
        const protect = (element: HTMLElement) => {
            remember(element);
            // Closed dialogs must remain ready for showModal's synchronous
            // initial focus. Exempt only visible overlays, never their siblings.
            const overlay = element.tagName === 'DIALOG' || (element.matches(overlays)
                && getComputedStyle(element).visibility === 'visible' && element.getClientRects().length > 0);
            if (overlay) { element.inert = background.get(element)!; return; }
            if (element.querySelector(overlays)) {
                element.inert = background.get(element)!;
                for (const child of element.children) if (child instanceof HTMLElement) protect(child);
            } else element.inert = true;
        };
        const updateBackground = () => {
            for (let layer: HTMLElement | null = mapPage; layer && layer !== document.body && layer.parentElement; layer = layer.parentElement) {
                for (const sibling of layer.parentElement.children) {
                    if (sibling !== layer && sibling instanceof HTMLElement && sibling.tagName !== 'SCRIPT' && sibling.tagName !== 'STYLE') protect(sibling);
                }
            }
        };
        updateBackground();
        const observer = new MutationObserver(updateBackground);
        observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ["open", "class", "style"] });
        document.body.style.overflow = "hidden";
        return () => {
            observer.disconnect();
            document.body.style.overflow = overflow;
            for (const [element, inert] of background) element.inert = inert;
        };
    });

    function guardFullscreenShortcut(event: KeyboardEvent) {
        if (event.isComposing || event.defaultPrevented) return;
        if (event.key === "F11" && document.querySelector("dialog:modal")) {
            event.preventDefault();
        }
    }
    async function toggleFullscreenSearch() {
        if (fullscreenSearchVisible) searchBar?.dismiss();
        fullscreenSearchVisible = !fullscreenSearchVisible;
        await tick();
        if (fullscreenSearchVisible) toolbar.querySelector<HTMLInputElement>('input[type="search"]')?.focus();
        else mapPage.querySelector<HTMLButtonElement>(".fullscreen-search-toggle")?.focus();
    }

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

<main class="map-page" class:fullscreen class:fullscreen-search-open={fullscreen && fullscreenSearchVisible} bind:this={mapPage}>
    <div class="map-search-header" hidden={fullscreen && !fullscreenSearchVisible}>
        <div id="map-search-toolbar" class="map-search-toolbar" bind:this={toolbar}>
            <TrailSearchBar bind:this={searchBar} categories={page.data.categories} subcategories={page.data.subcategories}
                bind:filter defaults={page.data.filter} onapply={applySearch} inlineControls
                onplaceselect={location => mapResults?.focusLocation(location)} trailSuggestions
                layout={fullscreen ? "overlay" : $searchLayout.expansion} filterSections={page.data.settings?.behavior?.searchFilterSections} />
            <TrailResultControls buttonsOnly sort={filter.sort === "relevance" && !filter.q.trim() ? "created" : filter.sort}
                order={filter.sortOrder} {sortOptions} onsort={(sort, sortOrder) => applySearch({ ...filter, sort: sort as TrailFilter["sort"], sortOrder })}
                display={resultDisplay} {displayOptions} ondisplay={selectResultDisplay}
                searchExpansion={$searchLayout.expansion} onsearchexpansion={!fullscreen ? expansion => updateSearchLayout({ expansion }) : undefined} />
        </div>
    </div>
    {#if resultError || markerError}
        <p class="map-error" role="alert">{$_("trail-search-error")} <button type="button" onclick={refresh}>{$_("trail-search-retry")}</button></p>
    {/if}
    <div class="map-content">
        <div class="map-results-container" aria-busy={loading || refreshing}>
            <TrailMapResults bind:this={mapResults} bind:fullscreen
                {fullscreenSearchVisible} onfullscreensearchtoggle={toggleFullscreenSearch}
                onsearchfocus={() => searchBar?.focusInput()}
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
    {#if fullscreen}
        <a class="fullscreen-logo" href="/" aria-label="wanderer">
            <span aria-hidden="true">
                {#if $theme === "light"}<LogoText />{:else}<LogoTextLight />{/if}
            </span>
        </a>
    {/if}
</main>

<style>
    .map-page { --trail-controls-inset: .6rem; --trail-controls-top: .3125rem; display: grid; gap: 1rem; margin-inline: auto; min-width: 0; width: calc(100% - 3rem); }
    .map-search-header { min-width: 0; }
    .map-search-toolbar { min-width: 0; position: relative; }
    .map-content { align-items: start; display: grid; gap: 1rem; min-width: 0; }
    .map-results-container { min-width: 0; }
    .map-results-container[aria-busy="true"] :global(.trail-results) { opacity: .55; }
    .map-page.fullscreen { --map-fullscreen-tool-size: 3.125rem; --map-overlay-top: calc(var(--map-fullscreen-tool-size) + 2rem); --map-overlay-bottom: 6rem; --map-controls-top: 0px; background: rgba(var(--background)); display: block; height: 100%; inset: 0; margin: 0; position: fixed; width: 100%; z-index: 40; }
    .fullscreen.fullscreen-search-open { --map-overlay-top: max(calc(var(--map-fullscreen-tool-size) + 2rem), calc(var(--map-toolbar-height, 50px) + 2rem)); --map-controls-top: calc(var(--map-toolbar-height, 50px) + 2rem); }
    .fullscreen .map-search-header { left: calc(var(--map-fullscreen-tool-size) + 2rem); position: absolute; right: 1rem; top: 1rem; z-index: 3; }
    .fullscreen .map-search-header[hidden] { display: none; }
    .fullscreen .map-search-toolbar { box-shadow: 0 3px 16px rgb(0 0 0 / .2); }
    .fullscreen-logo { border-radius: 50%; bottom: 1rem; display: block; filter: drop-shadow(0 2px 4px rgb(0 0 0 / .25)); height: 64px; left: 1rem; overflow: hidden; position: absolute; width: 64px; z-index: 3; }
    .fullscreen-logo span { display: block; height: 100%; }
    .fullscreen-logo :global(svg) { display: block; height: 100%; max-width: none; width: auto; }
    .fullscreen-logo:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: 3px; }
    .fullscreen .map-error { background: rgba(var(--menu-background)); border-radius: .5rem; bottom: 1rem; left: 50%; padding: .5rem; position: absolute; transform: translateX(-50%); z-index: 3; }
    .load-more { display: grid; justify-content: center; padding: .5rem 0; }
    .load-more button { border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; padding: .7rem 1.3rem; }
    .load-more button:disabled { cursor: wait; opacity: .6; }
    @media (max-width: 639px) {
        .map-page { --trail-controls-inset: .4rem; width: calc(100% - 1.5rem); }
    }
</style>
