<script lang="ts">
    import { browser } from "$app/environment";
    import { beforeNavigate, goto } from "$app/navigation";
    import { page } from "$app/state";
    import TrailFilterPanel from "$lib/components/trail/trail_filter_panel.svelte";
    import TrailList from "$lib/components/trail/trail_list.svelte";
    import type { Trail, TrailFilter } from "$lib/models/trail";
    import { trails_search_filter } from "$lib/stores/trail_store";
    import type { Snapshot } from "@sveltejs/kit";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";
    import { APIError } from "$lib/util/api_util";
    import { sanitizeTrailFilter } from "$lib/util/trail_filter_util";

    const TRAIL_LIST_FILTER_STORAGE_KEY = "trailListFilter";

    function restoreStoredFilter(defaultFilter: TrailFilter): TrailFilter {
        if (!browser) {
            return defaultFilter;
        }

        const stored = localStorage.getItem(TRAIL_LIST_FILTER_STORAGE_KEY);
        if (!stored) {
            return defaultFilter;
        }

        try {
            const parsed = JSON.parse(stored) as Record<string, unknown>;
            return sanitizeTrailFilter(parsed, defaultFilter);
        } catch {
            localStorage.removeItem(TRAIL_LIST_FILTER_STORAGE_KEY);
            return defaultFilter;
        }
    }

    function persistFilter() {
        if (!browser) {
            return;
        }
        localStorage.setItem(TRAIL_LIST_FILTER_STORAGE_KEY, JSON.stringify(filter));
    }

    let filterExpanded: boolean = $state(true);

    let loading: boolean = $state(true);

    let filter: TrailFilter = $state(restoreStoredFilter(page.data.filter));
    const pagination: { page: number; totalPages: number, items: number } = $state({
        page: page.url.searchParams.has("page")
            ? parseInt(page.url.searchParams.get("page")!)
            : 1,
        totalPages: 1,
        items: 25,
    });
    let trails: Trail[] = $state([]);

    export const snapshot: Snapshot<TrailFilter> = {
        capture: () => filter,
        restore: (value) => {
            filter = sanitizeTrailFilter(value, page.data.filter);
            handleFilterUpdate();
        },
    };

    onMount(() => {
        if (window.innerWidth < 768) {
            filterExpanded = false;
        }
    });

    beforeNavigate(({ to }) => {
        if (!browser || !to?.url) {
            return;
        }

        // Keep filter for in-page navigation (e.g. pagination on /trails).
        if (to.url.pathname.startsWith("/trails")) {
            return;
        }

        localStorage.removeItem(TRAIL_LIST_FILTER_STORAGE_KEY);
    });

    async function handleFilterUpdate() {
        loading = true;
        persistFilter();

        await paginate(1, pagination.items, false);

        loading = false;
    }

    async function paginate(newPage: number, items: number, scrollToTop: boolean = true) {
        pagination.page = newPage;

        try {
            await doPaginate(newPage, items);
        } catch (err: any) {
            let apiError : APIError = err;
            if (apiError.status == 413) { // content too large
                
                let newItems = 10;
                    
                if (items == 12 || items == 24 || items == 48 || items == 96) { // cards view
                    if (items > 96) {
                        newItems = 96;
                    } else if (items > 48) {
                        newItems = 48;
                    } else if (items > 24) {
                        newItems = 24;
                    } else {
                        newItems = 12;
                    }
                } else {
                    if (items > 100) {
                        newItems = 100;
                    } else if (items > 50) {
                        newItems = 50;
                    } else if (items > 25) {
                        newItems = 25;
                    } else {
                        newItems = 10;
                    }
                }
                    
                await doPaginate(newPage, newItems);
            }
        }
        
        page.url.searchParams.set("page", newPage.toString());
        goto(`?${page.url.searchParams.toString()}`, { keepFocus: true, noScroll: !scrollToTop });
    }

    async function doPaginate(newPage: number, items: number) {
        const response = await trails_search_filter(filter, newPage, items);
        if (items) {
            pagination.items = items;
        }
        trails = response.items;
        pagination.page = response.page;
        pagination.totalPages = response.totalPages;
    }
</script>

<svelte:head>
    <title>{$_("trail", { values: { n: 2 } })} | wanderer</title>
</svelte:head>

<main
    class="grid grid-cols-1 md:grid-cols-[300px_1fr] items-start gap-8 max-w-7xl mx-6 md:mx-auto"
>
    <TrailFilterPanel
        categories={page.data.categories}
        bind:filter
        {filterExpanded}
        onupdate={handleFilterUpdate}
    ></TrailFilterPanel>
    <TrailList
        bind:filter
        {loading}
        {trails}
        {pagination}
        onupdate={handleFilterUpdate}
        onpagination={paginate}
    ></TrailList>
</main>
