<script lang="ts">
    import type { Category } from "$lib/models/category";
    import type { Subcategory } from "$lib/models/subcategory";
    import type { TrailFilter } from "$lib/models/trail";
    import { categoryPreferences } from "$lib/stores/category_preference_store";
    import { subcategoryPreferences } from "$lib/stores/subcategory_preference_store";
    import { subcategories } from "$lib/stores/subcategory_store";
    import {
        noSubcategoryFilterCategory,
        noSubcategoryFilterValue,
    } from "$lib/util/trail_filter_util";
    import {
        displayCategoryIcon,
        displayCategoryName,
        displaySubcategoryBadgeIcon,
        displaySubcategoryIcon,
        displaySubcategoryLabel,
        displaySubcategoryShortBadge,
        preferenceForCategory,
        sortedCategoriesByPreference,
        sortedSubcategoriesByPreference,
        subcategoryVisible,
    } from "$lib/util/category_util";
    import { _, locale } from "svelte-i18n";
    import type { SelectItem } from "../base/select.svelte";

    interface Props {
        categories: Category[];
        filter: TrailFilter;
        onupdate?: (filter: TrailFilter) => void;
    }

    type CategorySelectItem = SelectItem & {
        icon: string;
    };

    const CATEGORY_BUTTON_WIDTH = 40;
    const CATEGORY_BUTTON_GAP = 8;
    const FALLBACK_VISIBLE_CATEGORY_LIMIT = 4;

    let { categories, filter = $bindable(), onupdate }: Props = $props();

    let categorySelectItems = $derived(
        sortedCategoriesByPreference(
            categories,
            $categoryPreferences,
            $locale,
        )
            .filter(
                (c) =>
                    preferenceForCategory($categoryPreferences, c.id)?.visible !==
                        false || filter.category.includes(c.id),
            )
            .map((c) => ({
                value: c.id,
                text: displayCategoryName(c, $locale),
                icon: displayCategoryIcon(c),
            })),
    );
    let orderedCategorySelectItems = $derived(
        [...categorySelectItems].sort((a, b) => {
            const aSelected = filter.category.includes(a.value);
            const bSelected = filter.category.includes(b.value);

            if (aSelected === bSelected) {
                return 0;
            }

            return aSelected ? -1 : 1;
        }),
    );
    let categoryListElement: HTMLDivElement | undefined = $state();
    let categoryListWidth = $state(0);
    let visibleCategoryLimit = $derived(
        visibleCategoryLimitForWidth(categoryListWidth),
    );
    let calculatedVisibleCategoryItems = $derived(
        orderedCategorySelectItems.slice(0, visibleCategoryLimit),
    );
    let calculatedOverflowCategoryItems = $derived(
        visibleCategoryLimit >= orderedCategorySelectItems.length
            ? []
            : orderedCategorySelectItems.slice(visibleCategoryLimit),
    );
    let overflowExpanded = $state(false);
    let visibleCategorySnapshot: CategorySelectItem[] = $state([]);
    let overflowCategorySnapshot: CategorySelectItem[] = $state([]);
    let visibleCategoryItems = $derived(
        overflowExpanded
            ? visibleCategorySnapshot
            : calculatedVisibleCategoryItems,
    );
    let overflowCategoryItems = $derived(
        overflowExpanded
            ? overflowCategorySnapshot
            : calculatedOverflowCategoryItems,
    );
    let overflowHasActiveFilters = $derived(
        overflowCategoryItems.some((category) =>
            filter.category.includes(category.value),
        ),
    );
    let activeHiddenCategories = $derived(
        categories
            .filter(
                (category) =>
                    filter.category.includes(category.id) &&
                    preferenceForCategory($categoryPreferences, category.id)
                        ?.visible === false,
            )
            .map((category) => displayCategoryName(category, $locale)),
    );
    let hoveredCategoryId: string | undefined = $state();
    let categoryTooltip = $state("");
    let categoryTooltipStyle = $state("");
    let longPressTimer: ReturnType<typeof setTimeout> | undefined;
    let longPressCategoryId: string | undefined;
    let longPressStart:
        | {
              x: number;
              y: number;
          }
        | undefined;
    let suppressCategoryClick: string | undefined;
    let suppressCategoryClickTimer: ReturnType<typeof setTimeout> | undefined;
    let hoveredCategoryItem = $derived(
        categorySelectItems.find(
            (category) => category.value === hoveredCategoryId,
        ),
    );
    let selectedSubcategoryIds = $derived(filter.subcategory ?? []);
    let hoveredSubcategories = $derived(
        hoveredCategoryId
            ? sortedSubcategoriesByPreference(
                  $subcategories.filter(
                      (subcategory) =>
                          subcategory.category === hoveredCategoryId &&
                          (subcategoryVisible(
                              subcategory.id,
                              $subcategoryPreferences,
                          ) ||
                              selectedSubcategoryIds.includes(subcategory.id)),
                  ),
                  $subcategoryPreferences,
                  $locale,
              )
            : [],
    );
    let subcategoryOverlayStyle = $state("");

    $effect(() => {
        if (!categoryListElement || typeof ResizeObserver === "undefined") {
            return;
        }

        const observer = new ResizeObserver(([entry]) => {
            categoryListWidth = entry.contentRect.width;
        });

        observer.observe(categoryListElement);
        return () => observer.disconnect();
    });

    function visibleCategoryLimitForWidth(width: number) {
        if (width <= 0) {
            return FALLBACK_VISIBLE_CATEGORY_LIMIT;
        }

        const slotCount = Math.max(
            1,
            Math.floor(
                (width + CATEGORY_BUTTON_GAP) /
                    (CATEGORY_BUTTON_WIDTH + CATEGORY_BUTTON_GAP),
            ),
        );

        if (orderedCategorySelectItems.length <= slotCount) {
            return orderedCategorySelectItems.length;
        }

        return Math.max(0, slotCount - 1);
    }

    async function update() {
        onupdate?.(filter);
    }

    function toggleOverflow() {
        if (overflowExpanded) {
            closeOverflow();
            return;
        }

        visibleCategorySnapshot = calculatedVisibleCategoryItems;
        overflowCategorySnapshot = calculatedOverflowCategoryItems;
        overflowExpanded = true;
    }

    function closeOverflow() {
        overflowExpanded = false;
        visibleCategorySnapshot = [];
        overflowCategorySnapshot = [];
        hoveredCategoryId = undefined;
        subcategoryOverlayStyle = "";
        hideFilterTooltip();
    }

    function toggleCategoryFilter(category: CategorySelectItem) {
        if (filter.category.includes(category.value)) {
            filter.category = filter.category.filter((id) => id !== category.value);
            if (hoveredCategoryId === category.value) {
                hoveredCategoryId = undefined;
            }
            filter.subcategory = selectedSubcategoryIds.filter(
                (id) =>
                    noSubcategoryFilterCategory(id) !== category.value &&
                    !$subcategories.some(
                        (subcategory) =>
                            subcategory.id === id &&
                            subcategory.category === category.value,
                    ),
            );
        } else {
            filter.category = [...filter.category, category.value];
        }

        update();
    }

    function handleCategoryClick(e: MouseEvent, category: CategorySelectItem) {
        if (suppressCategoryClick === category.value) {
            e.preventDefault();
            e.stopPropagation();
            clearSuppressedCategoryClick();
            return;
        }

        toggleCategoryFilter(category);
    }

    function toggleNoSubcategoryFilter(category: CategorySelectItem) {
        const value = noSubcategoryFilterValue(category.value);

        if (selectedSubcategoryIds.includes(value)) {
            filter.subcategory = selectedSubcategoryIds.filter((id) => id !== value);
        } else {
            if (!filter.category.includes(category.value)) {
                filter.category = [...filter.category, category.value];
            }
            filter.subcategory = [...selectedSubcategoryIds, value];
        }

        update();
    }

    function toggleSubcategoryFilter(subcategory: Subcategory) {
        if (selectedSubcategoryIds.includes(subcategory.id)) {
            filter.subcategory = selectedSubcategoryIds.filter(
                (id) => id !== subcategory.id,
            );
        } else {
            if (!filter.category.includes(subcategory.category)) {
                filter.category = [...filter.category, subcategory.category];
            }
            filter.subcategory = [...selectedSubcategoryIds, subcategory.id];
        }

        update();
    }

    function showSubcategoryOverlay(
        category: CategorySelectItem,
        hasSubcategories: boolean,
        target: EventTarget | null,
    ) {
        if (!(target instanceof HTMLElement)) {
            hoveredCategoryId = undefined;
            subcategoryOverlayStyle = "";
            hideFilterTooltip();
            return;
        }

        const rect = target.getBoundingClientRect();
        showCategoryTooltip(category.text, rect);

        if (!hasSubcategories) {
            hoveredCategoryId = undefined;
            subcategoryOverlayStyle = "";
            return;
        }

        subcategoryOverlayStyle = [
            `top: ${rect.bottom}px`,
            `left: ${rect.left}px`,
            "max-width: calc(100vw - 2rem)",
        ].join("; ");
        hoveredCategoryId = category.value;
    }

    function hideSubcategoryOverlay(category: CategorySelectItem) {
        if (hoveredCategoryId === category.value) {
            hoveredCategoryId = undefined;
            subcategoryOverlayStyle = "";
        }
        hideFilterTooltip();
    }

    function startCategoryLongPress(
        e: PointerEvent,
        category: CategorySelectItem,
        hasSubcategories: boolean,
    ) {
        if (e.pointerType === "mouse" || !hasSubcategories) {
            return;
        }

        clearCategoryLongPress();
        longPressCategoryId = category.value;
        longPressStart = { x: e.clientX, y: e.clientY };
        const target = e.currentTarget;

        longPressTimer = setTimeout(() => {
            suppressNextCategoryClick(category.value);
            showSubcategoryOverlay(category, hasSubcategories, target);
            longPressTimer = undefined;
        }, 450);
    }

    function moveCategoryLongPress(e: PointerEvent) {
        if (!longPressStart || e.pointerType === "mouse") {
            return;
        }

        const deltaX = Math.abs(e.clientX - longPressStart.x);
        const deltaY = Math.abs(e.clientY - longPressStart.y);
        if (deltaX > 10 || deltaY > 10) {
            clearCategoryLongPress();
        }
    }

    function clearCategoryLongPress() {
        if (longPressTimer) {
            clearTimeout(longPressTimer);
        }
        longPressTimer = undefined;
        longPressCategoryId = undefined;
        longPressStart = undefined;
    }

    function handleCategoryPointerUp(e: PointerEvent) {
        if (
            e.pointerType !== "mouse" &&
            longPressCategoryId &&
            !longPressTimer
        ) {
            e.preventDefault();
        }

        clearCategoryLongPress();
    }

    function suppressNextCategoryClick(categoryId: string) {
        clearSuppressedCategoryClick();
        suppressCategoryClick = categoryId;
        suppressCategoryClickTimer = setTimeout(() => {
            clearSuppressedCategoryClick();
        }, 700);
    }

    function clearSuppressedCategoryClick() {
        if (suppressCategoryClickTimer) {
            clearTimeout(suppressCategoryClickTimer);
        }
        suppressCategoryClickTimer = undefined;
        suppressCategoryClick = undefined;
    }

    function showFilterTooltip(text: string, target: EventTarget | null) {
        if (!(target instanceof HTMLElement)) {
            hideFilterTooltip();
            return;
        }

        showCategoryTooltip(text, target.getBoundingClientRect());
    }

    function hideFilterTooltip() {
        categoryTooltip = "";
        categoryTooltipStyle = "";
    }

    function showCategoryTooltip(text: string, rect: DOMRect) {
        const tooltipWidth = text.length * 7 + 20;
        const viewportPadding = 8;
        const left = Math.max(
            viewportPadding,
            Math.min(
                rect.left + rect.width / 2 - tooltipWidth / 2,
                window.innerWidth - viewportPadding - tooltipWidth,
            ),
        );
        categoryTooltip = text;
        categoryTooltipStyle = [
            `top: calc(${rect.top}px + var(--tooltip-offset-top))`,
            `left: ${left}px`,
            `width: ${tooltipWidth}px`,
            "background: var(--tooltip-background)",
            "border-radius: var(--tooltip-border-radius)",
            "color: var(--tooltip-color)",
            "font-size: var(--tooltip-font-size)",
            "padding: var(--tooltip-padding)",
        ].join("; ");
    }

    function handleSubcategoryOverlayFocusOut(
        e: FocusEvent,
        category: CategorySelectItem,
    ) {
        const nextTarget = e.relatedTarget;
        if (
            nextTarget instanceof Node &&
            (e.currentTarget as HTMLElement).contains(nextTarget)
        ) {
            return;
        }

        hideSubcategoryOverlay(category);
    }

</script>

{#snippet categoryButton(category: CategorySelectItem)}
    {@const selected = filter.category.includes(category.value)}
    {@const hasSubcategories = $subcategories.some(
        (subcategory) => subcategory.category === category.value,
    )}
    {@const selectedSubcategoriesForCategory = selectedSubcategoryIds.filter(
        (id) =>
            noSubcategoryFilterCategory(id) === category.value ||
            $subcategories.some(
                (subcategory) =>
                    subcategory.id === id &&
                    subcategory.category === category.value,
            ),
    )}
    {@const noSubcategorySelected = selectedSubcategoryIds.includes(
        noSubcategoryFilterValue(category.value),
    )}
    {@const noSubcategoryInherited =
        selected && selectedSubcategoriesForCategory.length === 0}
    {@const noSubcategoryActive =
        noSubcategorySelected || noSubcategoryInherited}
    <div
        class="relative shrink-0"
        role="presentation"
        onmouseenter={(e) =>
            showSubcategoryOverlay(
                category,
                hasSubcategories,
                e.currentTarget,
            )}
        onmouseleave={() => hideSubcategoryOverlay(category)}
        onfocusin={(e) =>
            showSubcategoryOverlay(
                category,
                hasSubcategories,
                e.currentTarget,
            )}
        onfocusout={(e) => handleSubcategoryOverlayFocusOut(e, category)}
        onpointerdown={(e) =>
            startCategoryLongPress(e, category, hasSubcategories)}
        onpointermove={moveCategoryLongPress}
        onpointerup={handleCategoryPointerUp}
        onpointercancel={clearCategoryLongPress}
        oncontextmenu={(e) => {
            if (suppressCategoryClick === category.value) {
                e.preventDefault();
            }
        }}
    >
        <button
            type="button"
            aria-label={category.text}
            aria-pressed={selected}
            class="relative flex h-10 w-10 items-center justify-center rounded-md border transition-colors focus:outline-none focus:ring-1 focus:ring-inset focus:ring-input-ring"
            class:border-primary={selected}
            class:bg-primary={selected}
            class:text-white={selected}
            class:border-input-border={!selected}
            class:bg-input-background={!selected}
            class:text-gray-500={!selected}
            class:hover:bg-menu-item-background-hover={!selected}
            onclick={(e) => handleCategoryClick(e, category)}
        >
            <i class="fa {category.icon} text-2xl"></i>
            {#if selectedSubcategoriesForCategory.length > 0}
                <i
                    class="fa fa-filter absolute left-1 top-1 text-[8px] text-white"
                ></i>
            {/if}
        </button>
        {#if hoveredCategoryId === category.value && hoveredCategoryItem && hoveredSubcategories.length}
            <div class="fixed z-20 min-w-max pt-1" style={subcategoryOverlayStyle}>
                <div
                    class="rounded-md border border-input-border bg-menu-background p-2 shadow-lg"
                >
                    <p class="mb-2 text-xs font-medium text-gray-500">
                        {hoveredCategoryItem.text}
                    </p>
                    <div class="flex items-center gap-2">
                        <button
                            type="button"
                            aria-label={$_("no-subcategory")}
                            aria-pressed={noSubcategoryActive}
                            class="relative flex h-10 w-10 items-center justify-center rounded-md border transition-colors focus:outline-none focus:ring-1 focus:ring-inset focus:ring-input-ring"
                            class:border-primary={noSubcategoryActive}
                            class:bg-primary={noSubcategoryActive}
                            class:text-white={noSubcategoryActive}
                            class:opacity-70={noSubcategoryInherited}
                            class:border-input-border={!noSubcategoryActive}
                            class:bg-input-background={!noSubcategoryActive}
                            class:text-gray-500={!noSubcategoryActive}
                            class:hover:bg-menu-item-background-hover={!noSubcategoryActive}
                            onmouseenter={(e) =>
                                showFilterTooltip(
                                    $_("no-subcategory"),
                                    e.currentTarget,
                                )}
                            onmouseleave={hideFilterTooltip}
                            onfocus={(e) =>
                                showFilterTooltip(
                                    $_("no-subcategory"),
                                    e.currentTarget,
                                )}
                            onblur={hideFilterTooltip}
                            onclick={() => toggleNoSubcategoryFilter(category)}
                        >
                            <i class="fa {category.icon} text-2xl"></i>
                            <i
                                class="fa-regular fa-circle absolute -bottom-1 -right-1 rounded-full bg-background text-[10px] text-content"
                                class:text-white={noSubcategoryActive}
                            ></i>
                        </button>
                        <div class="h-8 border-l border-separator"></div>
                        {#each hoveredSubcategories as subcategory}
                            {@const subcategorySelected = selectedSubcategoryIds.includes(subcategory.id)}
                            {@const subcategoryInherited = selected && selectedSubcategoriesForCategory.length === 0}
                            {@const subcategoryActive = subcategorySelected || subcategoryInherited}
                            {@const subcategoryLabel = displaySubcategoryLabel(subcategory, $locale)}
                            {@const badge = displaySubcategoryShortBadge(
                                subcategory,
                                $locale,
                            )}
                            {@const badgeIcon = displaySubcategoryBadgeIcon(subcategory)}
                            <button
                                type="button"
                                aria-label={subcategoryLabel}
                                aria-pressed={subcategoryActive}
                                class="relative flex h-10 w-10 items-center justify-center rounded-md border transition-colors focus:outline-none focus:ring-1 focus:ring-inset focus:ring-input-ring"
                                class:border-primary={subcategoryActive}
                                class:bg-primary={subcategoryActive}
                                class:text-white={subcategoryActive}
                                class:opacity-70={subcategoryInherited}
                                class:border-input-border={!subcategoryActive}
                                class:bg-input-background={!subcategoryActive}
                                class:text-gray-500={!subcategoryActive}
                                class:hover:bg-menu-item-background-hover={!subcategoryActive}
                                onmouseenter={(e) =>
                                    showFilterTooltip(
                                        subcategoryLabel,
                                        e.currentTarget,
                                    )}
                                onmouseleave={hideFilterTooltip}
                                onfocus={(e) =>
                                    showFilterTooltip(
                                        subcategoryLabel,
                                        e.currentTarget,
                                    )}
                                onblur={hideFilterTooltip}
                                onclick={() => toggleSubcategoryFilter(subcategory)}
                            >
                                <i
                                    class="fa {displaySubcategoryIcon(
                                        subcategory,
                                        hoveredCategoryItem,
                                    )} text-2xl"
                                ></i>
                                {#if badgeIcon}
                                    <i
                                        class="fa {badgeIcon} absolute right-0.5 top-0.5 text-[10px] text-gray-500"
                                        class:text-white={subcategoryActive}
                                    ></i>
                                {/if}
                                {#if badge}
                                    <span
                                        class="absolute -bottom-1 -right-1 max-w-10 truncate rounded-sm border border-input-border bg-background px-0.5 text-[7px] font-semibold leading-3 text-content"
                                    >
                                        {badge}
                                    </span>
                                {/if}
                            </button>
                        {/each}
                    </div>
                </div>
            </div>
        {/if}
    </div>
{/snippet}

<div>
    <p class="text-sm font-medium pb-2">{$_("categories")}</p>
    <div bind:this={categoryListElement} class="flex gap-2 overflow-visible pb-2">
        {#each visibleCategoryItems as category}
            {@render categoryButton(category)}
        {/each}

        {#if overflowCategoryItems.length}
            <div
                class="relative shrink-0"
                onfocusout={(e) => {
                    const nextTarget = e.relatedTarget;
                    if (
                        nextTarget instanceof Node &&
                        (e.currentTarget as HTMLElement).contains(nextTarget)
                    ) {
                        return;
                    }

                    closeOverflow();
                }}
            >
                <button
                    type="button"
                    aria-label={$_("more")}
                    aria-expanded={overflowExpanded}
                    class="relative flex h-10 w-10 items-center justify-center rounded-md border transition-colors focus:outline-none focus:ring-1 focus:ring-inset focus:ring-input-ring"
                    class:border-primary={overflowHasActiveFilters}
                    class:bg-primary={overflowHasActiveFilters}
                    class:text-white={overflowHasActiveFilters}
                    class:border-input-border={!overflowHasActiveFilters}
                    class:bg-input-background={!overflowHasActiveFilters}
                    class:text-gray-500={!overflowHasActiveFilters}
                    class:hover:bg-menu-item-background-hover={!overflowHasActiveFilters}
                    onclick={toggleOverflow}
                >
                    <span class="text-xs font-semibold">
                        +{overflowCategoryItems.length}
                    </span>
                    {#if overflowHasActiveFilters}
                        <i
                            class="fa fa-filter absolute left-1 top-1 text-[8px] text-white"
                        ></i>
                    {/if}
                </button>

                {#if overflowExpanded}
                    <div
                        class="absolute right-0 top-full z-10 mt-1 rounded-md border border-input-border bg-menu-background p-2 shadow-lg"
                    >
                        <div
                            class="grid gap-2"
                            style="grid-template-columns: repeat(4, 2.5rem);"
                        >
                            {#each overflowCategoryItems as category}
                                {@render categoryButton(category)}
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>
        {/if}
    </div>
    {#if categoryTooltip}
        <div
            class="fixed z-30 pointer-events-none whitespace-nowrap"
            style={categoryTooltipStyle}
        >
            {categoryTooltip}
        </div>
    {/if}
</div>
{#if activeHiddenCategories.length}
    <div
        class="mt-3 rounded-xl border border-yellow-500/40 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-800 dark:text-yellow-200"
    >
        <i class="fa fa-warning mr-2"></i>
        {$_("category-filter-hidden-active", {
            values: {
                categories: activeHiddenCategories.join(", "),
            },
        })}
    </div>
{/if}
