<script lang="ts">
    import { searchLists, searchLocations, type ListSearchResult, type LocationSearchResult } from "$lib/stores/search_store";
    import { findTrailSearchShortcut } from "$lib/util/trail_search_shortcuts";
    import { _ } from "svelte-i18n";

    interface Props {
        id: string;
        query: string;
        caret: number;
        selectionEnd: number;
        onlocationselect: (location: LocationSearchResult) => void;
    }

    let { id, query, caret, selectionEnd, onlocationselect }: Props = $props();
    let root = $state<HTMLDivElement>();
    let places = $state<LocationSearchResult[]>([]);
    let lists = $state<ListSearchResult[]>([]);
    const text = $derived(query.trim());
    const eligible = $derived(text.length >= 2
        && !findTrailSearchShortcut(query, caret, selectionEnd));

    $effect(() => {
        const current = text;
        places = [];
        lists = [];
        if (!eligible) return;
        const controller = new AbortController();
        const fetcher: typeof fetch = (url, options) => fetch(url, { ...options, signal: controller.signal });
        const timer = setTimeout(() => {
            void Promise.allSettled([
                searchLocations(current, 3, fetcher).then(results => {
                    if (!controller.signal.aborted) places = results.slice(0, 3);
                }),
                searchLists(current, {
                    limit: 3,
                    attributesToSearchOn: ["name"],
                    attributesToRetrieve: ["id", "name", "author_name", "domain"],
                }, fetcher).then(results => {
                    if (!controller.signal.aborted) lists = results.slice(0, 3);
                }),
            ]);
        }, 350);
        return () => {
            clearTimeout(timer);
            controller.abort();
        };
    });

    function listHandle(list: ListSearchResult) {
        return `@${list.author_name}${list.domain ? `@${list.domain}` : ""}`;
    }

    export function handleKeydown(event: KeyboardEvent, fromInput: boolean) {
        if (!eligible || (!fromInput && !(event.target instanceof Node && root?.contains(event.target)))) return false;
        const forward = event.key === "ArrowDown" || (!fromInput && event.key === "ArrowRight");
        const backward = event.key === "ArrowUp" || (!fromInput && event.key === "ArrowLeft");
        if (!forward && !backward) return false;
        const options = [...(root?.querySelectorAll<HTMLElement>(".direct-option") ?? [])];
        if (!options.length) return false;
        event.preventDefault();
        const current = options.findIndex(option => option === event.target);
        const next = current === -1 ? (forward ? 0 : options.length - 1)
            : (current + (forward ? 1 : -1) + options.length) % options.length;
        options[next].focus({ preventScroll: true });
        options[next].scrollIntoView({ block: "nearest" });
        return true;
    }
</script>

<div {id} bind:this={root}>
    {#if eligible && (places.length || lists.length)}
        <div class="direct-suggestions">
            {#if places.length}
                <section class="suggestion-group" data-kind="places" aria-labelledby={`${id}-places`}>
                    <h3 id={`${id}-places`}>{$_("trail-search-suggestions-places")}</h3>
                    <ul>
                        {#each places as location}
                            <li>
                                <button type="button" class="direct-option" onclick={() => onlocationselect(location)} title={`${location.name} · ${location.description}`}>
                                    <i class="fa fa-location-dot" aria-hidden="true"></i>
                                    <span><strong>{location.name}</strong><small>{location.description}</small></span>
                                </button>
                            </li>
                        {/each}
                    </ul>
                </section>
            {/if}
            {#if lists.length}
                <section class="suggestion-group" data-kind="lists" aria-labelledby={`${id}-lists`}>
                    <h3 id={`${id}-lists`}>{$_("trail-search-suggestions-lists")}</h3>
                    <ul>
                        {#each lists as list (list.id)}
                            <li>
                                <a class="direct-option" href={`/lists/${encodeURIComponent(listHandle(list))}/${encodeURIComponent(list.id)}`}
                                    title={`${list.name} · ${listHandle(list)}`}>
                                    <i class="fa fa-list-ul" aria-hidden="true"></i>
                                    <span><strong>{list.name}</strong><small>{listHandle(list)}</small></span>
                                </a>
                            </li>
                        {/each}
                    </ul>
                </section>
            {/if}
        </div>
    {/if}
</div>

<style>
    .direct-suggestions { border-bottom: 1px solid rgba(var(--separator)); display: grid; gap: .75rem; margin-bottom: 1rem; padding-bottom: .8rem; }
    .suggestion-group { align-items: center; display: flex; flex-wrap: wrap; gap: .5rem .75rem; min-width: 0; }
    h3 { color: rgba(var(--content), .65); flex-shrink: 0; font-size: .8rem; margin: 0; }
    ul { display: flex; flex-wrap: wrap; gap: .45rem; list-style: none; margin: 0; max-width: 100%; min-width: 0; padding: 0; }
    li { max-width: 100%; min-width: 0; }
    .direct-option { align-items: center; border: 1px solid rgba(var(--input-border)); border-radius: .5rem; color: inherit; cursor: pointer; display: inline-flex; font: inherit; gap: .5rem; max-width: 100%; padding: .4rem .7rem; text-align: left; text-decoration: none; }
    .direct-option > i { flex-shrink: 0; opacity: .65; }
    .direct-option > span { align-items: baseline; display: flex; gap: .45rem; min-width: 0; }
    strong, small { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    strong { flex-shrink: 0; font-size: .85rem; font-weight: 500; max-width: 65%; }
    small { color: rgba(var(--content), .65); font-size: .75rem; }
    .direct-option:hover, .direct-option:focus-visible { background: rgba(var(--secondary-hover)); }
    .direct-option:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: 1px; }
</style>
