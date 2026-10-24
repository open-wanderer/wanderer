<script lang="ts">
    import type { SearchPosition, SearchExpansion } from "$lib/models/search_layout";
    import { tick, type Snippet } from "svelte";
    import { _ } from "svelte-i18n";
    import type { SelectItem } from "../base/select.svelte";

    interface Props {
        sort?: string;
        order?: "+" | "-";
        sortOptions: SelectItem[];
        display?: string;
        displayOptions?: SelectItem[];
        pageSize?: number;
        pageSizes?: SelectItem[];
        wide?: boolean;
        onsort?: (sort: string, order: "+" | "-") => void;
        ondisplay?: (display: string) => void;
        onpagesize?: (size: number) => void;
        onwidthtoggle?: () => void;
        selectionActions?: Snippet;
        buttonsOnly?: boolean;
        actions?: Snippet;
        searchPosition?: SearchPosition;
        searchExpansion?: SearchExpansion;
        onsearchposition?: (position: SearchPosition) => void;
        onsearchexpansion?: (expansion: SearchExpansion) => void;
    }
    let { sort, order = "+", sortOptions, display = "cards", displayOptions = [],
        pageSize = 25, pageSizes = [], wide = false, onsort, ondisplay, onpagesize, onwidthtoggle, selectionActions, buttonsOnly = false, actions, searchPosition = "top", searchExpansion = "overlay", onsearchposition, onsearchexpansion }: Props = $props();
    const id = $props.id();
    let root: HTMLDivElement;
    let sortButton = $state<HTMLButtonElement>();
    let viewButton = $state<HTMLButtonElement>();
    let open = $state<"sort" | "view" | null>(null);
    let above = $state(false);
    let maxHeight = $state(500);

    const displayIcons: Record<string, string> = { cards: "fa-grip", list: "fa-list-ul", table: "fa-table", map: "fa-map" };
    const directionKeys: Record<string, [string, string]> = {
        name: ["name-asc", "name-desc"], distance: ["distance-asc", "distance-desc"],
        duration: ["duration-asc", "duration-desc"], difficulty: ["difficulty-asc", "difficulty-desc"],
        elevation_gain: ["gain-asc", "gain-desc"], elevation_loss: ["loss-asc", "loss-desc"],
        like_count: ["likes-asc", "likes-desc"], created: ["created-asc", "created-desc"],
        date: ["date-asc", "date-desc"],
    };
    function sortLabel(direction = order) {
        if (sort === "relevance") return $_("trail-results-relevance");
        const keys = directionKeys[sort ?? ""];
        return keys ? $_(`trail-results-${keys[direction === "+" ? 0 : 1]}`) : $_("sort");
    }
    let viewLabel = $derived(displayOptions.find((item) => item.value === display)?.text ?? $_("trail-results-view"));

    function positionMenu() {
        if (!open) return;
        const button = open === "sort" ? sortButton : viewButton;
        if (!button) return;
        const bounds = button.getBoundingClientRect();
        const viewport = window.visualViewport;
        const top = viewport?.offsetTop ?? 0;
        const below = top + (viewport?.height ?? window.innerHeight) - bounds.bottom - 12;
        const over = bounds.top - top - 12;
        above = below < 330 && over > below;
        maxHeight = Math.max(0, above ? over : below);
    }
    async function toggle(menu: "sort" | "view", focusOption = false) {
        open = open === menu ? null : menu;
        positionMenu();
        if (open && focusOption) {
            await tick();
            focusSelectedOption();
        }
    }
    function focusSelectedOption() {
        const menu = root.querySelector(".result-menu");
        (menu?.querySelector<HTMLInputElement>("input:checked") ?? menu?.querySelector<HTMLInputElement>("input"))?.focus();
    }
    function close(restoreFocus = false) {
        const previous = open;
        open = null;
        if (restoreFocus) (previous === "sort" ? sortButton : viewButton)?.focus();
    }
    function outside(event: PointerEvent) {
        if (open && event.target instanceof Node && !root.contains(event.target)) close();
    }
    function focusOut(event: FocusEvent) {
        if (open && event.relatedTarget instanceof Node && !root.contains(event.relatedTarget)) close();
    }
    function keydown(event: KeyboardEvent) {
        if (!(event.target instanceof Node) || !root?.contains(event.target)) return;
        if (event.key === "Escape" && open) { event.preventDefault(); event.stopPropagation(); close(true); }
        else if (event.key === "ArrowDown" && (event.target === sortButton || event.target === viewButton)) {
            event.preventDefault();
            const menu = event.target === sortButton ? "sort" : "view";
            if (open !== menu) toggle(menu, true);
            else focusSelectedOption();
        }
    }
</script>

<svelte:window onpointerdown={outside} onkeydown={keydown} onresize={positionMenu} onscroll={positionMenu} />

<div class="result-controls" class:buttons-only={buttonsOnly} bind:this={root} onfocusout={focusOut}>
    <div class="result-summary">
        {#if selectionActions}{@render selectionActions()}{/if}
    </div>
    <div class="result-buttons">
        {#if onsort && sort}
        <div class="menu-anchor">
            <button type="button" class="control-button" bind:this={sortButton} aria-label={`${$_("sort")}: ${sortLabel()}`} title={`${$_("sort")}: ${sortLabel()}`} aria-expanded={open === "sort"} aria-controls={`${id}-sort`} onclick={() => toggle("sort")}>
                <i class={`fa ${order === "+" ? "fa-arrow-down-short-wide" : "fa-arrow-down-wide-short"}`} aria-hidden="true"></i>
                <span class="control-copy">{sortLabel()}</span><i class="fa fa-chevron-down chevron" aria-hidden="true"></i>
            </button>
            {#if open === "sort"}
            <div id={`${id}-sort`} class="result-menu" class:above role="region" aria-label={$_("sort")} style={`max-height: ${maxHeight}px`}>
                <fieldset>
                    <legend class="sort-heading">
                        <span>{$_("trail-results-sort-by")}</span>
                        {#if sort !== "relevance"}<button type="button" class="sort-direction-button"
                            aria-label={`${$_("trail-results-toggle-order")}: ${sortLabel()}`}
                            title={`${sortLabel()} · ${$_("trail-results-toggle-order")}`}
                            onclick={() => onsort?.(sort!, order === "+" ? "-" : "+")}>
                            <i class="fa fa-arrow-up" class:descending={order === "-"} aria-hidden="true"></i>
                        </button>{/if}
                    </legend>
                    {#each sortOptions as option}
                        <label class="menu-option" class:chosen={sort === option.value}>
                            <input type="radio" name={`${id}-sort-key`} value={option.value} checked={sort === option.value} onchange={() => onsort?.(option.value, order)} />
                            <span>{option.text}</span><i class="fa fa-check check" aria-hidden="true"></i>
                        </label>
                    {/each}
                </fieldset>
            </div>
            {/if}
        </div>
        {/if}
        {#if (ondisplay && displayOptions.length) || onsearchposition || onsearchexpansion}<div class="menu-anchor">
            <button type="button" class="control-button" bind:this={viewButton} aria-label={displayOptions.length ? `${$_("trail-results-view")}: ${viewLabel}` : $_("trail-results-view")} title={displayOptions.length ? `${$_("trail-results-view")}: ${viewLabel}` : $_("trail-results-view")} aria-expanded={open === "view"} aria-controls={`${id}-view`} onclick={() => toggle("view")}>
                <i class={`fa ${displayOptions.length ? displayIcons[display] ?? "fa-grip" : "fa-table-columns"}`} aria-hidden="true"></i><span class="control-copy">{$_("trail-results-view")}</span><i class="fa fa-chevron-down chevron" aria-hidden="true"></i>
            </button>
            {#if open === "view"}
            <div id={`${id}-view`} class="result-menu view-menu" class:above role="region" aria-label={$_("trail-results-view")} style={`max-height: ${maxHeight}px`}>
                {#if ondisplay && displayOptions.length}<fieldset><legend>{$_("trail-results-presentation")}</legend>
                    {#each displayOptions as option}
                        <label class="menu-option" class:chosen={display === option.value}>
                            <input type="radio" name={`${id}-display`} value={option.value} checked={display === option.value} onchange={() => ondisplay?.(option.value)} />
                            <i class={`fa ${displayIcons[option.value]}`} aria-hidden="true"></i><span>{option.text}</span><i class="fa fa-check check" aria-hidden="true"></i>
                        </label>
                    {/each}
                </fieldset>{/if}
                {#if onsearchposition}
                    <fieldset class="layout-section"><legend>{$_("search-layout-position")}</legend>
                        {#each ["top", "left", "right"] as position}
                            <label class="menu-option" class:chosen={searchPosition === position}>
                                <input type="radio" name={`${id}-search-position`} value={position} checked={searchPosition === position}
                                    onchange={() => onsearchposition?.(position as SearchPosition)} />
                                <span>{$_(`search-layout-${position}`)}</span><i class="fa fa-check check" aria-hidden="true"></i>
                            </label>
                        {/each}
                    </fieldset>
                {/if}
                {#if onsearchexpansion}
                    <fieldset class="layout-section"><legend>{$_("search-layout-expansion")}</legend>
                        {#each ["overlay", "push"] as expansion}
                            <label class="menu-option" class:chosen={searchExpansion === expansion}>
                                <input type="radio" name={`${id}-search-expansion`} value={expansion} checked={searchExpansion === expansion}
                                    onchange={() => onsearchexpansion?.(expansion as SearchExpansion)} />
                                <span>{$_(`search-layout-${expansion}`)}</span><i class="fa fa-check check" aria-hidden="true"></i>
                            </label>
                        {/each}
                    </fieldset>
                {/if}
                {#if onwidthtoggle}
                    <div class="width-section"><button type="button" class="width-option" role="switch" aria-checked={wide} onclick={onwidthtoggle}><i class="fa fa-expand" aria-hidden="true"></i><span>{$_("trail-results-wide")}</span><span class="switch" class:active={wide} aria-hidden="true"></span></button></div>
                {/if}
                {#if onpagesize && pageSizes.length}<fieldset class="size-section"><legend>{$_("trail-results-per-page")}</legend>
                    <div class="size-options">{#each pageSizes as option}<button type="button" class:chosen={pageSize === option.value} aria-pressed={pageSize === option.value} onclick={() => { onpagesize?.(option.value); close(true); }}>{option.text}</button>{/each}</div>
                </fieldset>{/if}
            </div>
            {/if}
        </div>{/if}
        {#if actions}{@render actions()}{/if}
    </div>
</div>

<style>
    .result-controls { align-items: center; display: flex; gap: 1rem; justify-content: space-between; min-height: 2.5rem; min-width: 0; }
    .result-summary { align-items: center; display: flex; flex-wrap: wrap; gap: .75rem; min-width: 0; }
    .result-buttons { align-items: center; display: flex; flex-shrink: 0; gap: .5rem; }
    /* The search page provides the positioning context beside its input. */
    .buttons-only { display: contents; }
    .buttons-only .result-buttons { gap: .5rem; position: absolute; right: var(--trail-controls-inset, .6rem); top: var(--trail-controls-top, .65rem); z-index: 31; }
    .buttons-only .control-copy, .buttons-only .chevron { display: none; }
    .buttons-only .control-button { justify-content: center; padding: .5rem; width: 2.5rem; }
    .buttons-only .result-menu:not(.view-menu) { right: var(--result-sort-menu-offset, -3rem); }
    .menu-anchor { position: relative; }
    button { cursor: pointer; font: inherit; }
    .control-button { align-items: center; border: 1px solid transparent; border-radius: .5rem; display: inline-flex; font-size: .8rem; gap: .55rem; min-height: 2.5rem; padding: .5rem .7rem; }
    .control-button:hover, .control-button[aria-expanded="true"] { background: rgba(var(--input-background)); border-color: rgba(var(--input-border)); }
    .control-button > i:first-child { opacity: .65; }
    .chevron { font-size: .55rem; opacity: .5; }
    .result-menu { background: rgba(var(--menu-background)); border: 1px solid rgba(var(--input-border)); border-radius: .5rem; box-shadow: 0 8px 24px rgb(0 0 0 / .12); max-width: calc(100vw - 3rem); overflow-y: auto; padding: .55rem; position: absolute; right: 0; top: calc(100% + .45rem); width: 18rem; z-index: 25; }
    .result-menu.above { bottom: calc(100% + .45rem); top: auto; }
    .view-menu { width: 16rem; }
    fieldset { border: 0; margin: 0; min-width: 0; padding: 0; }
    legend { color: rgba(var(--content), .55); font-size: .7rem; font-weight: 500; padding: .35rem .55rem .4rem; }
    .sort-heading { align-items: center; display: flex; justify-content: space-between; padding: .1rem .2rem .35rem .55rem; width: 100%; }
    .sort-direction-button { align-items: center; border-radius: .45rem; color: rgba(var(--content)); display: inline-flex; font-size: .9rem; height: 2rem; justify-content: center; width: 2rem; }
    .sort-direction-button i { transition: transform .15s; }
    .sort-direction-button i.descending { transform: rotate(180deg); }
    .menu-option { align-items: center; border-radius: .45rem; cursor: pointer; display: flex; font-size: .8rem; gap: .65rem; min-height: 2.15rem; padding: .4rem .55rem; position: relative; }
    .menu-option input { cursor: pointer; height: 100%; inset: 0; margin: 0; opacity: 0; position: absolute; width: 100%; z-index: 1; }
    .menu-option > span { flex: 1; }
    .menu-option > i:first-of-type:not(.check) { opacity: .6; text-align: center; width: 1rem; }
    .menu-option:hover, .width-option:hover, .size-options button:hover, .sort-direction-button:hover { background: rgba(var(--secondary-hover)); }
    .menu-option:has(input:focus-visible), button:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: 1px; }
    .chosen { background: rgba(var(--input-border-focus), .08); }
    .check { font-size: .65rem; visibility: hidden; }
    .chosen .check { visibility: visible; }
    .size-section, .width-section, .layout-section { border-top: 1px solid rgba(var(--separator)); margin-top: .5rem; padding-top: .5rem; }
    .size-options { display: flex; gap: .4rem; padding: 0 .2rem .2rem; }
    .size-options button { border: 1px solid rgba(var(--input-border)); border-radius: .45rem; flex: 1; font-size: .75rem; min-height: 2rem; padding: .35rem; }
    .size-options button.chosen { border-color: rgba(var(--input-border-focus)); }
    .width-option { align-items: center; border-radius: .45rem; display: flex; font-size: .8rem; gap: .65rem; min-height: 2.5rem; padding: .4rem .55rem; width: 100%; }
    .width-option > span:first-of-type { flex: 1; text-align: left; }
    .switch { background: rgba(var(--content), .2); border-radius: 1rem; height: 1.05rem; padding: .15rem; width: 1.85rem; }
    .switch::after { background: rgba(var(--menu-background)); border-radius: 50%; content: ""; display: block; height: .75rem; width: .75rem; }
    .switch.active { background: rgba(var(--content)); }
    .switch.active::after { transform: translateX(.8rem); }
    @media (max-width: 639px) {
        .control-copy, .chevron { display: none; }
        .control-button { justify-content: center; padding: .5rem; width: 2.5rem; }
        .result-controls { gap: .5rem; }
        .result-buttons { gap: .15rem; }
        .result-menu:not(.view-menu) { right: var(--result-sort-menu-offset, -2.65rem); }
    }
</style>
