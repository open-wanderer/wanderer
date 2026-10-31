<script lang="ts">
    import type { Snippet } from "svelte";
    import type { Trail } from "#lib/models/trail.ts";
    import { _ } from "svelte-i18n";

    interface Props {
        trail: Trail;
        href: string;
        list?: boolean;
        selected: boolean;
        selectable?: boolean;
        onclick: (event: MouseEvent) => void;
        onselect: () => void;
        onhover?: (hovered: boolean) => void;
        children: Snippet<[boolean]>;
    }

    let { trail, href, list = false, selected, selectable = true,
        onclick, onselect, onhover, children }: Props = $props();
    let hovered = $state(false);
    let focused = $state(false);
</script>

<!-- svelte-ignore a11y_no_static_element_interactions (The container tracks focus and hover for its link and sibling checkbox.) -->
<div class="trail-result relative max-w-full flex-1" class:basis-full={list}
    onmouseenter={() => { hovered = true; onhover?.(true); }}
    onmouseleave={() => { hovered = false; onhover?.(false); }}
    onfocusin={() => focused = true}
    onfocusout={event => {
        if (document.hasFocus() && !(event.relatedTarget instanceof Node && event.currentTarget.contains(event.relatedTarget))) focused = false;
    }}>
    <a {href} {onclick}>{@render children(hovered)}</a>
    {#if selectable && (hovered || focused || selected)}
        <label class="selection" class:list>
            <input type="checkbox" checked={selected}
                aria-label={$_("trail-results-select-trail", { values: { name: trail.name } })}
                class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                onclick={onselect} />
        </label>
    {/if}
</div>

<style>
    a { display: block; }
    .selection { position: absolute; top: 1rem; left: 1rem; display: flex; align-items: center; justify-content: center; width: 2rem; height: 2rem; border-radius: 50%; background: rgba(var(--background)); color: rgba(var(--content)); }
    .selection, .selection input { cursor: pointer; }
    .selection.list { top: auto; left: auto; bottom: 1rem; right: 1rem; }
</style>
