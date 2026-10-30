<script lang="ts">
    import { browser } from "$app/env";
    import { beforeNavigate } from "$app/navigation";
    import { page } from "$app/state";
    import TrailSearchBar from "#lib/components/trail/trail_search_bar.svelte";
    import TrailList from "#lib/components/trail/trail_list.svelte";
    import type { Trail, TrailFilter } from "#lib/models/trail.ts";
    import { searchPositionForViewport } from "#lib/models/search_layout.ts";
    import { loadSearchLayout, searchLayout, updateSearchLayout } from "#lib/stores/search_layout_store.ts";
    import { trails_search_filter, trails_search_filter_pages } from "#lib/stores/trail_store.ts";
    import { sanitizeTrailFilter } from "#lib/util/trail_filter_util.ts";
    import type { Snapshot } from "@sveltejs/kit";
    import { onMount, tick } from "svelte";
    import { _ } from "svelte-i18n";

    const FILTER_STORAGE_KEY = "trailListFilter";
    const PAGE_SIZE = 48;
    const views = ["cards", "list", "table"];
    const validSorts = ["relevance", "name", "distance", "duration", "difficulty", "elevation_gain", "elevation_loss", "like_count", "created", "date"];
    type SearchSnapshot = { filter: TrailFilter };

    function restoreFilter() {
        if (!browser || ["author", "category", "subcategory"].some(key => page.url.searchParams.has(key))) return page.data.filter;
        try {
            return sanitizeTrailFilter(JSON.parse(localStorage.getItem(FILTER_STORAGE_KEY) ?? "null"), page.data.filter);
        } catch { return page.data.filter; }
    }
    function applySavedSort(value: TrailFilter): TrailFilter {
        try {
            const sort = localStorage.getItem("trailSearchSort") ?? localStorage.getItem("sort");
            const order = localStorage.getItem("trailSearchSortOrder") ?? localStorage.getItem("sort_order");
            return { ...value, sort: validSorts.includes(sort ?? "") ? sort as TrailFilter["sort"] : "relevance",
                sortOrder: order === "+" || order === "-" ? order : page.data.filter.sortOrder };
        } catch { return { ...value, sort: "relevance", sortOrder: page.data.filter.sortOrder }; }
    }

    let filter: TrailFilter = $state(restoreFilter());
    let currentDisplayMode = $state("cards");
    let mounted = false;
    let searchBarComponent: TrailSearchBar | undefined = $state();
    let filterPanelTarget = $state<HTMLDivElement>();
    let wideSearchLayout = $state(false);
    let searchPosition = $derived(searchPositionForViewport($searchLayout.position, wideSearchLayout));
    let loading = $state(true);
    let refreshing = $state(false);
    let loadingMore = $state(false);
    let searchError = $state(false);
    let moreError = $state(false);
    let trails: Trail[] = $state([]);
    let pagination = $state({ page: 1, totalPages: 1, items: PAGE_SIZE });
    let loadedSearchKey = $state("");
    let hasMore = $derived(pagination.page < pagination.totalPages && loadedSearchKey === searchKey());
    let searchController: AbortController | undefined;
    let widthPreferences: Record<string, boolean> = $state({});
    let trailsFullWidth = $derived(widthPreferences[currentDisplayMode] ?? false);

    export const snapshot: Snapshot<SearchSnapshot | TrailFilter> = {
        capture: () => JSON.parse(JSON.stringify({ filter })),
        restore: value => {
            const restored = sanitizeTrailFilter("filter" in value ? value.filter : value, page.data.filter);
            filter = mounted ? { ...restored, sort: filter.sort, sortOrder: filter.sortOrder } : restored;
            searchBarComponent?.restoreFilter(filter);
            if (mounted) void searchResults();
        },
    };

    onMount(() => {
        filter = applySavedSort(filter);
        try {
            const saved = localStorage.getItem("displayOption");
            if (saved && views.includes(saved)) currentDisplayMode = saved;
            const stored = JSON.parse(localStorage.getItem("trailWidthPreferences") ?? "{}");
            if (stored && typeof stored === "object") widthPreferences = stored;
        } catch { /* Use the default display when storage is unavailable. */ }
        mounted = true;
        loadSearchLayout();
        const searchWidth = window.matchMedia("(min-width: 1024px)");
        const updateSearchWidth = () => { wideSearchLayout = searchWidth.matches; };
        updateSearchWidth();
        searchWidth.addEventListener("change", updateSearchWidth);
        void searchResults();
        return () => {
            mounted = false;
            searchController?.abort();
            searchController = undefined;
            searchWidth.removeEventListener("change", updateSearchWidth);
        };
    });

    beforeNavigate(({ to }) => {
        if (!browser || !to?.url || to.url.pathname.startsWith("/trails") || to.url.pathname === "/map") return;
        try { localStorage.removeItem(FILTER_STORAGE_KEY); } catch { /* Storage may be unavailable. */ }
    });

    function searchKey() {
        return JSON.stringify(filter);
    }
    function persistSearch() {
        try {
            localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify(filter));
            localStorage.setItem("trailSearchSort", filter.sort);
            localStorage.setItem("trailSearchSortOrder", filter.sortOrder);
        } catch { /* Keep searching when storage is unavailable. */ }
    }
    function selectDisplay(mode: string) {
        if (!views.includes(mode)) return;
        currentDisplayMode = mode;
        try { localStorage.setItem("displayOption", mode); } catch { /* Keep the current view. */ }
    }
    function toggleWidth() {
        widthPreferences = { ...widthPreferences, [currentDisplayMode]: !trailsFullWidth };
        try { localStorage.setItem("trailWidthPreferences", JSON.stringify(widthPreferences)); } catch { /* Keep the current width. */ }
    }
    async function searchResults(reset = true, preservePages = false) {
        searchController?.abort();
        const controller = new AbortController();
        searchController = controller;
        const currentFilter = $state.snapshot(filter);
        const key = searchKey();
        const nextPage = reset ? 1 : pagination.page + 1;
        if (reset) {
            refreshing = true;
            loading = !loadedSearchKey;
            loadingMore = false;
            searchError = false;
        }
        moreError = false;
        persistSearch();
        try {
            const fetcher: typeof fetch = (url, options) => fetch(url, { ...options, signal: controller.signal });
            const result = preservePages
                ? await trails_search_filter_pages(currentFilter, pagination.page, PAGE_SIZE, fetcher)
                : await trails_search_filter(currentFilter, nextPage, PAGE_SIZE, fetcher);
            if (controller !== searchController || key !== searchKey()) return;
            const scrollY = preservePages ? window.scrollY : undefined;
            if (reset) trails = result.items;
            else {
                const existing = new Set(trails.map(trail => trail.id));
                trails = [...trails, ...result.items.filter(trail => !existing.has(trail.id))];
            }
            pagination = { page: result.page, totalPages: result.totalPages, items: PAGE_SIZE };
            loadedSearchKey = key;
            if (scrollY !== undefined) {
                await tick();
                if (controller === searchController) window.scrollTo({ top: scrollY });
            }
        } catch {
            if (controller === searchController) {
                if (reset) searchError = true;
                else moreError = true;
            }
        } finally {
            if (controller === searchController) {
                loading = false;
                refreshing = false;
                loadingMore = false;
            }
        }
    }
    function loadMore() {
        if (loading || refreshing || loadingMore || !hasMore) return;
        loadingMore = true;
        void searchResults(false);
    }
    function applySearch(next: TrailFilter) {
        filter = next;
        void searchResults();
    }
</script>

<svelte:head><title>{$_("trail", { values: { n: 2 } })} | wanderer</title></svelte:head>

<main class="trail-search-layout grid grid-cols-1 items-start gap-3 max-w-7xl" class:full-width={trailsFullWidth}>
    <TrailList bind:filter bind:trails bind:pagination bind:display={currentDisplayMode} {loading}
        onupdate={() => searchResults()} onresultsupdate={() => searchResults(true, true)} busy={loading || refreshing} ondisplaychange={selectDisplay}
        managed incremental compactControls inlineControls fullWidthCards
        fullWidth={trailsFullWidth} onwidthtoggle={toggleWidth} {moreResults}
        bind:filterPanelTarget {searchPosition} searchExpansion={$searchLayout.expansion}
        onsearchposition={wideSearchLayout ? position => updateSearchLayout({ position }) : undefined}
        onsearchexpansion={searchPosition === "top" ? expansion => updateSearchLayout({ expansion }) : undefined}>
        {#snippet searchBar()}
            <TrailSearchBar bind:this={searchBarComponent} categories={page.data.categories} subcategories={page.data.subcategories}
                bind:filter defaults={page.data.filter} onapply={applySearch} inlineControls {filterPanelTarget}
                layout={searchPosition === "top" ? $searchLayout.expansion : "sidebar"}
                filterSections={page.data.settings?.behavior?.searchFilterSections} />
            {#if searchError}<p role="alert" class="search-message">{$_("trail-search-error")} <button type="button" onclick={() => searchResults()}>{$_("trail-search-retry")}</button></p>{/if}
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
    .trail-search-layout { --trail-controls-inset: .6rem; --trail-controls-top: .3125rem; margin-inline: auto; min-width: 0; position: relative; width: calc(100% - 3rem); }
    .trail-search-layout.full-width { max-width: none; }
    .load-more { align-items: center; display: flex; flex-direction: column; gap: .5rem; padding: 1rem 0 1.5rem; }
    .load-more-button { border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; display: inline-flex; font-size: .9rem; gap: .6rem; justify-content: center; min-width: 10rem; padding: .7rem 1.3rem; }
    .load-more-button:hover { background: rgba(var(--input-background)); }
    .load-more-button:disabled { cursor: wait; opacity: .6; }
    .search-message { font-size: .8rem; }
    .search-message button { text-decoration: underline; }
    @media (max-width: 639px) {
        .trail-search-layout { --trail-controls-inset: .4rem; width: calc(100% - 1.5rem); }
    }
</style>
