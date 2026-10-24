<script lang="ts">
    import SearchFilterSummary, { type SearchFilterChip } from "../base/search_filter_summary.svelte";
    import ActorSearch from "$lib/components/actor_search.svelte";
    import type { SearchItem } from "$lib/components/base/search.svelte";
    import TrailResultControls from "$lib/components/trail/trail_result_controls.svelte";
    import type { ListFilter } from "$lib/models/list";
    import { deepEqual } from "$lib/util/deep_util";
    import { fromDraft, toDraft, type ListSearchDraft } from "$lib/util/list_search_draft";
    import { loadSearchLayout, searchLayout, updateSearchLayout } from "$lib/stores/search_layout_store";
    import { currentUser } from "$lib/stores/user_store";
    import { onMount, untrack } from "svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        filter: ListFilter;
        wide?: boolean;
        onwidthtoggle?: () => void;
        onapply: () => void;
    }
    let { filter = $bindable(), wide = true, onwidthtoggle, onapply }: Props = $props();
    const id = $props.id();
    const defaults = { q: "", author: "", public: true, shared: true };
    let draft = $state<ListSearchDraft>(toDraft(filter));
    let expanded = $state(false);
    let activeEditor = $state<"author" | "visibility">("author");
    let authorQuery = $state("");
    let authorNames = $state<Record<string, string>>({});
    let panelMaxHeight = $state(0);
    let root: HTMLDivElement;
    let heading: HTMLDivElement;
    let input: HTMLInputElement;
    let restoringFocus = false;
    let updateTimer: ReturnType<typeof setTimeout> | undefined;
    let composing = $state(false);
    let resetEnabled = $derived(Boolean(expanded || (expanded ? draft : filter).q || summary(expanded ? draft : filter).length));
    let chips = $derived(summary(filter));
    let sortOptions = $derived([
        { text: $_("alphabetical"), value: "name" },
        { text: $_("creation-date"), value: "created" },
    ]);

    $effect(() => {
        if (!expanded || composing) return;
        const next = toDraft(draft);
        if (untrack(() => deepEqual(next, toDraft(filter)))) return;
        const timer = setTimeout(() => publish(next), 350);
        updateTimer = timer;
        return () => clearTimeout(timer);
    });

    function open() {
        if (expanded) return;
        draft = toDraft(filter);
        authorQuery = filter.author ? authorNames[filter.author] ?? filter.author : "";
        expanded = true;
        updatePanelHeight();
    }
    function dismiss(restoreFocus = false) {
        if (!expanded) return;
        publish();
        expanded = false;
        if (restoreFocus) {
            restoringFocus = true;
            input.focus();
            restoringFocus = false;
        }
    }
    function updateQuery(event: Event) {
        const q = (event.currentTarget as HTMLInputElement).value;
        open();
        draft.q = q;
    }
    function publish(next = toDraft(draft)) {
        clearTimeout(updateTimer);
        const applied = fromDraft(next, filter);
        if (deepEqual(applied, filter)) return;
        filter = applied;
        onapply();
    }
    function apply(event?: SubmitEvent) {
        event?.preventDefault();
        if (!composing) dismiss(true);
    }

    function selectAuthor(item: SearchItem) {
        draft.author = item.value.id;
        authorNames[item.value.id] = item.value.preferred_username ?? item.text;
    }
    function reset() {
        draft = { ...toDraft(expanded ? draft : filter), ...defaults };
        authorQuery = "";
        publish();
    }
    function summary(value: ListSearchDraft) {
        const result: SearchFilterChip[] = [];
        if ($currentUser && value.author) result.push({ key: "author", label: `${$_("author")}: ${authorNames[value.author] ?? value.author}`, text: authorNames[value.author] ?? value.author, icon: "fa-user" });
        if ($currentUser && (value.public !== defaults.public || value.shared !== defaults.shared)) {
            const visibility = [value.public && $_("public"), value.shared && $_("shared")].filter(Boolean).join(", ") || "–";
            result.push({ key: "visibility", label: `${$_("visibilty-status")}: ${visibility}`, text: visibility, icon: "fa-eye" });
        }
        return result;
    }
    function removeChip(key: string) {
        const next = toDraft(expanded ? draft : filter);
        if (key === "author") next.author = "";
        else { next.public = defaults.public; next.shared = defaults.shared; }
        draft = next;
        publish();
    }
    function sort(sort: string, order: "+" | "-") {
        if (sort !== "name" && sort !== "created") return;
        filter = { ...filter, sort, sortOrder: order };
        onapply();
    }
    function updatePanelHeight() {
        if (!expanded || !heading) return;
        const viewport = window.visualViewport;
        const top = viewport?.offsetTop ?? 0;
        const bottom = top + (viewport?.height ?? window.innerHeight);
        let available = bottom - heading.getBoundingClientRect().bottom - 16;
        const room = heading.getBoundingClientRect().top - top - 12;
        if (available < 240 && room > 0) {
            window.scrollBy({ top: Math.min(240 - available, room), behavior: "instant" });
            available = bottom - heading.getBoundingClientRect().bottom - 16;
        }
        panelMaxHeight = Math.max(0, available);
    }
    function outside(event: PointerEvent) {
        if (expanded && event.target instanceof Node && !root.contains(event.target)) dismiss();
    }
    function focusOut(event: FocusEvent) {
        if (expanded && event.relatedTarget instanceof Node && !root.contains(event.relatedTarget)) dismiss();
    }
    function keydown(event: KeyboardEvent) {
        if (event.isComposing || composing) return;
        if (!(event.target instanceof Node) || !root?.contains(event.target)) return;
        if (event.key === "Escape" && expanded) {
            event.preventDefault(); event.stopPropagation(); dismiss(true);
        } else if (event.key === "ArrowDown" && event.target === input && !expanded) {
            event.preventDefault(); open();
        }
    }
    onMount(() => {
        loadSearchLayout();
        const observer = new ResizeObserver(updatePanelHeight);
        observer.observe(heading);
        observer.observe(root);
        window.visualViewport?.addEventListener("resize", updatePanelHeight);
        return () => {
            observer.disconnect();
            window.visualViewport?.removeEventListener("resize", updatePanelHeight);
        };
    });
</script>

<svelte:window onpointerdown={outside} onkeydown={keydown} onresize={updatePanelHeight} onscroll={updatePanelHeight} />

<div class="list-search-bar" data-search-root class:in-flow={$searchLayout.expansion === "push"} bind:this={root} onfocusout={focusOut}>
    <form class="search-form" class:expanded role="search" aria-label={$_("search-list")} onsubmit={apply}>
        <div class="search-heading" bind:this={heading}>
            <i class="fa fa-magnifying-glass" aria-hidden="true"></i>
            <input bind:this={input} type="search" autocomplete="off" value={expanded ? draft.q : filter.q}
                aria-label={$_("search-list")} placeholder={$_("list-search-placeholder")} aria-controls={`${id}-panel`}
                onfocus={() => { if (!restoringFocus) open(); }} onclick={open} oninput={updateQuery}
                oncompositionstart={() => composing = true}
                oncompositionend={(event) => { composing = false; updateQuery(event); }} />
            <button type="button" class="icon-button" disabled={!resetEnabled} aria-label={$_("trail-search-reset")} title={$_("trail-search-reset")} onclick={reset}>
                <i class="fa fa-arrow-rotate-left" aria-hidden="true"></i>
            </button>
            <button type="button" class="icon-button" aria-label={$_(expanded ? "trail-search-close" : "trail-search-filter-by")}
                aria-expanded={expanded} aria-controls={`${id}-panel`} onclick={() => expanded ? dismiss(true) : open()}>
                <i class={`fa ${expanded ? "fa-xmark" : "fa-sliders"}`} aria-hidden="true"></i>
            </button>
        </div>
        {#if expanded}
            <div id={`${id}-panel`} class="search-panel" role="region" aria-label={$_("trail-search-filter-by")} style={`max-height: ${panelMaxHeight}px`}>
                <div class="filter-body">
                    {#if $currentUser}
                        <p class="filter-caption">{$_("trail-search-filter-by")}</p>
                        <div class="filter-chips" role="group" aria-label={$_("trail-search-filter-by")}>
                            <button type="button" class="filter-chip" class:editing={activeEditor === "author"} class:selected={Boolean(draft.author)}
                                aria-pressed={activeEditor === "author"} aria-controls={`${id}-editor`} onclick={() => activeEditor = "author"}>
                                <i class="fa fa-user" aria-hidden="true"></i><span>{$_("author")}</span>
                                {#if draft.author}<i class="fa fa-check" aria-hidden="true"></i>{/if}
                            </button>
                            <button type="button" class="filter-chip" class:editing={activeEditor === "visibility"} class:selected={draft.public !== defaults.public || draft.shared !== defaults.shared}
                                aria-pressed={activeEditor === "visibility"} aria-controls={`${id}-editor`} onclick={() => activeEditor = "visibility"}>
                                <i class="fa fa-eye" aria-hidden="true"></i><span>{$_("visibilty-status")}</span>
                                {#if draft.public !== defaults.public || draft.shared !== defaults.shared}<i class="fa fa-check" aria-hidden="true"></i>{/if}
                            </button>
                        </div>
                        <section id={`${id}-editor`} class="editor" class:author-editor={activeEditor === "author"} class:author-suggestions={activeEditor === "author" && Boolean(authorQuery.trim()) && authorQuery !== authorNames[draft.author ?? ""]} aria-label={$_(activeEditor === "author" ? "author" : "visibilty-status")}>
                            {#if activeEditor === "author"}
                                <ActorSearch bind:value={authorQuery} label={$_("author")} placeholder={$_("trail-search-author-placeholder")} clearAfterSelect={false}
                                    onclick={selectAuthor} onclear={() => draft.author = ""} />
                            {:else}
                                <div class="visibility-options">
                                    <label><input type="checkbox" bind:checked={draft.public} /><span>{$_("public")}</span></label>
                                    <label><input type="checkbox" bind:checked={draft.shared} /><span>{$_("shared")}</span></label>
                                </div>
                            {/if}
                        </section>
                    {/if}
                </div>

            </div>
        {/if}
    </form>
    <TrailResultControls buttonsOnly sort={filter.sort ?? "created"} order={filter.sortOrder} {sortOptions} onsort={sort}
        {wide} {onwidthtoggle} searchExpansion={$searchLayout.expansion}
        onsearchexpansion={expansion => updateSearchLayout({ expansion })} />
    {#if chips.length}<SearchFilterSummary {chips} onremove={removeChip} />{/if}
</div>

<style>
    .list-search-bar { --trail-controls-inset: .6rem; --trail-controls-top: .3125rem; min-width: 0; position: relative; width: 100%; }
    button { cursor: pointer; }
    button, input { font: inherit; }
    button:focus-visible, input:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: 3px; }
    .search-form { background: rgba(var(--input-background)); border-radius: .5rem; position: relative; }
    .search-form:hover .search-heading, .search-form:focus-within .search-heading, .search-form.expanded .search-heading { border-color: rgba(var(--input-border-focus)); }
    .search-form.expanded { background: rgba(var(--menu-background)); border-radius: .5rem .5rem 0 0; z-index: 30; }
    .search-form.expanded .search-heading { border-bottom-color: transparent; }
    .search-heading { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: inherit; display: flex; gap: 1rem; min-height: 3.125rem; padding: .25rem calc(5.5rem + var(--trail-controls-inset, .6rem) + .5rem) .25rem 1.2rem; }
    .search-heading > i { color: rgba(var(--content), .65); font-size: 1.15rem; }
    .search-heading input { background: transparent; border: 0; flex: 1; font-size: 1rem; min-width: 0; outline: 0; padding: .35rem 0; text-overflow: ellipsis; }
    .search-heading input::placeholder { color: rgba(var(--content), .48); }
    .search-heading input:focus-visible { outline: none; }
    .icon-button { align-items: center; border-radius: 50%; display: flex; flex-shrink: 0; height: 2rem; justify-content: center; width: 2rem; }
    .icon-button:hover { background: rgba(var(--secondary-hover)); }
    .icon-button:disabled { cursor: default; opacity: .3; }
    .icon-button:disabled:hover { background: transparent; }
    .in-flow .search-panel { box-shadow: none; margin-bottom: -1px; position: relative; top: -1px; }
    .search-panel { background: rgba(var(--menu-background)); border: 1px solid rgba(var(--input-border-focus)); border-top: 0; border-radius: 0 0 .5rem .5rem; box-shadow: 0 14px 24px rgb(0 0 0 / .12); display: flex; flex-direction: column; left: 0; overflow: hidden; position: absolute; right: 0; top: calc(100% - 1px); }
    .filter-body { min-height: 0; min-width: 0; overflow-y: auto; padding: 1.3rem 1.5rem 1.6rem; }
    .filter-caption { color: rgba(var(--content), .65); font-size: .8rem; font-weight: 600; margin: 0 0 .8rem; }
    .filter-chips { display: flex; flex-wrap: wrap; gap: .55rem; }
    .filter-chip { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: .5rem; display: inline-flex; font-size: .85rem; font-weight: 500; gap: .55rem; max-width: 100%; min-height: 2.55rem; overflow-wrap: anywhere; padding: .55rem .95rem; }
    .filter-chip > i { color: rgba(var(--content), .6); }
    .filter-chip.editing, .filter-chip.selected { border-color: rgba(var(--input-border-focus)); }
    .filter-chip.editing { box-shadow: inset 0 0 0 1px rgba(var(--input-border-focus)); }
    .editor { margin-top: 1.25rem; max-width: 30rem; }
    .editor.author-editor :global(label) { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; border: 0; }
    .editor.author-suggestions { min-height: 21rem; }
    .visibility-options { display: flex; flex-wrap: wrap; gap: 1rem 2rem; }
    .visibility-options label { align-items: center; cursor: pointer; display: flex; gap: .6rem; }
    .visibility-options input { accent-color: rgba(var(--primary)); height: 1rem; width: 1rem; }
    @media (max-width: 767px) {
        .search-form { border-radius: .5rem; }
        .search-form.expanded { border-radius: .5rem .5rem 0 0; }
        .search-panel { border-radius: 0 0 .5rem .5rem; }
        .search-heading { gap: .75rem; padding-left: 1rem; }
    }
    @media (max-width: 639px) {
        .list-search-bar { --trail-controls-inset: .4rem; }
        .search-heading { gap: .5rem; padding-left: .65rem; }
        .search-heading > i { display: none; }
        .filter-body { padding: 1rem; }
    }
</style>
