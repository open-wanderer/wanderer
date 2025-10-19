<script>
    import {dndzone} from 'svelte-dnd-action';
    import {flip} from 'svelte/animate';
    import TrailAnchorCard from './trail_anchor_card.svelte';
    import { onMount, onDestroy, tick } from 'svelte';
    
    export let itemsData;
    export let onDrop;
    export let idPropertyName = "id";
    export let flipDurationMs = 300;
    
    function handleConsider(e) {
        itemsData = e.detail.items;
    }
    function handleFinalize(e) {
        itemsData = e.detail.items;
        onDrop(e.detail.items);
    }

    function handleAnchorDelete(detail) {
        const { index } = detail;
        itemsData.splice(index, 1);
        itemsData = itemsData; // trigger reactivity
        onDrop(itemsData);
    }

    let sectionEl;
    let firstItemEl;
    let cardHeight = 0;
    let gapPx = 0;

    let ro;
    function parsePx(v) {
        if (!v) return 0;
        const n = Number(String(v).replace('px',''));
        return isNaN(n) ? 0 : n;
    }
    function measureAndApplyHeights() {
        if (!sectionEl) return;

        // If list is empty, clear inline sizing to let CSS class take over
        if (!itemsData || itemsData.length === 0) {
            sectionEl.style.minHeight = '';
            sectionEl.style.maxHeight = '';
            sectionEl.style.overflowY = '';
            return;
        }

        // read gap from computed style
        const cs = getComputedStyle(sectionEl);
        gapPx = parsePx(cs.rowGap || cs.gap);

        // find the first item wrapper div (direct child) and measure
        const firstChild = sectionEl.firstElementChild;
        firstItemEl = firstChild || null;
        const h = firstItemEl ? firstItemEl.getBoundingClientRect().height : 0;
        cardHeight = h;

        // Fallback: if CSS gap couldn't be parsed (e.g., returns 'normal' or an
        // unrecognized unit), measure the visual gap between first and second rows.
        if ((!gapPx || isNaN(gapPx)) && sectionEl.children.length >= 2 && firstItemEl) {
            const secondEl = sectionEl.children[1];
            const r1 = firstItemEl.getBoundingClientRect();
            const r2 = secondEl.getBoundingClientRect();
            const measuredGap = Math.max(0, Math.round(r2.top - r1.bottom));
            // Only use the measured gap if it's a sensible value (< card height*2)
            if (measuredGap >= 0 && measuredGap < Math.max(8, cardHeight * 2)) {
                gapPx = measuredGap;
            }
        }

        if (cardHeight > 0) {
            const minRows = 2;
            const maxRows = 6;
            const gapsForMin = Math.max(0, minRows - 1);
            const gapsForMax = Math.max(0, maxRows - 1);
            const verticalPadding = parsePx(cs.paddingTop) + parsePx(cs.paddingBottom);
            const minH = cardHeight * minRows + gapPx * gapsForMin + verticalPadding;
            const maxH = cardHeight * maxRows + gapPx * gapsForMax + verticalPadding;
            sectionEl.style.minHeight = `${Math.round(minH)}px`;
            sectionEl.style.maxHeight = `${Math.round(maxH)}px`;
            sectionEl.style.overflowY = 'auto';
        }
    }

    async function scheduleMeasure() {
        await tick();
        measureAndApplyHeights();
    }

    onMount(() => {
        // observe size changes of the container and first item
        ro = new ResizeObserver(() => measureAndApplyHeights());
        if (sectionEl) ro.observe(sectionEl);
        scheduleMeasure();
        window.addEventListener('resize', measureAndApplyHeights);
    });
    onDestroy(() => {
        try { ro && ro.disconnect(); } catch {}
        window.removeEventListener('resize', measureAndApplyHeights);
    });

    // Recompute when list length changes (DOM changes after tick)
    $: (itemsData?.length, scheduleMeasure());
</script>

<section
	use:dndzone={{ items: itemsData, flipDurationMs }}
    on:consider={handleConsider} 
    on:finalize={handleFinalize}
    class:empty={!itemsData || itemsData.length === 0}
    bind:this={sectionEl}
>
    {#each itemsData as item, i (item[idPropertyName])}
        <div animate:flip={{duration: flipDurationMs}}>
            <TrailAnchorCard anchor={item} index={i} isFirst={i == 0} isLast={i == itemsData.length - 1} onDelete={handleAnchorDelete}></TrailAnchorCard>
        </div>
    {/each}
</section>

<style>
    section {
        width: 100%;
        overflow-x: hidden;
        /* overflow-y is set dynamically; default to auto for safety */
        overflow-y: auto;
        /* small padding so hover-scaled cards aren't clipped */
        padding: 8px;
		padding-left: 4px;
        /* heights set dynamically in script to match 2–6 cards */
        gap: 0.5em;
        display: flex;
        flex-direction: column;
    }

    /* collapse completely when no items */
    section.empty {
        padding: 0;
        height: 0;
        min-height: 0;
        max-height: 0;
        overflow: hidden;
    }
</style>
