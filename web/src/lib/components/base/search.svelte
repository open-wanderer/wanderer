<script module lang="ts">
    export type SearchItem = {
        text: string;
        description?: string;
        value: any;
        icon: string;
    };
</script>

<script lang="ts">
    import type { Snippet } from "svelte";
    import { fade } from "svelte/transition";
    import SearchList from "./search_list.svelte";
    import TextField from "./text_field.svelte";

    interface Props {
        maxSearchLength?: number;
        timeBetweenUpdates?: number,
        value?: string;
        items?: SearchItem[];
        placeholder?: string;
        large?: boolean;
        extraClasses?: string;
        label?: string;
        clearAfterSelect?: boolean;
        prepend?: Snippet<[any]>;
        onupdate?: (q: string) => void;
        onclick?: (item: SearchItem) => void;
    }

    let {
        maxSearchLength = 5,
        timeBetweenUpdates = 500,
        value = $bindable(""),
        items = $bindable([]),
        placeholder = "Search...",
        large = false,
        extraClasses = "",
        label = "",
        clearAfterSelect = true,
        prepend,
        onupdate,
        onclick,
    }: Props = $props();

    let lastSearch: string = "";
    let searching: boolean = $state(false);
    let typingTimer!: any;

    let dropDownOpen = $derived(
        value.length > 0 && items.length > 0 && searching,
    );

    function onSearchType() {
        clearTimeout(typingTimer);
        if (Math.abs(value.length - lastSearch.length) > maxSearchLength) {
            update(value);
            return;
        }
        typingTimer = setTimeout(() => {
            update(value);
        }, timeBetweenUpdates);
    }

    function update(q: string) {
        lastSearch = q;
        onupdate?.(q);
    }

    function handleItemClick(e: Event, item: SearchItem) {
        e.stopPropagation();
        searching = false;
        onclick?.(item);
        if (clearAfterSelect) {
            clear();
        }
    }

    function clear() {
        value = "";
        items = [];
        update(value);
    }
</script>

<div class="relative {extraClasses}">
    <span
        class="absolute {large
            ? 'bottom-[31px]'
            : 'bottom-[25px]'} translate-y-1/2 left-0 pl-4"
    >
        <i class="fa fa-search" class:text-xl={large}></i>
    </span>
    {#if value.length > 0}
        <button
            aria-label="Clear"
            type="button"
            class="btn-icon absolute {large
                ? 'bottom-[31px]'
                : 'bottom-[25px]'} translate-y-1/2 right-0 mr-2"
            onclick={clear}
            in:fade={{ duration: 150 }}
            out:fade={{ duration: 150 }}
        >
            <i class="fa fa-close text-sm"></i>
        </button>
    {/if}
    <TextField
        type="search"
        name="q"
        autocomplete="off"
        {label}
        extraClasses={large
            ? "px-12 py-4 text-xl min-w-80 w-[33vw] max-w-[532px] rounded-xl"
            : "px-10"}
        {placeholder}
        bind:value
        oninput={onSearchType}
        onfocusin={() => (searching = true)}
        onfocusout={() => (searching = false)}
    ></TextField>

    {#if dropDownOpen}
        <SearchList {items} extraClasses="w-full" {prepend} onclick={handleItemClick}></SearchList>
    {/if}
</div>
