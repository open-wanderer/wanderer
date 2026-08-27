<script lang="ts">
    import { browser } from "$app/environment";
    import { beforeNavigate, goto } from "$app/navigation";
    import { page } from "$app/state";
    import TrailFilterPanel from "$lib/components/trail/trail_filter_panel.svelte";
    import TrailFilterPreview from "$lib/components/trail/trail_filter_preview.svelte";
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
        if (
            page.url.searchParams.has("category") ||
            page.url.searchParams.has("subcategory") ||
            page.url.searchParams.has("author")
        ) {
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
        localStorage.setItem(
            TRAIL_LIST_FILTER_STORAGE_KEY,
            JSON.stringify(filter),
        );
    }

    let filterExpanded: boolean = $state(true);
    let showFilterConcept = $state(true);
    let filterConceptExpanded = $state(true);

    let loading: boolean = $state(true);

    let filter: TrailFilter = $state(restoreStoredFilter(page.data.filter));
    const pagination: { page: number; totalPages: number; items: number } =
        $state({
            page: page.url.searchParams.has("page")
                ? parseInt(page.url.searchParams.get("page")!)
                : 1,
            totalPages: 1,
            items: 25,
        });
    let trails: Trail[] = $state([]);
    let trailsFullWidth = $state(false);
    let widthPreferences: Record<string, boolean> = $state({});
    let currentDisplayMode: string = $state("cards");
    const widthPreferenceStorageKey = "trailWidthPreferences";

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
            filterConceptExpanded = false;
        }
        loadWidthPreferences();
    });

    function loadWidthPreferences() {
        if (typeof localStorage === "undefined") {
            return;
        }
        try {
            const stored = localStorage.getItem(widthPreferenceStorageKey);
            if (stored) {
                const parsed = JSON.parse(stored);
                if (parsed && typeof parsed === "object") {
                    widthPreferences = parsed;
                }
            }
        } catch (err) {
            console.warn("Failed to parse trail width preferences", err);
        }
        applyWidthPreference(currentDisplayMode);
    }

    function applyWidthPreference(mode: string) {
        trailsFullWidth = widthPreferences[mode] ?? false;
    }

    function persistWidthPreferences() {
        if (typeof localStorage === "undefined") {
            return;
        }
        try {
            localStorage.setItem(
                widthPreferenceStorageKey,
                JSON.stringify(widthPreferences),
            );
        } catch (err) {
            console.warn("Failed to persist trail width preferences", err);
        }
    }

    function updateWidthPreference(mode: string, value: boolean) {
        widthPreferences = { ...widthPreferences, [mode]: value };
        persistWidthPreferences();
    }

    function handleDisplayModeChange(mode: string) {
        currentDisplayMode = mode;
        applyWidthPreference(mode);
    }

    function toggleTrailWidth() {
        trailsFullWidth = !trailsFullWidth;
        updateWidthPreference(currentDisplayMode, trailsFullWidth);
    }

    function showPreviewResults() {
        document.getElementById("trails")?.scrollIntoView({
            behavior: "smooth",
            block: "start",
        });
    }

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

    async function handleFilterUpdate(resetPagination: boolean = true) {
        loading = true;
        persistFilter();

        await paginate(resetPagination ? 1 : pagination.page, pagination.items);

        loading = false;
    }

    async function paginate(
        newPage: number,
        items: number,
        scrollToTop: boolean = true,
    ) {
        pagination.page = newPage;

        try {
            await doPaginate(newPage, items);
        } catch (err: any) {
            let apiError: APIError = err;
            if (apiError.status == 413) {
                // content too large

                let newItems = 10;

                if (items == 12 || items == 24 || items == 48 || items == 96) {
                    // cards view
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
        goto(`?${page.url.searchParams.toString()}`, {
            keepFocus: true,
            noScroll: !scrollToTop,
        });
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
    class={`grid grid-cols-1 md:grid-cols-[360px_1fr] items-start gap-8 mx-6 ${trailsFullWidth ? "md:mx-6 max-w-full" : "md:mx-auto max-w-7xl"}`}
>
    <div class="min-w-0 space-y-2">
        <div
            class="grid grid-cols-2 rounded-xl border border-input-border bg-input-background p-1"
            role="group"
            aria-label={$_("filter-preview-view")}
        >
            <button
                type="button"
                class="rounded-lg px-3 py-2 text-sm transition-all"
                class:bg-background={!showFilterConcept}
                class:font-semibold={!showFilterConcept}
                class:shadow-sm={!showFilterConcept}
                aria-pressed={!showFilterConcept}
                onclick={() => (showFilterConcept = false)}
            >
                {$_("filter-preview-current")}
            </button>
            <button
                type="button"
                class="rounded-lg px-3 py-2 text-sm transition-all"
                class:bg-background={showFilterConcept}
                class:font-semibold={showFilterConcept}
                class:shadow-sm={showFilterConcept}
                aria-pressed={showFilterConcept}
                onclick={() => (showFilterConcept = true)}
            >
                <i class="fa fa-flask mr-1.5 text-gray-500"></i>
                {$_("filter-preview-concept")}
            </button>
        </div>

        {#if showFilterConcept}
            <button
                type="button"
                class="flex w-full items-center justify-between rounded-xl border border-input-border bg-input-background px-4 py-3 text-sm font-semibold md:hidden"
                aria-expanded={filterConceptExpanded}
                onclick={() => (filterConceptExpanded = !filterConceptExpanded)}
            >
                <span>
                    <i class="fa fa-sliders mr-2 text-gray-500"></i>
                    {$_("filter-preview-title")}
                </span>
                <i
                    class="fa fa-chevron-down text-xs transition-transform"
                    class:rotate-180={filterConceptExpanded}
                ></i>
            </button>
        {/if}

        <div
            class:hidden={!showFilterConcept}
            class:mobile-filter-collapsed={!filterConceptExpanded}
        >
            <TrailFilterPreview
                categories={page.data.categories}
                subcategories={page.data.subcategories}
                onapply={showPreviewResults}
            />
        </div>
        <div class:hidden={showFilterConcept}>
            <TrailFilterPanel
                categories={page.data.categories}
                bind:filter
                {filterExpanded}
                onupdate={() => handleFilterUpdate()}
            ></TrailFilterPanel>
        </div>
    </div>
    <div class="min-w-0">
        {#if showFilterConcept}
            <p
                class="mb-4 flex items-center gap-2 rounded-xl border border-dashed border-input-border bg-input-background px-4 py-3 text-sm text-gray-500"
            >
                <i class="fa fa-flask"></i>
                {$_("filter-preview-results-unchanged")}
            </p>
        {/if}
        <TrailList
            bind:filter
            {loading}
            bind:trails
            {pagination}
            onupdate={() => handleFilterUpdate(false)}
            onpagination={paginate}
            ondisplaychange={handleDisplayModeChange}
        >
            {#snippet trailWidthToggleSnippet()}
                <button
                    type="button"
                    class="btn-icon"
                    onclick={toggleTrailWidth}
                    aria-pressed={trailsFullWidth}
                    aria-label={trailsFullWidth
                        ? $_("collapse-trail-list")
                        : $_("expand-trail-list")}
                    title={trailsFullWidth
                        ? $_("collapse-trail-list")
                        : $_("expand-trail-list")}
                >
                    <i
                        class="fa {trailsFullWidth ? 'fa-compress' : 'fa-expand'}"
                    ></i>
                </button>
            {/snippet}
        </TrailList>
    </div>
</main>

<style>
    @media (max-width: 767px) {
        .mobile-filter-collapsed {
            display: none;
        }
    }
</style>
