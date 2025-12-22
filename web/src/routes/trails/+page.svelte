<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import TrailFilterPanel from "$lib/components/trail/trail_filter_panel.svelte";
    import TrailList from "$lib/components/trail/trail_list.svelte";
    import type { Trail, TrailFilter, TrailSearchResult } from "$lib/models/trail";
    import { trails_search_filter } from "$lib/stores/trail_store";
    import type { Snapshot } from "@sveltejs/kit";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";
    import { APIError } from "$lib/util/api_util";

    let filterExpanded: boolean = $state(true);

    let loading: boolean = $state(true);

    let filter: TrailFilter = $state(page.data.filter);
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
            const difficultyMap: Record<string, 0 | 1 | 2> = {
                easy: 0,
                moderate: 1,
                difficult: 2,
            };
            // defensive copy
            const migrated = { ...value };

            if (Array.isArray(migrated.difficulty)) {
                migrated.difficulty = migrated.difficulty.map((d: any) => {
                    if (typeof d === "string" && d in difficultyMap) {
                        return difficultyMap[d];
                    }
                    return d;
                });
            }

            filter = migrated;
            handleFilterUpdate();
        },
    };

    onMount(() => {
        if (window.innerWidth < 768) {
            filterExpanded = false;
        }
    });

    async function handleFilterUpdate() {
        loading = true;

        await paginate(1, pagination.items);

        loading = false;
    }

    async function paginate(newPage: number, items: number) {
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
        goto(`?${page.url.searchParams.toString()}`);
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
