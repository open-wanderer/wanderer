<script lang="ts">
    import type { TrailSearchDraft } from "$lib/util/trail_search_draft";
    import { searchActors } from "$lib/stores/search_store";
    import { tags_index } from "$lib/stores/tag_store";
    import { currentUser } from "$lib/stores/user_store";
    import {
        findTrailSearchShortcut, trailSearchAuthorHandle,
        type TrailSearchShortcut, type TrailSearchShortcutChoice,
    } from "$lib/util/trail_search_shortcuts";
    import { _ } from "svelte-i18n";

    interface Props {
        id: string;
        filter: TrailSearchDraft;
        caret: number;
        selectionEnd: number;
        onselect: (choice: TrailSearchShortcutChoice, shortcut: TrailSearchShortcut) => void;
        onremove: (choice: TrailSearchShortcutChoice) => void;
        onfocusinput: () => void;
    }

    let { id, filter, caret, selectionEnd, onselect, onremove, onfocusinput }: Props = $props();
    let root = $state<HTMLDivElement>();
    let choices = $state<TrailSearchShortcutChoice[]>([]);
    let captured = $state<TrailSearchShortcutChoice[]>([]);
    let failed = $state(false);
    const shortcut = $derived(findTrailSearchShortcut(filter.q, caret, selectionEnd));
    const visible = $derived(Boolean(shortcut));
    const needsLogin = $derived(shortcut?.kind === "author" && !$currentUser);
    const caption = $derived($_(shortcut?.kind === "author" ? "author" : "tags"));
    const capturedFilters = $derived(captured.filter(choice => choice.kind === "author"
        ? filter.author === choice.author.id
        : filter.tags.includes(choice.tag.name)));

    $effect(() => {
        const current = shortcut;
        choices = [];
        failed = false;
        if (!current || !visible || needsLogin) return;
        const controller = new AbortController();
        const fetcher: typeof fetch = (url, options) => fetch(url, { ...options, signal: controller.signal });
        const timer = setTimeout(async () => {
            try {
                let results: TrailSearchShortcutChoice[];
                if (current.kind === "author") {
                    const authors = await searchActors(current.query, true, fetcher);
                    results = authors.map(author => ({ kind: "author", author }));
                } else {
                    const tags = await tags_index(current.query, fetcher);
                    results = tags.items.slice(0, 5).map(tag => ({ kind: "tag", tag }));
                }
                if (!controller.signal.aborted) choices = results;
            } catch {
                if (!controller.signal.aborted) failed = true;
            }
        }, 350);
        return () => {
            clearTimeout(timer);
            controller.abort();
        };
    });

    function identity(choice: TrailSearchShortcutChoice) {
        if (choice.kind === "author") return "author";
        return `tag:${choice.tag.name}`;
    }

    function label(choice: TrailSearchShortcutChoice) {
        if (choice.kind === "author") return trailSearchAuthorHandle(choice.author);
        return `#${choice.tag.name}`;
    }

    function select(choice: TrailSearchShortcutChoice) {
        if (!shortcut) return;
        captured = [...captured.filter(item => identity(item) !== identity(choice)), choice];
        onselect(choice, shortcut);
    }

    export function containsTarget(target: EventTarget | null) {
        return target instanceof Node && Boolean(root?.contains(target));
    }

    export function handleKeydown(event: KeyboardEvent, fromInput: boolean) {
        if (!visible || (!fromInput && !containsTarget(event.target))) return false;
        const forward = event.key === "ArrowDown" || (!fromInput && event.key === "ArrowRight");
        const backward = event.key === "ArrowUp" || (!fromInput && event.key === "ArrowLeft");
        if (!forward && !backward) return false;
        const buttons = [...(root?.querySelectorAll<HTMLButtonElement>(".shortcut-option") ?? [])];
        if (!buttons.length) return false;
        event.preventDefault();
        const current = buttons.findIndex(button => button === event.target);
        const next = current === -1 ? (forward ? 0 : buttons.length - 1)
            : (current + (forward ? 1 : -1) + buttons.length) % buttons.length;
        buttons[next].focus({ preventScroll: true });
        buttons[next].scrollIntoView({ block: "nearest" });
        return true;
    }
</script>

<div {id} bind:this={root}>
    {#if capturedFilters.length}
        <div class="captured-filters" aria-label={$_("trail-search-active")}>
            {#each capturedFilters as choice (identity(choice))}
                <button type="button" class="captured-filter" aria-label={`${$_("remove")}: ${label(choice)}`} onclick={() => { onremove(choice); onfocusinput(); }}>
                    <i class={`fa ${choice.kind === "author" ? "fa-user" : "fa-tag"}`} aria-hidden="true"></i>
                    <span>{label(choice)}</span><i class="fa fa-xmark" aria-hidden="true"></i>
                </button>
            {/each}
        </div>
    {/if}
    {#if visible && (choices.length || needsLogin || failed)}
        <section class="shortcut-suggestions" aria-label={caption}>
            <h3>{caption}</h3>
            {#if needsLogin}<p role="status">{$_("trail-search-shortcut-login")}</p>
            {:else if failed}<p role="status">{$_("trail-search-shortcut-error")}</p>
            {:else if choices.length}
                <ul>
                    {#each choices as choice}
                        <li><button type="button" class="shortcut-option" onclick={() => select(choice)}>
                            <i class={`fa ${choice.kind === "author" ? "fa-user" : "fa-tag"}`} aria-hidden="true"></i>
                            <span><strong>{choice.kind === "author" ? choice.author.username : `#${choice.tag.name}`}</strong>
                                {#if choice.kind === "author"}<small>{trailSearchAuthorHandle(choice.author)}</small>{/if}
                            </span>
                        </button></li>
                    {/each}
                </ul>
            {/if}
        </section>
    {/if}
</div>

<style>
    .captured-filters { display: flex; flex-wrap: wrap; gap: .45rem; margin-bottom: .9rem; }
    .captured-filter { align-items: center; background: rgba(var(--input-background)); border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; display: inline-flex; font-size: .8rem; gap: .5rem; max-width: 100%; padding: .4rem .7rem; }
    .captured-filter span { min-width: 0; overflow-wrap: anywhere; }
    .captured-filter i { flex-shrink: 0; opacity: .65; }
    .shortcut-suggestions { align-items: center; border-bottom: 1px solid rgba(var(--separator)); display: flex; flex-wrap: wrap; gap: .5rem .75rem; margin-bottom: 1rem; padding-bottom: .8rem; }
    .shortcut-suggestions h3 { color: rgba(var(--content), .65); flex-shrink: 0; font-size: .8rem; margin: 0; }
    .shortcut-suggestions p { color: rgba(var(--content), .65); font-size: .8rem; margin: .5rem 0; }
    .shortcut-suggestions ul { display: flex; flex-wrap: wrap; gap: .45rem; list-style: none; margin: 0; max-width: 100%; min-width: 0; padding: 0; }
    .shortcut-suggestions li { max-width: 100%; min-width: 0; }
    .shortcut-option { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; display: inline-flex; gap: .5rem; max-width: 100%; padding: .4rem .7rem; text-align: left; }
    .shortcut-option > i { flex-shrink: 0; opacity: .65; }
    .shortcut-option > span { align-items: baseline; display: flex; gap: .45rem; min-width: 0; }
    .shortcut-option strong, .shortcut-option small { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .shortcut-option strong { font-size: .85rem; font-weight: 500; }
    .shortcut-option small { color: rgba(var(--content), .65); font-size: .75rem; }
    .shortcut-option:hover, .shortcut-option:focus-visible, .captured-filter:hover { background: rgba(var(--secondary-hover)); }
    button:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: 1px; }
</style>
