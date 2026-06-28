<script module lang="ts">
    export type CategoryPickerSelection = {
        category: string;
        subcategory: string;
    };
</script>

<script lang="ts">
    import { categories, categories_index } from "$lib/stores/category_store";
    import {
        categoryPreferences,
        category_preferences_index,
    } from "$lib/stores/category_preference_store";
    import {
        subcategories,
        subcategories_index,
    } from "$lib/stores/subcategory_store";
    import {
        subcategoryPreferences,
        subcategory_preferences_index,
    } from "$lib/stores/subcategory_preference_store";
    import {
        designSelectableCategories,
        displayCategoryIcon,
        displayCategoryName,
        displaySubcategoryBadgeIcon,
        displaySubcategoryIcon,
        displaySubcategoryLabel,
        sortedSubcategoriesByPreference,
        subcategoryVisible,
    } from "$lib/util/category_util";
    import { onDestroy, onMount } from "svelte";
    import { _, locale } from "svelte-i18n";

    type CategoryPickerItem = {
        text: string;
        detail?: string;
        value: string;
        icon: string;
        badgeIcon?: string;
    };

    interface Props {
        value?: string;
        label?: string;
        placeholder?: string;
        hiddenInputs?: boolean;
        loadData?: boolean;
        currentCategoryId?: string | null;
        fixedDropdown?: boolean;
        onchange?: (selection: CategoryPickerSelection) => void;
    }

    let {
        value = "",
        label = $_("category"),
        placeholder = $_("category"),
        hiddenInputs = false,
        loadData = false,
        currentCategoryId = null,
        fixedDropdown = false,
        onchange,
    }: Props = $props();

    let open = $state(false);
    let buttonElement: HTMLButtonElement;
    let dropdownElement: HTMLUListElement | undefined = $state();
    let dropdownStyle = $state("");
    let fixedDropdownListenersAttached = false;
    let selectedSubcategoryId = $derived(
        value.startsWith("subcategory:") ? value.replace("subcategory:", "") : "",
    );

    let items: CategoryPickerItem[] = $derived(
        designSelectableCategories(
            $categories,
            $categoryPreferences,
            $locale,
            currentCategoryId,
        ).flatMap((category): CategoryPickerItem[] => [
            {
                text: displayCategoryName(category, $locale),
                value: `category:${category.id}`,
                icon: displayCategoryIcon(category),
            },
            ...sortedSubcategoriesByPreference(
                $subcategories.filter(
                    (subcategory) =>
                        subcategory.category === category.id &&
                        (subcategoryVisible(
                            subcategory.id,
                            $subcategoryPreferences,
                        ) ||
                            subcategory.id === selectedSubcategoryId),
                ),
                $subcategoryPreferences,
                $locale,
            )
                .map((subcategory) => ({
                    text: displayCategoryName(category, $locale),
                    detail: displaySubcategoryLabel(subcategory, $locale),
                    value: `subcategory:${subcategory.id}`,
                    icon: displaySubcategoryIcon(subcategory, category),
                    badgeIcon: displaySubcategoryBadgeIcon(subcategory),
                })),
        ]),
    );

    let selectedItem = $derived(items.find((item) => item.value === value));
    let hiddenCategory = $derived(resolveSelection(value)?.category ?? "");
    let hiddenSubcategory = $derived(resolveSelection(value)?.subcategory ?? "");

    onMount(async () => {
        if (!loadData) {
            return;
        }

        await Promise.all([
            categories_index(),
            subcategories_index(),
            category_preferences_index(),
            subcategory_preferences_index(),
        ]);
    });

    function resolveSelection(nextValue: string): CategoryPickerSelection | undefined {
        if (nextValue.startsWith("subcategory:")) {
            const subcategoryId = nextValue.replace("subcategory:", "");
            const subcategory = $subcategories.find(
                (item) => item.id === subcategoryId,
            );

            if (!subcategory) {
                return undefined;
            }

            return {
                category: subcategory.category,
                subcategory: subcategory.id,
            };
        }

        if (nextValue.startsWith("category:")) {
            return {
                category: nextValue.replace("category:", ""),
                subcategory: "",
            };
        }

        return undefined;
    }

    function handlePickerClick(e: MouseEvent) {
        e.stopPropagation();
    }

    function handleWindowClick() {
        closePicker();
    }

    function updateDropdownPosition() {
        if (!fixedDropdown || !buttonElement) {
            dropdownStyle = "";
            return;
        }

        const rect = buttonElement.getBoundingClientRect();
        const viewportPadding = 16;
        const left = Math.max(
            viewportPadding,
            Math.min(rect.left, window.innerWidth - viewportPadding - rect.width),
        );

        dropdownStyle = [
            "position: fixed",
            `top: ${rect.bottom + 4}px`,
            `left: ${left}px`,
            `min-width: ${rect.width}px`,
            `max-height: min(18rem, calc(100vh - ${rect.bottom + viewportPadding + 4}px))`,
        ].join("; ");
    }

    function handleFixedDropdownResize() {
        if (!open || !fixedDropdown) {
            return;
        }

        updateDropdownPosition();
    }

    function handleFixedDropdownScroll(e: Event) {
        if (!open || !fixedDropdown) {
            return;
        }
        if (
            e.target instanceof Node &&
            dropdownElement?.contains(e.target)
        ) {
            return;
        }

        closePicker();
    }

    function attachFixedDropdownListeners() {
        if (!fixedDropdown || fixedDropdownListenersAttached) {
            return;
        }

        window.addEventListener("scroll", handleFixedDropdownScroll, true);
        window.addEventListener("resize", handleFixedDropdownResize);
        fixedDropdownListenersAttached = true;
    }

    function detachFixedDropdownListeners() {
        if (!fixedDropdownListenersAttached) {
            return;
        }

        window.removeEventListener("scroll", handleFixedDropdownScroll, true);
        window.removeEventListener("resize", handleFixedDropdownResize);
        fixedDropdownListenersAttached = false;
    }

    function togglePicker() {
        if (!open) {
            updateDropdownPosition();
            open = true;
            attachFixedDropdownListeners();
            return;
        }

        closePicker();
    }

    function closePicker() {
        open = false;
        detachFixedDropdownListeners();
    }

    function selectItem(item: CategoryPickerItem) {
        const selection = resolveSelection(item.value);
        if (!selection) {
            return;
        }

        value = item.value;
        onchange?.(selection);
        closePicker();
    }

    onDestroy(() => {
        detachFixedDropdownListeners();
    });
</script>

<svelte:window onmouseup={handleWindowClick} />

<div class="relative" role="presentation" onmouseup={handlePickerClick}>
    {#if hiddenInputs}
        <input type="hidden" name="category" value={hiddenCategory} />
        <input type="hidden" name="subcategory" value={hiddenSubcategory} />
    {/if}
    {#if label}
        <label for="category-picker" class="text-sm font-medium pb-1">
            {label}
        </label>
    {/if}
    <button
        id="category-picker"
        bind:this={buttonElement}
        type="button"
        class="relative flex h-10 w-full cursor-pointer items-center justify-between gap-3 rounded-md border border-input-border bg-input-background px-4 pr-10 transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0"
        onclick={togglePicker}
    >
        <span class="flex min-w-0 items-center gap-3">
            {#if selectedItem}
                <span class="relative w-4 shrink-0 text-center">
                    <i class="fa {selectedItem.icon}"></i>
                    {#if selectedItem.badgeIcon}
                        <i
                            class="fa {selectedItem.badgeIcon} absolute -right-1 -top-1 text-[8px]"
                        ></i>
                    {/if}
                </span>
                <span class="truncate">
                    {selectedItem.detail?.trim() ?? selectedItem.text.trim()}
                </span>
            {:else}
                <i
                    class="fa fa-shapes w-4 shrink-0 text-center text-gray-500"
                ></i>
                <span class="truncate text-gray-500">
                    {placeholder}
                </span>
            {/if}
        </span>
        <i
            class="fa fa-caret-down absolute right-4 top-1/2 -translate-y-1/2 text-gray-500 transition-transform"
            class:rotate-180={open}
        ></i>
    </button>
    {#if open}
        <ul
            bind:this={dropdownElement}
            class="{fixedDropdown
                ? 'fixed z-50 min-w-full w-max max-w-[calc(100vw-2rem)] overflow-y-auto rounded-md border border-input-border bg-menu-background shadow-lg'
                : 'absolute z-10 mt-1 max-h-72 min-w-full w-max max-w-[calc(100vw-2rem)] overflow-y-auto rounded-md border border-input-border bg-menu-background shadow-lg'}"
            style={dropdownStyle}
        >
            {#each items as item}
                <li>
                    <button
                        type="button"
                        class="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus"
                        class:bg-menu-item-background-focus={item.value === value}
                        onclick={() => selectItem(item)}
                    >
                        <span class="relative w-4 shrink-0 text-center">
                            <i class="fa {item.icon}"></i>
                            {#if item.badgeIcon}
                                <i
                                    class="fa {item.badgeIcon} absolute -right-1 -top-1 text-[8px]"
                                ></i>
                            {/if}
                        </span>
                        <span class="whitespace-nowrap">
                            {item.text.trim()}
                            {#if item.detail}
                                <span class="text-gray-500">
                                    / {item.detail.trim()}
                                </span>
                            {/if}
                        </span>
                    </button>
                </li>
            {/each}
        </ul>
    {/if}
</div>
