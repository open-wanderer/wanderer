<script lang="ts">
    import type { SearchPosition, SearchExpansion } from "$lib/models/search_layout";
    import type { Trail, TrailFilter } from "$lib/models/trail";
    import { _ } from "svelte-i18n";
    import Pagination from "../base/pagination.svelte";
    import Select, { type SelectItem } from "../base/select.svelte";
    import EmptyStateSearch from "../empty_states/empty_state_search.svelte";
    import TrailCard from "./trail_card.svelte";
    import TrailListItem from "./trail_list_item.svelte";
    import TrailTable from "./trail_table.svelte";
    import SkeletonCard from "../base/skeleton_card.svelte";
    import SkeletonListItem from "../base/skeleton_list_item.svelte";
    import { onMount, tick } from "svelte";
    import type { Snippet } from "svelte";
    import TrailDropdown from "$lib/components/trail/trail_dropdown.svelte";
    import TrailResultControls from "./trail_result_controls.svelte";

    interface Props {
        filter?: TrailFilter | null;
        trails: Trail[];
        pagination?: { page: number; totalPages: number; items: number };
        loading?: boolean;
        fullWidthCards?: boolean;
        onupdate?: (
            filter: TrailFilter | null,
            selection: Set<Trail> | undefined,
        ) => void;
        onpagination?: (page: number, items: number) => void;
        ondisplaychange?: (display: string) => void;
        trailWidthToggleSnippet?: Snippet;
        compactControls?: boolean;
        inlineControls?: boolean;
        fullWidth?: boolean;
        onwidthtoggle?: () => void;
        display?: string;
        managed?: boolean;
        incremental?: boolean;
        searchBar?: Snippet;
        mapView?: Snippet;
        moreResults?: Snippet;
        filterPanelTarget?: HTMLDivElement;
        searchPosition?: SearchPosition;
        searchExpansion?: SearchExpansion;
        onsearchposition?: (position: SearchPosition) => void;
        onsearchexpansion?: (expansion: SearchExpansion) => void;
    }

    let {
        filter = $bindable(null),
        trails = $bindable([]),
        pagination = $bindable({
            page: 1,
            totalPages: 1,
            items: 25,
        }),
        loading = false,
        fullWidthCards = false,
        onupdate,
        onpagination,
        ondisplaychange,
        trailWidthToggleSnippet: trailWidthToggleSnippet,
        compactControls = false,
        inlineControls = false,
        fullWidth = false,
        onwidthtoggle,
        display = $bindable("cards"),
        managed = false,
        incremental = false,
        searchBar,
        mapView,
        moreResults,
        filterPanelTarget = $bindable(),
        searchPosition = "top",
        searchExpansion = "overlay",
        onsearchposition,
        onsearchexpansion,
    }: Props = $props();

    const displayOptions: SelectItem[] = $derived([
        { text: $_("card", { values: { n: 2 } }), value: "cards" },
        { text: $_("list", { values: { n: 1 } }), value: "list" },
        { text: $_("table"), value: "table" },
        ...(mapView ? [{ text: $_("map"), value: "map" }] : []),
    ]);

    const perPageOptions: SelectItem[] = [
        { text: "10", value: 10 },
        { text: "25", value: 25 },
        { text: "50", value: 50 },
        { text: "100", value: 100 },
    ];

    const perPageOptionsCards: SelectItem[] = [
        { text: "12", value: 12 },
        { text: "24", value: 24 },
        { text: "48", value: 48 },
        { text: "96", value: 96 },
    ];


    let selection: Set<Trail> | undefined = $state();
    let hoveredTrail: Trail | undefined = $state();
    let mapVisited = $state(false);
    $effect(() => { if (display === "map") mapVisited = true; });

    function notifyDisplayChange() {
        ondisplaychange?.(display);
    }

    function roundItems(items: number, display: string): number {
        const cards = display === "cards";
        if (items > 50) {
            return cards ? 96 : 100;
        }
        if (items > 25) {
            return cards ? 48 : 50;
        }
        if (items > 12) {
            return cards ? 24 : 25;
        }
        return cards ? 12 : 10;
    }

    const sortOptions: SelectItem[] = $derived([
        ...(managed && filter?.q.trim() ? [{ text: $_("trail-results-relevance"), value: "relevance" }] : []),
        { text: $_("name"), value: "name" },
        { text: $_("distance"), value: "distance" },
        { text: $_("duration"), value: "duration" },
        { text: $_("difficulty"), value: "difficulty" },
        { text: $_("elevation-gain"), value: "elevation_gain" },
        { text: $_("elevation-loss"), value: "elevation_loss" },
        { text: $_("likes"), value: "like_count" },
        { text: $_("creation-date"), value: "created" },
        { text: $_("date"), value: "date" },
    ]);

    onMount(() => {
        if (managed) return;
        const storedDisplayOption = localStorage.getItem("displayOption");
        const storedSort = localStorage.getItem("sort");
        const storedSortOrder = localStorage.getItem("sort_order");
        const paginationItems = localStorage.getItem("paginationItems");

        if (storedDisplayOption) {
            display = storedDisplayOption;
        }
        if (filter) {
            filter.sort =
                (storedSort as typeof filter.sort | null) ?? filter.sort;
        }
        if (filter) {
            filter.sortOrder =
                (storedSortOrder as typeof filter.sortOrder | null) ??
                filter.sortOrder;
        }
        if (paginationItems || compactControls) {
            pagination.items = roundItems(paginationItems ? +paginationItems : pagination.items, display);
            localStorage.setItem("paginationItems", pagination.items.toString());
        }
        onupdate?.(filter, selection);
        notifyDisplayChange();
    });

    function setDisplayOption() {
        if (managed) { notifyDisplayChange(); return; }
        localStorage.setItem("displayOption", display);

        const previousItems = pagination.items;
        pagination.items = roundItems(previousItems, display);

        if (pagination.items !== previousItems) {
            setItemsPerPage();
        }
        notifyDisplayChange();
    }

    function setSort() {
        if (!filter) {
            return;
        }
        localStorage.setItem(managed ? "trailSearchSort" : "sort", filter.sort);
        onupdate?.(filter, selection);
    }

    function setMenuSort(sort: string, order: "+" | "-") {
        if (!filter) return;
        filter.sort = sort as TrailFilter["sort"];
        filter.sortOrder = order;
        localStorage.setItem(managed ? "trailSearchSortOrder" : "sort_order", order);
        setSort();
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
        localStorage.setItem(managed ? "trailSearchSortOrder" : "sort_order", filter.sortOrder);
        onupdate?.(filter, selection);
    }

    function handleSortUpdate(sort: any) {
        if (!filter) {
            return;
        }
        if (sort === filter.sort) {
            setSortOrder();
        } else {
            filter.sort = sort;
            filter.sortOrder = "+";
            setSort();
        }
    }

    function isHovered(trail: Trail): boolean {
        if (trail === undefined) {
            return false;
        }

        if (hoveredTrail === undefined) {
            return false;
        }

        return hoveredTrail.id === trail.id;
    }

    function isSelected(trail: Trail): boolean {
        if (trail === undefined) {
            return false;
        } else {
            if (selection === undefined) {
                return false;
            } else {
                for (const sTrail of selection) {
                    if (sTrail !== undefined && sTrail.id === trail.id) {
                        return true;
                    }
                }
            }
        }

        return false;
    }

    function handleSelectionUpdate(trail: Trail) {
        let newSelection = new Set<Trail>();

        if (trail !== undefined) {
            let isSelected = false;

            if (selection !== undefined && selection.size > 0) {
                for (const sTrail of selection) {
                    if (sTrail !== undefined && sTrail.id === trail.id) {
                        isSelected = true;
                        continue;
                    }

                    newSelection.add(sTrail);
                }
            }

            if (!isSelected) {
                newSelection.add(trail);
            }
        } else if (
            selection === undefined ||
            selection.size === 0 ||
            (trails !== undefined && selection.size !== trails.length)
        ) {
            for (const eTrail of trails) {
                newSelection.add(eTrail);
            }
        }

        selection = newSelection;
    }

    function handleHoverUpdate(hTrail: Trail) {
        if (hTrail === undefined) {
            return;
        }

        if (hoveredTrail === undefined) hoveredTrail = hTrail;
        else hoveredTrail = undefined;
    }

    async function handleTrailsMergeDone(resetSelection: boolean = false) {
        if (resetSelection) {
            selection?.clear();
            hoveredTrail = undefined;
        }
        await tick();
        onupdate?.(filter, selection);
    }

    async function handleTrailsEditDone(
        resetSelection: boolean = false,
        updatedTrails: Trail[] = [],
    ) {
        if (updatedTrails.length > 0) {
            const updatedByID = new Map(
                updatedTrails
                    .filter((trail) => trail.id)
                    .map((trail) => [trail.id, trail]),
            );
            // Keep the richer expand fields from the existing (search-derived)
            // trail (e.g. author, likes) that trails_update does not return,
            // and only overlay the changed values and their expands.
            const mergeUpdated = (trail: Trail) => {
                const updated = updatedByID.get(trail.id);
                if (!updated) return trail;
                return {
                    ...trail,
                    ...updated,
                    expand: { ...trail.expand, ...updated.expand },
                };
            };
            trails = trails.map(mergeUpdated);
            if (resetSelection) {
                selection = new Set<Trail>();
                hoveredTrail = undefined;
            } else if (selection) {
                selection = new Set([...selection].map(mergeUpdated));
            }
            // The updated trails already carry the fresh data; skip the refetch
            // because the search index is updated asynchronously and would
            // return stale categories, overwriting the local update.
            return;
        }
        if (resetSelection) {
            selection = new Set<Trail>();
            hoveredTrail = undefined;
        }
        await tick();
        onupdate?.(filter, selection);
    }

    function handleMouseEnter(trail: Trail) {
        handleHoverUpdate(trail);
    }
    function handleMouseLeave(trail: Trail) {
        handleHoverUpdate(trail);
    }

    function handleTrailClick(e: Event, trail: Trail) {
        if (selection && selection.size > 0) {
            e.stopPropagation();
            e.preventDefault();
            handleSelectionUpdate(trail);
        } 
        return true
    }

    function setItemsPerPage() {
        localStorage.setItem("paginationItems", pagination.items.toString());
        onpagination?.(1, pagination.items);
    }
</script>

{#snippet selectionTrigger({ toggleMenu, loading }: { toggleMenu: (event: MouseEvent) => void; loading: boolean })}
    <button type="button" class="btn-primary selection-trigger" class:btn-disabled={loading}
        aria-label={`${selection?.size} ${$_("selected")}`} aria-busy={loading} disabled={loading} onclick={toggleMenu}>
        <span>{selection?.size}</span><span class="selection-copy">{$_("selected")}</span>
        <i class={`fa ${loading ? "fa-spinner fa-spin" : "fa-caret-down"}`} aria-hidden="true"></i>
    </button>
{/snippet}

{#snippet selectionMenu()}
    {#if selection && selection.size > 0}
        <div class="selection-actions">
            <TrailDropdown
                trails={selection}
                mode="multi-select"
                toggle={searchBar ? selectionTrigger : undefined}
                onDelete={() => handleTrailsEditDone(true)}
                onShare={() => handleTrailsEditDone(false)}
                onMerge={() => handleTrailsMergeDone(true)}
                onUpdate={(updatedTrails) => handleTrailsEditDone(true, updatedTrails)}
            />
        </div>
    {/if}
{/snippet}

<div class="min-w-0" class:search-side={Boolean(searchBar) && searchPosition !== "top"} class:search-right={searchPosition === "right"}>
    {#if compactControls}
        <div class:search-toolbar={Boolean(searchBar)}>
            <div class:search-field={Boolean(searchBar)}>
                {#if searchBar}{@render searchBar()}{/if}
                <TrailResultControls
                    buttonsOnly={inlineControls}
                    sort={filter?.sort === "relevance" && !filter.q.trim() ? "created" : filter?.sort}
                    order={filter?.sortOrder}
                    {sortOptions}
                    display={display}
                    {displayOptions}
                    pageSize={pagination.items}
                    pageSizes={display === "cards" ? perPageOptionsCards : perPageOptions}
                    wide={fullWidth}
                    onsort={filter ? setMenuSort : undefined}
                    ondisplay={(mode) => { display = mode; setDisplayOption(); }}
                    onpagesize={incremental ? undefined : (size) => { pagination.items = size; setItemsPerPage(); }}
                    {onwidthtoggle}
                    {searchPosition} {searchExpansion} {onsearchposition} {onsearchexpansion}
                >
                    {#snippet selectionActions()}
                        {#if !searchBar}{@render selectionMenu()}{/if}
                    {/snippet}
                </TrailResultControls>
            </div>
            {#if searchBar}{@render selectionMenu()}{/if}
        </div>
    {:else}
    <div class="flex items-end flex-wrap lg:flex-nowrap gap-x-6 gap-y-2 mx-4">
        <div class="basis-full order-1 md:order-0">
            <Pagination
                page={pagination.page}
                totalPages={pagination.totalPages}
                perPage={pagination.items}
                {onpagination}
            ></Pagination>
        </div>
        {#if selection !== undefined && selection.size > 0}
            <div class="flex relative shrink-0">
                <TrailDropdown
                    trails={selection}
                    mode={"multi-select"}
                    onDelete={() => handleTrailsEditDone(true)}
                    onShare={() => handleTrailsEditDone(false)}
                    onMerge={() => handleTrailsMergeDone(true)}
                    onUpdate={(updatedTrails) =>
                        handleTrailsEditDone(true, updatedTrails)}
                />
            </div>
        {/if}
        {#if filter}
            <div class="shrink-0">
                {#if display !== "table"}
                    <p class="text-sm text-gray-500 pb-2">{$_("sort")}</p>
                    <div class="flex items-center gap-2">
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
                    </div>
                {/if}
            </div>
        {/if}
        <div class="shrink-0">
            <p class="text-sm text-gray-500 pb-2">{$_("display-as")}</p>
            <div class="flex items-center gap-2">
                <Select
                    bind:value={display}
                    items={displayOptions}
                    onchange={() => setDisplayOption()}
                ></Select>
                {#if trailWidthToggleSnippet}
                    {@render trailWidthToggleSnippet?.()}
                {/if}
            </div>
        </div>
    </div>
    {/if}

    <div class="filter-sidebar" bind:this={filterPanelTarget} hidden={searchPosition === "top" || !searchBar}></div>
    <div class="result-content">
    {#if mapView && (display === "map" || mapVisited)}
        <div hidden={display !== "map"} class:compact-results={compactControls} class:inline-results={inlineControls}>{@render mapView()}</div>
    {/if}
    {#if display !== "map" || !mapView}
    <div id="trails" class="flex items-start flex-wrap gap-8 py-8 max-w-full" class:compact-results={compactControls} class:inline-results={inlineControls} class:card-results={managed && display === "cards"} class:list-results={managed && display === "list"}>
        {#if loading}
            {#if display === "table"}
                <TrailTable
                    trails={null}
                    selection={new Set<Trail>()}
                    tableHeader={sortOptions.filter(option => option.value !== "elevation_loss" && option.value !== "relevance")}
                    items={pagination.items}
                ></TrailTable>
            {:else}
                {#each { length: pagination.items } as _, index}
                    {#if display === "cards"}
                        <div class="flex-1">
                            <SkeletonCard></SkeletonCard>
                        </div>
                    {:else if display === "list"}
                        <SkeletonListItem></SkeletonListItem>
                    {/if}
                {/each}
            {/if}
        {:else}
            {#if trails.length == 0}
                <div class="flex flex-col basis-full items-center">
                    <EmptyStateSearch width={356}></EmptyStateSearch>
                </div>
            {/if}
            {#if display === "table"}
                <TrailTable
                    {trails}
                    {selection}
                    tableHeader={sortOptions.filter(
                        (option) => option.value !== "elevation_loss" && option.value !== "relevance",
                    )}
                    {filter}
                    items={pagination.items}
                    onsort={handleSortUpdate}
                    onTrailSelect={(t) => handleSelectionUpdate(t)}
                ></TrailTable>
            {:else}
                {#each trails as trail}
                    <a
                        class="max-w-full flex-1"
                        class:basis-full={display === "list"}
                        onclick={(e) => handleTrailClick(e,trail)}
                        href="/trail/view/@{trail.author}{trail.domain
                            ? `@${trail.domain}`
                            : ''}/{trail.id}"
                        onmouseenter={(e) => handleMouseEnter(trail)}
                        onmouseleave={(e) => handleMouseLeave(trail)}
                    >
                        {#if display === "cards"}
                            <TrailCard
                                fullWidth={fullWidthCards}
                                {trail}
                                selected={isSelected(trail)}
                                hovered={isHovered(trail)}
                                onTrailSelect={() =>
                                    handleSelectionUpdate(trail)}
                            ></TrailCard>
                        {:else}
                            <TrailListItem
                                {trail}
                                selected={isSelected(trail)}
                                hovered={isHovered(trail)}
                                onTrailSelect={() =>
                                    handleSelectionUpdate(trail)}
                            ></TrailListItem>
                        {/if}
                    </a>
                {/each}
            {/if}
        {/if}
    </div>
    {#if moreResults}{@render moreResults()}{/if}
    {/if}
    {#if !incremental && (!compactControls || pagination.totalPages > 1)}
    <div class="flex items-start flex-wrap lg:flex-nowrap gap-x-6 gap-y-2 mx-4">
        <div class="basis-full order-1 md:order-0">
            <Pagination
                page={pagination.page}
                totalPages={pagination.totalPages}
                perPage={pagination.items}
                {onpagination}
            ></Pagination>
        </div>
        {#if !compactControls}<div class="shrink-0">
            <Select
                bind:value={pagination.items}
                items={display == "cards"
                    ? perPageOptionsCards
                    : perPageOptions}
                onchange={setItemsPerPage}
            ></Select>
        </div>{/if}
    </div>
    {/if}
    </div>
</div>

<style>
    .result-content { min-width: 0; }
    .search-side { display: grid; grid-template-columns: clamp(18rem, 26vw, 23rem) minmax(0, 1fr); align-items: start; gap: 1rem; }
    .search-side .search-toolbar { grid-column: 1 / -1; grid-row: 1; margin: 0; }
    .filter-sidebar { min-width: 0; }
    .search-side .filter-sidebar { grid-column: 1; grid-row: 2; }
    .search-side .result-content { grid-column: 2; grid-row: 2; }
    .search-right { grid-template-columns: minmax(0, 1fr) clamp(18rem, 26vw, 23rem); }
    .search-right .filter-sidebar { grid-column: 2; }
    .search-right .result-content { grid-column: 1; }
    .search-toolbar { align-items: flex-start; display: flex; gap: .75rem; margin-bottom: .75rem; min-width: 0; }
    .search-field { flex: 1; min-width: 0; position: relative; }
    .selection-actions { flex-shrink: 0; }
    .selection-actions:not(:has(> :global(.dropdown))) { display: none; }
    .selection-trigger { align-items: center; display: inline-flex; font-weight: 500; gap: .4rem; height: 3.125rem; justify-content: center; min-width: 3.125rem; white-space: nowrap; }
    /* Keep mouseup on the trigger so its dropdown can toggle closed. */
    .selection-trigger > * { pointer-events: none; }
    @media (max-width: 639px) {
        .selection-copy { display: none; }
        .selection-trigger { padding-inline: .65rem; }
    }
    .compact-results { padding-top: 1rem; }
    .inline-results { padding-top: .25rem; }
    .search-side .inline-results { padding-top: 0; }
    .card-results, .list-results { row-gap: 1rem; }
    .card-results {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(min(100%, 18rem), 1fr));
    }
    .card-results > .basis-full { grid-column: 1 / -1; }
    #sort-order-btn {
        transition: transform 0.5s ease;
    }
    :global(.rotated) {
        transform: rotate(180deg);
    }
</style>
