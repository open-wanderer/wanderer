<script lang="ts">
    import { _ } from "svelte-i18n";
    import type { SelectItem } from "./select.svelte";
    import Chip from "./chip.svelte";

    interface Props {
        items?: SelectItem[];
        value?: SelectItem[];
        label?: string;
        name?: string;
        placeholder?: string;
        onchange?: (value: SelectItem[], changedItem?: SelectItem) => void;
    }

    let {
        items = [],
        value = $bindable([]),
        label = "",
        name = "",
        placeholder = "",
        onchange,
    }: Props = $props();

    let showDropdown = $state(false);
    let dropdownRef = $state();

    function toggleItem(item: SelectItem) {
        const itemIndex = value.findIndex((i) => i.value == item.value);

        if (itemIndex >= 0) {
            value = value.filter((_, index) => index !== itemIndex);
        } else {
            value = [...value, item];
        }

        onchange?.(value, item);
    }

    function removeItem(e: Event, item: SelectItem) {
        e.stopPropagation();
        value = value.filter((i) => i !== item);
        onchange?.(value);
    }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<div class="relative max-w-full">
    {#if label.length}
        <label for={name} class="text-sm font-medium pb-1 block">
            {label}
        </label>
    {/if}
    <div
        role="presentation"
        class="relative min-w-44 w-full flex flex-wrap items-center gap-2 border border-input-border bg-input-background min-h-[50px] p-3 rounded-md transition-colors focus:border-input-border-focus focus:outline-none focus:ring-0 cursor-pointer pr-6"
        onclick={() => (showDropdown = !showDropdown)}
    >
        {#if value.length === 0}
            <span class="text-gray-400">{placeholder}</span>
        {/if}
        {#each value as item}
            <Chip
                text={$_(item.text)}
                closable
                onclick={(e) => removeItem(e, item)}
            ></Chip>
        {/each}
        <i
            class="fa fa-caret-down absolute right-4 top-1/2 -translate-y-1/2 text-gray-500 transition-transform"
            class:rotate-180={showDropdown}
        ></i>
    </div>

    <!-- Dropdown menu -->
    {#if showDropdown}
        <div
            bind:this={dropdownRef}
            class="absolute z-10 mt-1 w-full bg-menu-background border border-input-border rounded-md max-h-40 overflow-y-auto shadow-lg"
        >
            {#each items as item}
                <button
                    onclick={() => toggleItem(item)}
                    class="px-3 py-2 hover:bg-menu-item-background-hover cursor-pointer flex justify-between items-center w-full"
                >
                    <span>{$_(item.text)}</span>
                    {#if value.findIndex((i) => i.value == item.value) >= 0}
                        <div class="ml-auto">
                            <i class="fa fa-check"></i>
                        </div>
                    {/if}
                </button>
            {/each}
        </div>
    {/if}
</div>
