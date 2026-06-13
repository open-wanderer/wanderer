<script lang="ts">
    import Toggle from "$lib/components/base/toggle.svelte";
    import type { Category } from "$lib/models/category";
    import type { UserCategoryPreference } from "$lib/models/category_preference";
    import type { Subcategory } from "$lib/models/subcategory";
    import {
        categoryPreferences,
        category_preferences_index,
        category_preferences_reorder,
        category_preferences_save,
    } from "$lib/stores/category_preference_store";
    import {
        subcategoryPreferences,
        subcategory_preferences_save,
    } from "$lib/stores/subcategory_preference_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import {
        displayCategoryName,
        displaySubcategoryLabel,
        preferenceForCategory,
        sortedCategoriesByPreference,
        subcategoryVisible,
    } from "$lib/util/category_util";
    import { tick } from "svelte";
    import { _, locale } from "svelte-i18n";

    let { data } = $props();

    let savingCategory = $state<string | null>(null);
    let reordering = $state(false);
    let listElement: HTMLOListElement;
    let dragIndex = $state<number | null>(null);
    let insertBefore = $state<number | null>(null);
    let pointerId: number | null = null;
    let orderedCategories = $derived(
        sortedCategoriesByPreference(
            data.categories,
            $categoryPreferences,
            $locale,
        ),
    );

    function preference(category: Category): UserCategoryPreference | undefined {
        return preferenceForCategory($categoryPreferences, category.id);
    }

    function categoryIcon(category: Category) {
        const icon = category.icon?.trim().replace(/^fa-/, "");
        return icon ? `fa-${icon}` : "fa-layer-group";
    }

    function subcategoryIcon(subcategory: Subcategory) {
        const icon = subcategory.icon?.trim().replace(/^fa-/, "");
        return icon ? `fa-${icon}` : "";
    }

    function subcategoriesForCategory(category: Category): Subcategory[] {
        return data.subcategories.filter(
            (subcategory: Subcategory) => subcategory.category === category.id,
        );
    }

    function isSubcategoryVisible(subcategory: Subcategory): boolean {
        return subcategoryVisible(subcategory.id, $subcategoryPreferences);
    }

    async function toggleSubcategoryVisibility(subcategory: Subcategory) {
        savingCategory = subcategory.category;
        try {
            await subcategory_preferences_save({
                subcategory: subcategory.id,
                visible: !isSubcategoryVisible(subcategory),
            });
            show_toast({
                type: "success",
                icon: "check",
                text: $_("settings-saved"),
            });
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-saving-settings"),
            });
        } finally {
            savingCategory = null;
        }
    }

    async function updatePreference(
        category: Category,
        patch: Partial<UserCategoryPreference>,
    ) {
        const current = preference(category);
        savingCategory = category.id;
        try {
            await category_preferences_save({
                category: category.id,
                exclude_search:
                    patch.exclude_search ?? current?.exclude_search ?? false,
                hide_design: patch.hide_design ?? current?.hide_design ?? false,
                exclude_federated:
                    patch.exclude_federated ??
                    current?.exclude_federated ??
                    false,
            });
            show_toast({
                type: "success",
                icon: "check",
                text: $_("settings-saved"),
            });
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-saving-settings"),
            });
        } finally {
            savingCategory = null;
        }
    }

    async function reorderCategory(fromIndex: number, toIndex: number) {
        if (
            fromIndex < 0 ||
            toIndex < 0 ||
            fromIndex >= orderedCategories.length ||
            toIndex >= orderedCategories.length ||
            fromIndex === toIndex
        ) {
            return;
        }

        const categoryIds = orderedCategories.map((item) => item.id);
        const [categoryId] = categoryIds.splice(fromIndex, 1);
        categoryIds.splice(toIndex, 0, categoryId);

        reordering = true;
        try {
            await category_preferences_reorder(categoryIds);
            await category_preferences_index();
            show_toast({
                type: "success",
                icon: "check",
                text: $_("settings-saved"),
            });
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-saving-settings"),
            });
        } finally {
            reordering = false;
        }
    }

    function isValidInsert(pos: number): boolean {
        return (
            dragIndex !== null &&
            insertBefore === pos &&
            pos !== dragIndex &&
            pos !== dragIndex + 1
        );
    }

    function getInsertPosition(clientY: number): number {
        const items = Array.from(
            listElement.querySelectorAll<HTMLElement>("li[data-category-index]"),
        );

        for (const item of items) {
            const itemIndex = Number(item.dataset.categoryIndex);
            const rect = item.getBoundingClientRect();
            if (clientY < rect.top + rect.height / 2) {
                return itemIndex;
            }
        }

        return orderedCategories.length;
    }

    function clearDragState() {
        dragIndex = null;
        insertBefore = null;
        pointerId = null;
    }

    async function handleKeyDown(e: KeyboardEvent, index: number) {
        if (reordering) {
            return;
        }

        let toIndex: number | null = null;
        if (e.key === "ArrowUp" && index > 0) {
            e.preventDefault();
            toIndex = index - 1;
        } else if (
            e.key === "ArrowDown" &&
            index < orderedCategories.length - 1
        ) {
            e.preventDefault();
            toIndex = index + 1;
        }
        if (toIndex === null) {
            return;
        }

        await reorderCategory(index, toIndex);
        await tick();
        const handles = listElement.querySelectorAll<HTMLElement>(
            "li[data-category-index] button.drag-hitarea",
        );
        handles[toIndex]?.focus();
    }

    function handlePointerDown(e: PointerEvent, index: number) {
        if (reordering || e.button !== 0) {
            return;
        }

        e.preventDefault();
        dragIndex = index;
        insertBefore = index;
        pointerId = e.pointerId;
        (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    }

    function handlePointerMove(e: PointerEvent) {
        if (pointerId !== e.pointerId || dragIndex === null) {
            return;
        }

        e.preventDefault();
        insertBefore = getInsertPosition(e.clientY);
    }

    async function handlePointerUp(e: PointerEvent) {
        if (pointerId !== e.pointerId) {
            return;
        }

        e.preventDefault();
        if (dragIndex !== null && insertBefore !== null) {
            const toIndex =
                dragIndex < insertBefore ? insertBefore - 1 : insertBefore;
            if (toIndex !== dragIndex) {
                await reorderCategory(dragIndex, toIndex);
            }
        }

        clearDragState();
    }
</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>

<h2 class="text-2xl font-semibold">{$_("category-preferences")}</h2>
<p class="mt-2 text-sm text-gray-500">
    {$_("category-preferences-description")}
</p>
<hr class="mt-4 mb-6 border-input-border" />

<div
    class="hidden px-4 pb-1 text-sm font-medium md:grid md:grid-cols-[4rem_minmax(11rem,1fr)_max-content] md:gap-x-5"
>
    <div></div>
    <div></div>
    <div class="grid grid-cols-[6rem_7.25rem] justify-items-center gap-x-2">
        <span class="text-center">
            {$_("category-preference-visible")}
        </span>
        <span class="text-center">
            {$_("category-preference-federated")}
        </span>
    </div>
</div>

<ol bind:this={listElement} class="category-list flex flex-col gap-3 py-2">
    {#each orderedCategories as category, index}
        {@const currentPreference = preference(category)}
        {@const categoryHidden =
            (currentPreference?.exclude_search ?? false) ||
            (currentPreference?.hide_design ?? false)}
        {@const categoryName = displayCategoryName(category, $locale)}
        {@const childSubcategories = subcategoriesForCategory(category)}
        <li
            data-category-index={index}
            class="rounded-xl border border-input-border p-4 transition-colors hover:bg-secondary-hover"
            class:opacity-50={dragIndex === index}
            class:drop-above={isValidInsert(index)}
            class:drop-below={index === orderedCategories.length - 1 &&
                isValidInsert(orderedCategories.length)}
        >
            <div
                class="grid grid-cols-[4rem_minmax(0,1fr)] gap-x-5 gap-y-3 md:grid-cols-[4rem_minmax(11rem,1fr)_max-content]"
            >
                <button
                    class="drag-hitarea absolute inset-y-0 left-0 z-10 w-24 rounded-md p-0 disabled:cursor-not-allowed disabled:opacity-50"
                    type="button"
                    disabled={reordering}
                    aria-label={`${$_("category-preferences")}: ${categoryName}`}
                    aria-keyshortcuts="ArrowUp ArrowDown"
                    onkeydown={(e) => handleKeyDown(e, index)}
                    onpointerdown={(e) => handlePointerDown(e, index)}
                    onpointermove={handlePointerMove}
                    onpointerup={handlePointerUp}
                    onpointercancel={clearDragState}
                    onlostpointercapture={clearDragState}
                ></button>
                <span
                    class="drag-handle absolute inset-y-0 left-0 w-12 rounded-md"
                ></span>

                <span
                    class="category-icon pointer-events-none relative z-0 col-start-1 row-start-1 flex h-16 w-16 items-center justify-center rounded-2xl bg-input-background text-4xl text-content"
                >
                    <i class="fa {categoryIcon(category)}"></i>
                    <span
                        class="absolute -bottom-1 -right-1 flex h-6 min-w-6 items-center justify-center rounded-full border border-input-border bg-background px-1 text-xs font-semibold leading-none text-gray-500"
                    >
                        {index + 1}
                    </span>
                </span>

                <div class="col-start-2 min-h-16 min-w-0 self-start py-1">
                    <h3 class="truncate text-lg font-semibold leading-6">
                        {categoryName}
                    </h3>

                    {#if childSubcategories.length > 0}
                        <div
                            class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs leading-4"
                        >
                            {#each childSubcategories as subcategory (subcategory.id)}
                                {@const subcategoryLabel = displaySubcategoryLabel(
                                    subcategory,
                                    $locale,
                                )}
                                {@const visibleSubcategory = isSubcategoryVisible(subcategory)}
                                <button
                                    type="button"
                                    class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-sm text-left transition-colors hover:text-content focus:outline-none focus:ring-1 focus:ring-inset focus:ring-input-ring"
                                    class:text-content={visibleSubcategory}
                                    class:text-gray-300={!visibleSubcategory}
                                    class:dark:text-gray-700={!visibleSubcategory}
                                    aria-pressed={visibleSubcategory}
                                    aria-label={subcategoryLabel}
                                    disabled={savingCategory === category.id}
                                    onclick={() =>
                                        toggleSubcategoryVisibility(subcategory)}
                                >
                                    {#if subcategoryIcon(subcategory)}
                                        <i
                                            class="fa {subcategoryIcon(
                                                subcategory,
                                            )} text-xs opacity-70"
                                        ></i>
                                    {/if}
                                    {subcategoryLabel}
                                </button>
                            {/each}
                        </div>
                    {/if}
                </div>

                <div
                    class="col-span-2 row-start-2 grid w-max grid-cols-[6rem_7.25rem] items-center justify-items-center gap-x-2 gap-y-2 pl-[5.25rem] md:col-span-1 md:col-start-3 md:row-start-1 md:self-center md:justify-self-end md:pl-0"
                >
                    <span class="text-center text-sm font-medium md:hidden">
                        {$_("category-preference-visible")}
                    </span>
                    <span class="text-center text-sm font-medium md:hidden">
                        {$_("category-preference-federated")}
                    </span>
                    <div class="flex justify-center md:self-center">
                        <Toggle
                            value={!categoryHidden}
                            disabled={savingCategory === category.id}
                            onchange={(value) =>
                                updatePreference(
                                    category,
                                    value
                                        ? {
                                              exclude_search: false,
                                              hide_design: false,
                                          }
                                        : {
                                              exclude_search: true,
                                              hide_design: true,
                                              exclude_federated: true,
                                          },
                                )}
                        ></Toggle>
                    </div>
                    <div class="flex justify-center md:self-center">
                        <Toggle
                            value={!(currentPreference?.exclude_federated ?? false)}
                            disabled={savingCategory === category.id}
                            onchange={(value) =>
                                updatePreference(category, {
                                    exclude_federated: !value,
                                })}
                        ></Toggle>
                    </div>
                </div>
            </div>
        </li>
    {/each}
</ol>

<style>
    li {
        position: relative;
    }

    .drag-handle {
        pointer-events: none;
        background-image:
            radial-gradient(
                circle at 0.25rem 0.4375rem,
                rgba(var(--content), 0.5) 1.35px,
                transparent 1.65px
            ),
            radial-gradient(
                circle at 0.75rem 0.125rem,
                rgba(var(--content), 0.38) 1.35px,
                transparent 1.65px
            ),
            radial-gradient(
                circle at 1.25rem 0.4375rem,
                rgba(var(--content), 0.27) 1.35px,
                transparent 1.65px
            ),
            radial-gradient(
                circle at 1.75rem 0.125rem,
                rgba(var(--content), 0.17) 1.35px,
                transparent 1.65px
            ),
            radial-gradient(
                circle at 2.25rem 0.4375rem,
                rgba(var(--content), 0.1) 1.35px,
                transparent 1.65px
            ),
            radial-gradient(
                circle at 2.75rem 0.125rem,
                rgba(var(--content), 0.05) 1.35px,
                transparent 1.65px
            );
        background-repeat: repeat-y;
        background-size: 3rem 0.75rem;
        opacity: 0.82;
        transition: opacity 0.15s ease-in-out;
    }

    .drag-hitarea {
        cursor: grab;
        touch-action: none;
    }

    .drag-hitarea:active {
        cursor: grabbing;
    }

    li.drop-above::before,
    li.drop-below::after {
        content: "";
        position: absolute;
        left: 0;
        right: 0;
        height: 2px;
        border-radius: 9999px;
        background: rgba(var(--content));
    }

    li.drop-above::before {
        top: -6px;
    }

    li.drop-below::after {
        bottom: -6px;
    }

    @media (hover: hover) and (pointer: fine) {
        .drag-handle {
            opacity: 0.55;
        }

        li:hover .drag-handle,
        li:focus-within .drag-handle {
            opacity: 1;
        }
    }
</style>
