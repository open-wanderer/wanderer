<script lang="ts">
    import type { Category } from "$lib/models/category";
    import type { Subcategory } from "$lib/models/subcategory";
    import {
        displayCategoryIcon,
        displayCategoryName,
        displaySubcategoryIcon,
        displaySubcategoryLabel,
    } from "$lib/util/category_util";
    import { slide } from "svelte/transition";
    import { _, locale } from "svelte-i18n";
    import TrailFilterHistogram from "./trail_filter_histogram.svelte";
    import {
        PREVIEW_ALTITUDE_BINS,
        PREVIEW_BASE_RESULT_COUNT,
        PREVIEW_DISTANCE_BINS,
        PREVIEW_DURATION_BINS,
        PREVIEW_ELEVATION_BINS,
        PREVIEW_LOCATIONS,
        buildPreviewTrailCorpus,
        countByKey,
        filterPreviewTrails,
        type PreviewFilterState,
        type PreviewLocation,
        type PreviewRouteShape,
        type PreviewSlope,
    } from "./trail_filter_preview_model";

    interface Props {
        categories: Category[];
        subcategories: Subcategory[];
        onapply?: () => void;
    }

    const CATEGORY_ORDER = [
        "hiking",
        "walking",
        "biking",
        "running",
        "climbing",
        "skiing",
        "canoeing",
        "other",
    ];

    let { categories, subcategories, onapply }: Props = $props();

    let searchQuery = $state("");
    let locationQuery = $state("");
    let selectedLocation = $state<PreviewLocation | null>(null);
    let locationRadius = $state(25);
    let selectedCategoryIds: string[] = $state([]);
    let selectedSubcategoryIds: string[] = $state([]);
    let activeCategoryId: string | null = $state(null);
    let showAllCategories = $state(false);
    let showAllSubcategories = $state(false);
    let routeShape: PreviewRouteShape = $state("all");
    let photosOnly = $state(false);
    let publicTransport = $state(false);
    let distanceMin = $state(0);
    let distanceMax = $state(50_000);
    let durationMin = $state(0);
    let durationMax = $state(36_000);
    let difficulty: number[] = $state([]);
    let elevationMin = $state(0);
    let elevationMax = $state(2_000);
    let altitudeMin = $state(0);
    let altitudeMax = $state(3_500);
    let slope: PreviewSlope = $state("all");
    let minimumRating = $state(0);
    let moreExpanded = $state(false);

    let categoryItems = $derived(
        [...categories]
            .sort((a, b) => {
                const aRank = CATEGORY_ORDER.indexOf(a.name.toLowerCase());
                const bRank = CATEGORY_ORDER.indexOf(b.name.toLowerCase());
                return (aRank < 0 ? CATEGORY_ORDER.length : aRank) -
                    (bRank < 0 ? CATEGORY_ORDER.length : bRank);
            })
            .map((category) => ({
                ...category,
                label: displayCategoryName(category, $locale),
            })),
    );
    let visibleCategoryItems = $derived(
        showAllCategories ? categoryItems : categoryItems.slice(0, 4),
    );
    let hiddenCategoryCount = $derived(
        Math.max(0, categoryItems.length - visibleCategoryItems.length),
    );
    let subcategoryGroups = $derived(
        categoryItems.map((category) => ({
            category,
            items: subcategories
                .filter((subcategory) => subcategory.category === category.id)
                .sort((a, b) =>
                    displaySubcategoryLabel(a, $locale).localeCompare(
                        displaySubcategoryLabel(b, $locale),
                        $locale ?? undefined,
                        { sensitivity: "base" },
                    ),
                )
                .map((subcategory) => ({
                    ...subcategory,
                    label: displaySubcategoryLabel(subcategory, $locale),
                    icon: displaySubcategoryIcon(subcategory, category),
                })),
        })),
    );
    let selectedCategoryItems = $derived(
        categoryItems.filter((category) =>
            selectedCategoryIds.includes(category.id),
        ),
    );
    let activeSubcategoryGroup = $derived(
        subcategoryGroups.find((group) => group.category.id === activeCategoryId),
    );
    let activeSubcategoryItems = $derived(
        activeSubcategoryGroup?.items ?? [],
    );
    let visibleSubcategoryItems = $derived(
        showAllSubcategories
            ? activeSubcategoryItems
            : activeSubcategoryItems.slice(0, 4),
    );
    let hiddenSubcategoryCount = $derived(
        Math.max(0, activeSubcategoryItems.length - visibleSubcategoryItems.length),
    );
    let activeSelectedSubcategoryIds = $derived(
        activeSubcategoryItems
            .filter((subcategory) =>
                selectedSubcategoryIds.includes(subcategory.id),
            )
            .map((subcategory) => subcategory.id),
    );
    let locationSuggestions = $derived.by(() => {
        const query = locationQuery.trim().toLocaleLowerCase($locale ?? undefined);
        if (!query || selectedLocation) {
            return [];
        }

        return PREVIEW_LOCATIONS.filter((location) =>
            `${location.name} ${location.region} ${location.regionEn ?? ""}`
                .toLocaleLowerCase($locale ?? undefined)
                .includes(query),
        );
    });
    let mockTrails = $derived(
        buildPreviewTrailCorpus(
            categoryItems.map((category) => category.id),
            subcategories.map((subcategory) => ({
                id: subcategory.id,
                categoryId: subcategory.category,
            })),
        ),
    );
    let previewFilterState: PreviewFilterState = $derived({
        query: searchQuery,
        taxonomy: selectedCategoryIds.map((categoryId) => ({
            categoryId,
            subcategoryIds: subcategories
                .filter(
                    (subcategory) =>
                        subcategory.category === categoryId &&
                        selectedSubcategoryIds.includes(subcategory.id),
                )
                .map((subcategory) => subcategory.id),
        })),
        locationId: selectedLocation?.id ?? null,
        locationRadius,
        routeShape,
        photosOnly,
        publicTransport,
        distanceMin,
        distanceMax,
        durationMin,
        durationMax,
        difficulty,
        elevationMin,
        elevationMax,
        altitudeMin,
        altitudeMax,
        slope,
        minimumRating,
    });
    let facetPools = $derived.by(() => ({
        taxonomy: filterPreviewTrails(
            mockTrails,
            previewFilterState,
            "taxonomy",
        ),
        location: filterPreviewTrails(
            mockTrails,
            previewFilterState,
            "location",
        ),
        routeShape: filterPreviewTrails(
            mockTrails,
            previewFilterState,
            "routeShape",
        ),
        photos: filterPreviewTrails(
            mockTrails,
            previewFilterState,
            "photos",
        ),
        transit: filterPreviewTrails(
            mockTrails,
            previewFilterState,
            "transit",
        ),
        difficulty: filterPreviewTrails(
            mockTrails,
            previewFilterState,
            "difficulty",
        ),
    }));
    let categoryCountById = $derived(
        countByKey(facetPools.taxonomy, (trail) => trail.categoryId),
    );
    let subcategoryCountById = $derived(
        countByKey(facetPools.taxonomy, (trail) => trail.subcategoryId),
    );
    let locationCountById = $derived(
        new Map(
            PREVIEW_LOCATIONS.map((location) => [
                location.id,
                facetPools.location.filter(
                    (trail) =>
                        trail.locationDistancesKm[location.id] <= locationRadius,
                ).length,
            ]),
        ),
    );
    let routeShapeCounts = $derived({
        all: facetPools.routeShape.length,
        loop: facetPools.routeShape.filter(
            (trail) => trail.routeShape === "loop",
        ).length,
        "point-to-point": facetPools.routeShape.filter(
            (trail) => trail.routeShape === "point-to-point",
        ).length,
    });
    let photosCount = $derived(
        facetPools.photos.filter((trail) => trail.hasPhotos).length,
    );
    let transitCount = $derived(
        facetPools.transit.filter((trail) => trail.hasTransit).length,
    );
    let difficultyCountByValue = $derived(
        new Map(
            [0, 1, 2].map((value) => [
                value,
                facetPools.difficulty.filter(
                    (trail) => trail.difficulty === value,
                ).length,
            ]),
        ),
    );
    let previewResultCount = $derived(
        filterPreviewTrails(mockTrails, previewFilterState).length,
    );
    let announcedResultCount = $state(PREVIEW_BASE_RESULT_COUNT);
    $effect(() => {
        const nextCount = previewResultCount;
        const timer = window.setTimeout(() => {
            announcedResultCount = nextCount;
        }, 250);

        return () => window.clearTimeout(timer);
    });
    let activeMoreFilterCount = $derived(
        (difficulty.length > 0 ? 1 : 0) +
            (elevationMin > 0 || elevationMax < 2_000 ? 1 : 0) +
            (altitudeMin > 0 || altitudeMax < 3_500 ? 1 : 0) +
            (slope !== "all" ? 1 : 0) +
            (minimumRating > 0 ? 1 : 0),
    );

    function toggleCategory(categoryId: string) {
        if (selectedCategoryIds.includes(categoryId)) {
            selectedCategoryIds = selectedCategoryIds.filter(
                (id) => id !== categoryId,
            );
            const removedSubcategoryIds = new Set(
                subcategories
                    .filter((subcategory) => subcategory.category === categoryId)
                    .map((subcategory) => subcategory.id),
            );
            selectedSubcategoryIds = selectedSubcategoryIds.filter(
                (id) => !removedSubcategoryIds.has(id),
            );
            if (activeCategoryId === categoryId) {
                activeCategoryId = selectedCategoryIds.at(-1) ?? null;
            }
        } else {
            selectedCategoryIds = [...selectedCategoryIds, categoryId];
            activeCategoryId = categoryId;
        }

        showAllSubcategories = false;
    }

    function setActiveCategory(categoryId: string) {
        activeCategoryId = categoryId;
        showAllSubcategories = false;
    }

    function toggleSubcategory(subcategoryId: string) {
        selectedSubcategoryIds = selectedSubcategoryIds.includes(subcategoryId)
            ? selectedSubcategoryIds.filter((id) => id !== subcategoryId)
            : [...selectedSubcategoryIds, subcategoryId];
    }

    function clearActiveSubcategories() {
        const activeIds = new Set(
            activeSubcategoryItems.map((subcategory) => subcategory.id),
        );
        selectedSubcategoryIds = selectedSubcategoryIds.filter(
            (id) => !activeIds.has(id),
        );
    }

    function selectLocation(location: PreviewLocation) {
        selectedLocation = location;
        locationQuery = location.name;
    }

    function updateLocationQuery(event: Event) {
        const value = (event.currentTarget as HTMLInputElement).value;
        locationQuery = value;
        if (selectedLocation && selectedLocation.name !== value) {
            selectedLocation = null;
            locationRadius = 25;
        }
    }

    function clearLocation() {
        locationQuery = "";
        selectedLocation = null;
        locationRadius = 25;
    }

    function displayLocationRegion(location: PreviewLocation) {
        return ($locale ?? "en").toLowerCase().startsWith("de")
            ? location.region
            : (location.regionEn ?? location.region);
    }

    function toggleDifficulty(value: number) {
        difficulty = difficulty.includes(value)
            ? difficulty.filter((item) => item !== value)
            : [...difficulty, value];
    }

    function formatKilometers(value: number) {
        return `${value / 1_000} km`;
    }

    function formatHours(value: number) {
        const hours = value / 3_600;
        return `${Number.isInteger(hours) ? hours : hours.toFixed(1)} h`;
    }

    function formatMeters(value: number) {
        return `${new Intl.NumberFormat($locale ?? "en").format(value)} m`;
    }

    function resetPreview() {
        searchQuery = "";
        locationQuery = "";
        selectedLocation = null;
        locationRadius = 25;
        selectedCategoryIds = [];
        selectedSubcategoryIds = [];
        activeCategoryId = null;
        showAllCategories = false;
        showAllSubcategories = false;
        routeShape = "all";
        photosOnly = false;
        publicTransport = false;
        distanceMin = 0;
        distanceMax = 50_000;
        durationMin = 0;
        durationMax = 36_000;
        difficulty = [];
        elevationMin = 0;
        elevationMax = 2_000;
        altitudeMin = 0;
        altitudeMax = 3_500;
        slope = "all";
        minimumRating = 0;
    }
</script>

<aside class="overflow-hidden rounded-2xl border border-input-border bg-background shadow-sm">
    <div class="border-b border-separator bg-input-background/50 px-5 py-4">
        <div class="flex items-start justify-between gap-3">
            <div>
                <h2 class="font-semibold">{$_("filter-preview-title")}</h2>
                <p class="mt-0.5 text-xs text-gray-500">
                    {$_("filter-preview-subtitle")}
                </p>
            </div>
            <span
                class="rounded-full border border-input-border-focus bg-input-background px-2 py-1 text-[10px] font-bold uppercase tracking-wide text-content"
            >
                {$_("filter-preview-badge")}
            </span>
        </div>
    </div>

    <div class="space-y-5 p-5">
        <p class="flex items-start gap-2 rounded-lg bg-input-background px-3 py-2 text-xs">
            <i class="fa fa-flask mt-0.5 text-gray-500"></i>
            <span>{$_("filter-preview-notice")}</span>
        </p>

        <div class="space-y-1">
            <label for="preview-trail-search" class="text-sm font-medium">
                {$_("search-trails")}
            </label>
            <div class="relative">
                <i
                    class="fa fa-search pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-gray-500"
                ></i>
                <input
                    id="preview-trail-search"
                    name="preview-trail-search"
                    type="search"
                    autocomplete="off"
                    class="w-full rounded-md border border-input-border bg-input-background py-3 pl-10 pr-3 transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0"
                    placeholder="{$_('search-trails')}..."
                    bind:value={searchQuery}
                />
            </div>
        </div>

        <section class="space-y-2.5">
            <div class="flex items-center justify-between gap-3">
                <label
                    for="preview-location-search"
                    class="text-sm font-medium"
                >
                    {$_("filter-preview-location-radius")}
                </label>
                <span class="text-xs text-gray-500">
                    {$_("filter-preview-example-places")}
                </span>
            </div>

            <div class="relative">
                <i
                    class="fa fa-location-dot pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-gray-500"
                ></i>
                <input
                    id="preview-location-search"
                    name="preview-location-search"
                    type="text"
                    autocomplete="off"
                    class="w-full rounded-md border border-input-border bg-input-background py-3 pl-10 pr-10 transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0"
                    placeholder="{$_('search-places')}..."
                    value={locationQuery}
                    oninput={updateLocationQuery}
                />
                {#if locationQuery.length > 0}
                    <button
                        type="button"
                        class="btn-icon absolute right-2 top-1/2 -translate-y-1/2"
                        aria-label={$_("filter-preview-clear-location")}
                        title={$_("filter-preview-clear-location")}
                        onclick={clearLocation}
                    >
                        <i class="fa fa-close text-sm"></i>
                    </button>
                {/if}
            </div>

            {#if locationQuery.trim().length > 0 && !selectedLocation}
                <ul
                    id="preview-location-suggestions"
                    class="overflow-hidden rounded-xl border border-input-border bg-menu-background shadow-sm"
                    aria-label={$_("filter-preview-example-places")}
                >
                    {#each locationSuggestions as location}
                        {@const count = locationCountById.get(location.id) ?? 0}
                        <li>
                            <button
                                type="button"
                                class="flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus"
                                class:cursor-not-allowed={count === 0}
                                class:opacity-40={count === 0}
                                disabled={count === 0}
                                onclick={() => selectLocation(location)}
                            >
                                <i
                                    class="fa fa-location-dot w-5 shrink-0 text-center text-gray-500"
                                ></i>
                                <span class="min-w-0 flex-1">
                                    <span class="block truncate text-sm font-medium">
                                        {location.name}
                                    </span>
                                    <span class="block truncate text-xs text-gray-500">
                                        {displayLocationRegion(location)} · {$_(
                                            "filter-preview-demo-country",
                                        )}
                                    </span>
                                </span>
                                <span class="text-xs tabular-nums text-gray-500">
                                    {count}
                                </span>
                            </button>
                        </li>
                    {:else}
                        <li class="px-3 py-2.5 text-sm text-gray-500">
                            {$_("no-results")}
                        </li>
                    {/each}
                </ul>
            {:else if !selectedLocation}
                <div class="flex flex-wrap items-center gap-1.5">
                    {#each PREVIEW_LOCATIONS.slice(0, 3) as location}
                        {@const count = locationCountById.get(location.id) ?? 0}
                        <button
                            type="button"
                            class="rounded-full border border-input-border bg-input-background px-2.5 py-1 text-xs transition-colors hover:bg-menu-item-background-hover"
                            class:cursor-not-allowed={count === 0}
                            class:opacity-40={count === 0}
                            disabled={count === 0}
                            onclick={() => selectLocation(location)}
                        >
                            {location.name}
                            <span class="ml-1 text-[10px] text-gray-500">
                                {count}
                            </span>
                        </button>
                    {/each}
                </div>
            {/if}

            {#if selectedLocation}
                <div
                    class="space-y-3 rounded-xl border border-input-border-focus bg-input-background p-3"
                >
                    <div class="flex min-w-0 items-center gap-3">
                        <span
                            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-toggle-active text-white"
                        >
                            <i class="fa fa-location-dot"></i>
                        </span>
                        <span class="min-w-0 flex-1">
                            <span class="block truncate text-sm font-semibold">
                                {selectedLocation.name}
                            </span>
                            <span class="block truncate text-xs text-gray-500">
                                {displayLocationRegion(selectedLocation)} · {$_(
                                    "filter-preview-demo-country",
                                )}
                            </span>
                        </span>
                        <span
                            class="rounded-full bg-background px-2 py-1 text-xs font-semibold tabular-nums"
                        >
                            {previewResultCount}
                        </span>
                    </div>
                    <div class="flex items-center gap-3 border-t border-separator pt-3">
                        <span class="shrink-0 text-xs text-gray-500">
                            {$_("radius")}
                        </span>
                        <div
                            class="grid min-w-0 flex-1 grid-cols-4 rounded-lg bg-background p-1"
                            role="radiogroup"
                            aria-label={$_("radius")}
                        >
                            {#each [5, 10, 25, 50] as radius}
                                <label
                                    class="cursor-pointer rounded-md px-1 py-1.5 text-center text-[11px] transition-all focus-within:ring-2 focus-within:ring-input-ring"
                                    class:bg-toggle-active={locationRadius === radius}
                                    class:text-white={locationRadius === radius}
                                    class:font-semibold={locationRadius === radius}
                                >
                                    <input
                                        class="sr-only"
                                        type="radio"
                                        name="preview-location-radius"
                                        value={radius}
                                        checked={locationRadius === radius}
                                        onchange={() => (locationRadius = radius)}
                                    />
                                    {radius} km
                                </label>
                            {/each}
                        </div>
                    </div>
                </div>
            {/if}
        </section>

        <section class="space-y-2.5">
            <div class="flex items-center justify-between">
                <h3 class="text-sm font-medium">{$_("categories")}</h3>
                <span class="text-xs text-gray-500">
                    {$_("filter-preview-counts")}
                </span>
            </div>
            <div class="grid grid-cols-2 gap-2">
                {#each visibleCategoryItems as category}
                    {@const selected = selectedCategoryIds.includes(category.id)}
                    {@const count = categoryCountById.get(category.id) ?? 0}
                    {@const unavailable = count === 0 && !selected}
                    <button
                        type="button"
                        class="flex min-w-0 items-center gap-2 rounded-xl border px-3 py-2 text-left transition-colors"
                        class:border-input-border-focus={selected}
                        class:bg-toggle-active={selected}
                        class:text-white={selected}
                        class:border-input-border={!selected}
                        class:bg-input-background={!selected}
                        class:hover:bg-menu-item-background-hover={!selected &&
                            !unavailable}
                        class:cursor-not-allowed={unavailable}
                        class:opacity-40={unavailable}
                        disabled={unavailable}
                        aria-pressed={selected}
                        onclick={() => toggleCategory(category.id)}
                    >
                        <i class="fa {displayCategoryIcon(category)} w-5 shrink-0 text-center"></i>
                        <span class="min-w-0 flex-1 truncate text-xs font-medium">
                            {category.label}
                        </span>
                        <span
                            class="shrink-0 text-[10px] tabular-nums"
                            class:text-white={selected}
                            class:opacity-75={selected}
                            class:text-gray-500={!selected}
                        >
                            {count}
                        </span>
                    </button>
                {/each}
                {#if hiddenCategoryCount > 0}
                    <button
                        type="button"
                        class="rounded-xl border border-dashed border-input-border px-3 py-2 text-xs font-medium text-gray-500 hover:bg-menu-item-background-hover"
                        onclick={() => (showAllCategories = true)}
                    >
                        +{hiddenCategoryCount} {$_("more")}
                    </button>
                {/if}
            </div>

            <div
                class="space-y-3 rounded-xl border border-input-border bg-input-background/60 p-3"
            >
                <div class="flex items-center justify-between gap-3">
                    <h4 class="text-xs font-semibold">
                        {$_("filter-preview-subcategories")}
                    </h4>
                    {#if selectedSubcategoryIds.length > 0}
                        <span
                            class="flex h-5 min-w-5 items-center justify-center rounded-full bg-toggle-active px-1 text-[10px] font-semibold text-white"
                        >
                            {selectedSubcategoryIds.length}
                        </span>
                    {/if}
                </div>

                {#if selectedCategoryItems.length === 0}
                    <p class="flex items-start gap-2 text-xs leading-4 text-gray-500">
                        <i class="fa fa-layer-group mt-0.5"></i>
                        <span>{$_("filter-preview-choose-category")}</span>
                    </p>
                {:else}
                    {#if selectedCategoryItems.length > 1}
                        <div
                            class="flex gap-1 overflow-x-auto pb-1"
                            role="group"
                            aria-label={$_("categories")}
                        >
                            {#each selectedCategoryItems as category}
                                <button
                                    type="button"
                                    class="shrink-0 rounded-full border px-2.5 py-1 text-[11px] transition-colors"
                                    class:border-input-border-focus={activeCategoryId ===
                                        category.id}
                                    class:bg-toggle-active={activeCategoryId ===
                                        category.id}
                                    class:text-white={activeCategoryId === category.id}
                                    class:border-input-border={activeCategoryId !==
                                        category.id}
                                    class:bg-background={activeCategoryId !==
                                        category.id}
                                    aria-pressed={activeCategoryId === category.id}
                                    onclick={() => setActiveCategory(category.id)}
                                >
                                    {category.label}
                                </button>
                            {/each}
                        </div>
                    {/if}

                    {#if activeSubcategoryGroup && activeSubcategoryItems.length > 0}
                        <div
                            class="grid grid-cols-2 gap-2"
                            role="group"
                            aria-label={$_("filter-preview-subcategories-for", {
                                values: {
                                    category: activeSubcategoryGroup.category.label,
                                },
                            })}
                        >
                            <button
                                type="button"
                                class="flex min-w-0 items-center gap-2 rounded-lg border px-2 py-2 text-left text-xs transition-colors"
                                class:border-input-border-focus={activeSelectedSubcategoryIds.length ===
                                    0}
                                class:bg-toggle-active={activeSelectedSubcategoryIds.length ===
                                    0}
                                class:text-white={activeSelectedSubcategoryIds.length ===
                                    0}
                                class:border-input-border={activeSelectedSubcategoryIds.length >
                                    0}
                                class:bg-background={activeSelectedSubcategoryIds.length >
                                    0}
                                aria-pressed={activeSelectedSubcategoryIds.length === 0}
                                onclick={clearActiveSubcategories}
                            >
                                <i
                                    class="fa {displayCategoryIcon(
                                        activeSubcategoryGroup.category,
                                    )} w-4 shrink-0 text-center"
                                ></i>
                                <span class="min-w-0 flex-1 truncate">
                                    {$_("filter-preview-all")}
                                </span>
                                <span class="text-[10px] opacity-70">
                                    {categoryCountById.get(
                                        activeSubcategoryGroup.category.id,
                                    ) ?? 0}
                                </span>
                            </button>

                            {#each visibleSubcategoryItems as subcategory}
                                {@const selected = selectedSubcategoryIds.includes(
                                    subcategory.id,
                                )}
                                {@const count =
                                    subcategoryCountById.get(subcategory.id) ?? 0}
                                {@const unavailable = count === 0 && !selected}
                                <button
                                    type="button"
                                    class="flex min-w-0 items-center gap-2 rounded-lg border px-2 py-2 text-left text-xs transition-colors"
                                    class:border-input-border-focus={selected}
                                    class:bg-toggle-active={selected}
                                    class:text-white={selected}
                                    class:border-input-border={!selected}
                                    class:bg-background={!selected}
                                    class:hover:bg-menu-item-background-hover={!selected &&
                                        !unavailable}
                                    class:cursor-not-allowed={unavailable}
                                    class:opacity-40={unavailable}
                                    disabled={unavailable}
                                    aria-pressed={selected}
                                    onclick={() => toggleSubcategory(subcategory.id)}
                                >
                                    <i
                                        class="fa {subcategory.icon} w-4 shrink-0 text-center"
                                    ></i>
                                    <span class="min-w-0 flex-1 truncate">
                                        {subcategory.label}
                                    </span>
                                    <span class="text-[10px] opacity-70">
                                        {count}
                                    </span>
                                </button>
                            {/each}

                            {#if hiddenSubcategoryCount > 0}
                                <button
                                    type="button"
                                    class="rounded-lg border border-dashed border-input-border px-2 py-2 text-xs font-medium text-gray-500 transition-colors hover:bg-menu-item-background-hover"
                                    onclick={() => (showAllSubcategories = true)}
                                >
                                    +{hiddenSubcategoryCount} {$_("more")}
                                </button>
                            {/if}
                        </div>
                    {:else}
                        <p class="text-xs leading-4 text-gray-500">
                            {$_("filter-preview-no-subcategories")}
                        </p>
                    {/if}
                {/if}
            </div>
        </section>

        <section class="space-y-2.5">
            <h3 class="text-sm font-medium">{$_("filter-preview-route-shape")}</h3>
            <div
                class="grid grid-cols-3 rounded-xl bg-input-background p-1"
                role="radiogroup"
                aria-label={$_("filter-preview-route-shape")}
            >
                {#each [
                    {
                        value: "all",
                        label: $_("filter-preview-all"),
                        count: routeShapeCounts.all,
                    },
                    {
                        value: "loop",
                        label: $_("loop"),
                        count: routeShapeCounts.loop,
                    },
                    {
                        value: "point-to-point",
                        label: $_("filter-preview-point-to-point"),
                        count: routeShapeCounts["point-to-point"],
                    },
                ] as item}
                    {@const selected = routeShape === item.value}
                    {@const unavailable = item.count === 0 && !selected}
                    <label
                        class="cursor-pointer rounded-lg px-1.5 py-2 text-xs transition-all focus-within:ring-2 focus-within:ring-input-ring"
                        class:bg-background={selected}
                        class:font-semibold={selected}
                        class:shadow-sm={selected}
                        class:cursor-not-allowed={unavailable}
                        class:opacity-40={unavailable}
                    >
                        <input
                            class="sr-only"
                            type="radio"
                            name="preview-route-shape"
                            value={item.value}
                            checked={selected}
                            disabled={unavailable}
                            onchange={() =>
                                (routeShape = item.value as PreviewRouteShape)}
                        />
                        <span class="block truncate">{item.label}</span>
                        <span class="text-[10px] text-gray-500">{item.count}</span>
                    </label>
                {/each}
            </div>
        </section>

        <section class="space-y-2.5">
            <h3 class="text-sm font-medium">{$_("filter-preview-quick-filters")}</h3>
            <div class="grid grid-cols-2 gap-2">
                <button
                    type="button"
                    class="flex items-center gap-2 rounded-xl border px-3 py-2 text-left text-xs transition-colors"
                    class:border-input-border-focus={photosOnly}
                    class:bg-toggle-active={photosOnly}
                    class:text-white={photosOnly}
                    class:border-input-border={!photosOnly}
                    class:bg-input-background={!photosOnly}
                    class:cursor-not-allowed={photosCount === 0 && !photosOnly}
                    class:opacity-40={photosCount === 0 && !photosOnly}
                    disabled={photosCount === 0 && !photosOnly}
                    aria-pressed={photosOnly}
                    onclick={() => (photosOnly = !photosOnly)}
                >
                    <i class="fa fa-camera w-4"></i>
                    <span class="min-w-0 flex-1 truncate">{$_("filter-preview-has-photos")}</span>
                    <span class="text-[10px] opacity-70">{photosCount}</span>
                </button>
                <button
                    type="button"
                    class="flex items-center gap-2 rounded-xl border px-3 py-2 text-left text-xs transition-colors"
                    class:border-input-border-focus={publicTransport}
                    class:bg-toggle-active={publicTransport}
                    class:text-white={publicTransport}
                    class:border-input-border={!publicTransport}
                    class:bg-input-background={!publicTransport}
                    class:cursor-not-allowed={transitCount === 0 &&
                        !publicTransport}
                    class:opacity-40={transitCount === 0 && !publicTransport}
                    disabled={transitCount === 0 && !publicTransport}
                    aria-pressed={publicTransport}
                    onclick={() => (publicTransport = !publicTransport)}
                >
                    <i class="fa fa-train w-4"></i>
                    <span class="min-w-0 flex-1 truncate">
                        {$_("filter-preview-transit-access")}
                    </span>
                    <span class="text-[10px] opacity-70">{transitCount}</span>
                </button>
            </div>
        </section>

        <hr class="border-separator" />

        <TrailFilterHistogram
            id="preview-distance"
            label={$_("distance")}
            bins={PREVIEW_DISTANCE_BINS}
            max={50_000}
            step={1_000}
            bind:currentMin={distanceMin}
            bind:currentMax={distanceMax}
            formatValue={formatKilometers}
        />

        <TrailFilterHistogram
            id="preview-duration"
            label={$_("duration")}
            bins={PREVIEW_DURATION_BINS}
            max={36_000}
            step={1_800}
            bind:currentMin={durationMin}
            bind:currentMax={durationMax}
            formatValue={formatHours}
        />

        <p class="-mt-2 text-[11px] leading-4 text-gray-500">
            <i class="fa fa-circle-info mr-1"></i>
            {$_("filter-preview-capped-hint")}
        </p>

        <div class="border-y border-separator">
            <button
                type="button"
                class="flex w-full items-center justify-between py-3 text-sm font-semibold"
                aria-expanded={moreExpanded}
                aria-controls="preview-more-filters"
                onclick={() => (moreExpanded = !moreExpanded)}
            >
                <span class="flex items-center gap-2">
                    {$_("filter-preview-more-filters")}
                    {#if activeMoreFilterCount > 0}
                        <span
                            class="flex h-5 min-w-5 items-center justify-center rounded-full bg-toggle-active px-1 text-[10px] text-white"
                        >
                            {activeMoreFilterCount}
                        </span>
                    {/if}
                </span>
                <i
                    class="fa fa-chevron-down text-xs transition-transform"
                    class:rotate-180={moreExpanded}
                ></i>
            </button>

            {#if moreExpanded}
                <div id="preview-more-filters" class="space-y-6 pb-5" transition:slide>
                    <section class="space-y-2.5">
                        <h3 class="text-sm font-medium">{$_("difficulty")}</h3>
                        <div class="grid grid-cols-3 gap-2">
                            {#each [
                                {
                                    value: 0,
                                    label: $_("easy"),
                                    count: difficultyCountByValue.get(0) ?? 0,
                                },
                                {
                                    value: 1,
                                    label: $_("moderate"),
                                    count: difficultyCountByValue.get(1) ?? 0,
                                },
                                {
                                    value: 2,
                                    label: $_("difficult"),
                                    count: difficultyCountByValue.get(2) ?? 0,
                                },
                            ] as item}
                                {@const selected = difficulty.includes(item.value)}
                                {@const unavailable = item.count === 0 && !selected}
                                <button
                                    type="button"
                                    class="rounded-lg border px-1.5 py-2 text-xs transition-colors"
                                    class:border-input-border-focus={selected}
                                    class:bg-toggle-active={selected}
                                    class:text-white={selected}
                                    class:border-input-border={!selected}
                                    class:bg-input-background={!selected}
                                    class:cursor-not-allowed={unavailable}
                                    class:opacity-40={unavailable}
                                    disabled={unavailable}
                                    aria-pressed={selected}
                                    onclick={() => toggleDifficulty(item.value)}
                                >
                                    <span class="block truncate">{item.label}</span>
                                    <span class="text-[10px] opacity-70">{item.count}</span>
                                </button>
                            {/each}
                        </div>
                    </section>

                    <TrailFilterHistogram
                        id="preview-elevation"
                        label={$_("elevation-gain")}
                        bins={PREVIEW_ELEVATION_BINS}
                        max={2_000}
                        step={50}
                        bind:currentMin={elevationMin}
                        bind:currentMax={elevationMax}
                        formatValue={formatMeters}
                    />

                    <TrailFilterHistogram
                        id="preview-altitude"
                        label={$_("filter-preview-highest-point")}
                        bins={PREVIEW_ALTITUDE_BINS}
                        max={3_500}
                        step={100}
                        bind:currentMin={altitudeMin}
                        bind:currentMax={altitudeMax}
                        formatValue={formatMeters}
                    />

                    <section class="space-y-2.5">
                        <h3 class="text-sm font-medium">{$_("slope")}</h3>
                        <div
                            class="grid grid-cols-2 gap-2"
                            role="radiogroup"
                            aria-label={$_("slope")}
                        >
                            {#each [
                                { value: "all", label: $_("filter-preview-all") },
                                { value: "gentle", label: $_("filter-preview-gentle") },
                                { value: "balanced", label: $_("filter-preview-balanced") },
                                { value: "steep", label: $_("filter-preview-steep") },
                            ] as item}
                                <label
                                    class="cursor-pointer rounded-lg border px-2 py-2 text-center text-xs transition-colors focus-within:ring-2 focus-within:ring-input-ring"
                                    class:border-input-border-focus={slope === item.value}
                                    class:bg-toggle-active={slope === item.value}
                                    class:text-white={slope === item.value}
                                    class:border-input-border={slope !== item.value}
                                    class:bg-input-background={slope !== item.value}
                                >
                                    <input
                                        class="sr-only"
                                        type="radio"
                                        name="preview-slope"
                                        value={item.value}
                                        checked={slope === item.value}
                                        onchange={() =>
                                            (slope = item.value as PreviewSlope)}
                                    />
                                    {item.label}
                                </label>
                            {/each}
                        </div>
                    </section>

                    <section class="space-y-2.5">
                        <div class="flex items-center justify-between gap-3">
                            <h3 class="text-sm font-medium">
                                {$_("filter-preview-minimum-rating")}
                            </h3>
                            {#if minimumRating > 0}
                                <span class="text-xs font-semibold text-content">
                                    {minimumRating}+
                                </span>
                            {/if}
                        </div>
                        <div
                            class="flex items-center justify-between rounded-xl bg-input-background px-3 py-2"
                            role="radiogroup"
                            aria-label={$_("filter-preview-minimum-rating")}
                        >
                            {#each [1, 2, 3, 4, 5] as star}
                                <label
                                    class="btn-icon h-7 cursor-pointer rounded-full text-base focus-within:ring-2 focus-within:ring-input-ring"
                                >
                                    <input
                                        class="sr-only"
                                        type="radio"
                                        name="preview-minimum-rating"
                                        value={star}
                                        checked={minimumRating === star}
                                        aria-label={$_("filter-preview-stars", {
                                            values: { count: star },
                                        })}
                                        onchange={() => (minimumRating = star)}
                                    />
                                    <i
                                        class="fa-star"
                                        class:fa-solid={minimumRating >= star}
                                        class:fa-regular={minimumRating < star}
                                        class:text-amber-400={minimumRating >= star}
                                        class:text-gray-400={minimumRating < star}
                                    ></i>
                                </label>
                            {/each}
                        </div>
                    </section>
                </div>
            {/if}
        </div>

        <div class="flex gap-2">
            <button
                type="button"
                class="btn-secondary shrink-0 px-3"
                aria-label={$_("reset")}
                title={$_("reset")}
                onclick={resetPreview}
            >
                <i class="fa fa-rotate-left"></i>
            </button>
            <button
                type="button"
                class="btn-primary min-w-0 flex-1 truncate px-3"
                class:cursor-not-allowed={previewResultCount === 0}
                class:opacity-50={previewResultCount === 0}
                disabled={previewResultCount === 0}
                onclick={() => onapply?.()}
            >
                {$_("filter-preview-show-results", {
                    values: { count: previewResultCount },
                })}
            </button>
        </div>
        <p class="sr-only" role="status" aria-live="polite" aria-atomic="true">
            {$_("filter-preview-result-count-status", {
                values: { count: announcedResultCount },
            })}
        </p>
    </div>
</aside>
