<script lang="ts">
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
    import TrailDropdown from "$lib/components/trail/trail_dropdown.svelte";

    interface Props {
        filter?: TrailFilter | null;
        trails: Trail[];
        pagination?: { page: number; totalPages: number, items: number };
        loading?: boolean;
        fullWidthCards?: boolean;
        onupdate?: (
            filter: TrailFilter | null,
            selection: Set<Trail> | undefined,
        ) => void;
        onpagination?: (page: number, items: number) => void;
    }

    let {
        filter = $bindable(null),
        trails,
        pagination = {
            page: 1,
            totalPages: 1,
            items: 25,
        },
        loading = false,
        fullWidthCards = false,
        onupdate,
        onpagination,
    }: Props = $props();

    const displayOptions: SelectItem[] = [
        { text: $_("card", { values: { n: 2 } }), value: "cards" },
        { text: $_("list", { values: { n: 1 } }), value: "list" },
        { text: $_("table"), value: "table" },
    ];
    
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

    let selectedDisplayOption = $state(displayOptions[0].value);

    let selection: Set<Trail> | undefined = $state();
    let hoveredTrail: Trail | undefined = $state();

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

    onMount(() => {
        const storedDisplayOption = localStorage.getItem("displayOption");
        const storedSort = localStorage.getItem("sort");
        const storedSortOrder = localStorage.getItem("sort_order");
        const paginationItems = localStorage.getItem("paginationItems");

        if (storedDisplayOption) {
            selectedDisplayOption = storedDisplayOption;
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
        if (paginationItems) {
            pagination.items = +paginationItems;
        
            let itemsChanged = false;
            if (selectedDisplayOption == "cards") {
                if (pagination.items > 50) {
                    pagination.items = 96;
                    itemsChanged = true;
                } else if (pagination.items > 25) {
                    pagination.items = 48;
                    itemsChanged = true;
                } else if (pagination.items > 12) {
                    pagination.items = 24;
                    itemsChanged = true;
                } else {
                    pagination.items = 12;
                    itemsChanged = true;
                }
            } else {
                if (pagination.items > 50) {
                    pagination.items = 100;
                    itemsChanged = true;
                } else if (pagination.items > 25) {
                    pagination.items = 50;
                    itemsChanged = true;
                } else if (pagination.items > 12) {
                    pagination.items = 25;
                    itemsChanged = true;
                } else {
                    pagination.items = 10;
                    itemsChanged = true;
                }
            }

            if (itemsChanged) {
                localStorage.setItem("paginationItems", pagination.items.toString());
            }
        }
        onupdate?.(filter, selection);
    });

    function setDisplayOption() {
        localStorage.setItem("displayOption", selectedDisplayOption);

        let itemsChanged = false;
        if (selectedDisplayOption == "cards") {
            if (pagination.items > 50) {
                pagination.items = 96;
                itemsChanged = true;
            } else if (pagination.items > 25) {
                pagination.items = 48;
                itemsChanged = true;
            } else if (pagination.items > 12) {
                pagination.items = 24;
                itemsChanged = true;
            } else {
                pagination.items = 12;
                itemsChanged = true;
            }
        } else {
            if (pagination.items > 50) {
                pagination.items = 100;
                itemsChanged = true;
            } else if (pagination.items > 25) {
                pagination.items = 50;
                itemsChanged = true;
            } else if (pagination.items > 12) {
                pagination.items = 25;
                itemsChanged = true;
            } else {
                pagination.items = 10;
                itemsChanged = true;
            }
        }

        if (itemsChanged) {
            localStorage.setItem("paginationItems", pagination.items.toString());
        }
    }

    function setSort() {
        if (!filter) {
            return;
        }
        localStorage.setItem("sort", filter.sort);
        onupdate?.(filter, selection);
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

    async function handleTrailsEditDone(resetSelection: boolean = false) {
        if (resetSelection) {
            selection?.clear();
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

    function setItemsPerPage() {
        localStorage.setItem("paginationItems", pagination.items.toString());
        onpagination?.(1, pagination.items);
    }
</script>

<div class="min-w-0">
    <div class="flex items-end flex-wrap lg:flex-nowrap gap-x-6 gap-y-2 mx-4">
        <div class="basis-full order-1 md:order-none">
            <Pagination
                page={pagination.page}
                totalPages={pagination.totalPages}
                perPage={pagination.items}
                {onpagination}
            ></Pagination>
        </div>
        {#if selection !== undefined && selection.size > 0}
            <div class="flex relative flex-shrink-0">
                <TrailDropdown
                    trails={selection}
                    mode={"multi-select"}
                    onDelete={() => handleTrailsEditDone(true)}
                    onShare={() => handleTrailsEditDone(false)}
                />
            </div>
        {/if}
        {#if filter}
            <div class="shrink-0">
                {#if selectedDisplayOption !== "table"}
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

            <Select
                bind:value={selectedDisplayOption}
                items={displayOptions}
                onchange={() => setDisplayOption()}
            ></Select>
        </div>
    </div>

    <div id="trails" class="flex items-start flex-wrap gap-8 py-8 max-w-full">
        {#if loading}
            {#if selectedDisplayOption === "table"}
                <TrailTable
                    trails={null}
                    selection={new Set<Trail>()}
                    tableHeader={sortOptions}
                    items={pagination.items}
                ></TrailTable>
            {:else}
                {#each { length: pagination.items } as _, index}
                    {#if selectedDisplayOption === "cards"}
                        <div class="flex-1">
                            <SkeletonCard></SkeletonCard>
                        </div>
                    {:else if selectedDisplayOption === "list"}
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
            {#if selectedDisplayOption === "table"}
                <TrailTable
                    {trails}
                    {selection}
                    tableHeader={sortOptions.filter(
                        (option) => option.value !== "elevation_loss",
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
                        class:basis-full={selectedDisplayOption === "list"}
                        href="/trail/view/@{trail.author}{trail.domain
                            ? `@${trail.domain}`
                            : ''}/{trail.id}"
                        onmouseenter={(e) => handleMouseEnter(trail)}
                        onmouseleave={(e) => handleMouseLeave(trail)}
                    >
                        {#if selectedDisplayOption === "cards"}
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
    <div class="flex items-end flex-wrap lg:flex-nowrap gap-x-6 gap-y-2 mx-4">
        <div class="basis-full order-1 md:order-none">
            <Pagination
                page={pagination.page}
                totalPages={pagination.totalPages}
                perPage={pagination.items}
                {onpagination}
            ></Pagination>
        </div>
        <div class="shrink-0">
            <Select
                bind:value={pagination.items}
                items={selectedDisplayOption == "cards" ? perPageOptionsCards : perPageOptions}
                onchange={setItemsPerPage}
            ></Select>
        </div>
    </div>
</div>

<style>
    #sort-order-btn {
        transition: transform 0.5s ease;
    }
    :global(.rotated) {
        transform: rotate(180deg);
    }
</style>
