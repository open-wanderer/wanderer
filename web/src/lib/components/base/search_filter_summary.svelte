<script module lang="ts">
    export type SearchFilterChip = {
        key: string;
        label: string;
        icon: string;
        text?: string;
    };
</script>

<script lang="ts">
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        chips: SearchFilterChip[];
        onremove: (key: string) => void;
    }

    let { chips, onremove }: Props = $props();
    let element: HTMLDivElement;
    let inline = $state(false);
    let position = $state("");

    onMount(() => {
        const root = element.closest<HTMLElement>("[data-search-root]");
        const heading = root?.querySelector<HTMLElement>(".search-heading");
        const input = heading?.querySelector<HTMLInputElement>("input");
        if (!root || !heading || !input) return;
        const originalPadding = input.style.paddingRight;
        let frame = 0;

        function measure() {
            frame = 0;
            const gap = parseFloat(getComputedStyle(element).columnGap) || 0;
            const buttons = Array.from(element.children) as HTMLElement[];
            const width = buttons.reduce((sum, button) => sum + button.getBoundingClientRect().width, 0)
                + Math.max(0, buttons.length - 1) * gap;
            const inputBounds = input!.getBoundingClientRect();
            // Leave enough room for the search text and its native clear button.
            const fits = width + 12 + 240 <= inputBounds.width;
            inline = fits;
            const padding = fits ? `${Math.ceil(width + 12)}px` : originalPadding;
            if (input!.style.paddingRight !== padding) input!.style.paddingRight = padding;
            if (fits) {
                const rootBounds = root!.getBoundingClientRect();
                const headingBounds = heading!.getBoundingClientRect();
                position = `right: ${rootBounds.right - inputBounds.right}px; top: ${headingBounds.top - rootBounds.top + headingBounds.height / 2}px`;
            } else {
                position = "";
            }
        }

        function schedule() {
            if (!frame) frame = requestAnimationFrame(measure);
        }

        const resize = new ResizeObserver(schedule);
        resize.observe(heading);
        resize.observe(input);
        resize.observe(element);
        const contents = new MutationObserver(schedule);
        contents.observe(element, { childList: true, subtree: true, characterData: true });
        document.fonts.addEventListener("loadingdone", schedule);
        measure();
        return () => {
            cancelAnimationFrame(frame);
            resize.disconnect();
            contents.disconnect();
            document.fonts.removeEventListener("loadingdone", schedule);
            input.style.paddingRight = originalPadding;
        };
    });
</script>

<div bind:this={element} class="applied-filters" class:inline style={position}
    role="group" aria-label={$_("trail-search-active")}>
    {#each chips as chip (chip.key)}
        <button type="button" class="applied-chip" data-filter-key={chip.key} title={chip.label}
            aria-label={`${$_("remove")}: ${chip.label}`} onclick={() => onremove(chip.key)}>
            <i class={`fa ${chip.icon}`} aria-hidden="true"></i>
            {#if (chip.text ?? chip.label).length}<span>{chip.text ?? chip.label}</span>{/if}
            <i class="fa fa-xmark" aria-hidden="true"></i>
        </button>
    {/each}
</div>

<style>
    .applied-filters {
        align-items: center;
        display: flex;
        flex-wrap: wrap;
        gap: .4rem;
        margin-top: .5rem;
        min-width: 0;
    }

    .applied-filters.inline {
        flex-wrap: nowrap;
        margin: 0;
        position: absolute;
        transform: translateY(-50%);
        width: max-content;
        z-index: 31;
    }

    .applied-chip {
        align-items: center;
        background: rgba(var(--input-background));
        border: 1px solid rgba(var(--input-border));
        border-radius: .5rem;
        cursor: pointer;
        display: inline-flex;
        flex-shrink: 0;
        font: inherit;
        font-size: .75rem;
        gap: .4rem;
        max-width: min(14rem, 100%);
        min-height: 1.8rem;
        padding: .25rem .45rem;
    }

    .applied-chip span {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .applied-chip i {
        flex-shrink: 0;
        opacity: .65;
    }

    .applied-chip i:last-child {
        font-size: .6rem;
    }

    .applied-chip:hover {
        background: rgba(var(--secondary-hover));
    }

    .applied-chip:focus-visible {
        outline: 2px solid rgba(var(--input-border-focus));
        outline-offset: 2px;
    }
</style>
