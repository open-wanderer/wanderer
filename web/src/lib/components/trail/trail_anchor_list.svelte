<script lang="ts">
    import {dndzone} from 'svelte-dnd-action';
    import {flip} from 'svelte/animate';
    import TrailAnchorCard from './trail_anchor_card.svelte';
    import { onMount, onDestroy } from 'svelte';
    import type { ValhallaAnchor } from "$lib/models/valhalla";
    
    export let itemsData: ValhallaAnchor[] = [];
    export let onDrop: (items: ValhallaAnchor[]) => void;
    export let idPropertyName: keyof ValhallaAnchor = "id";
    export let flipDurationMs = 300;
    export let version = 0;

    let renderedItems: ValhallaAnchor[] = [];
    let renderedSignature: string | null = null;
    let previousPropSignature: string | null = null;
    let awaitingPropSync = false;
    let isDragging = false;
    let lastSyncedVersion = -1;

    const buildSignature = (items: ValhallaAnchor[]): string =>
        Array.isArray(items)
            ? items
                  .map((item) =>
                      item && idPropertyName in item
                          ? String(item[idPropertyName] ?? "")
                          : "",
                  )
                  .join("|")
            : "";

    const cloneItems = (items: ValhallaAnchor[]) =>
        Array.isArray(items) ? items.slice() : [];

    const getAnchorId = (anchor?: ValhallaAnchor) =>
        String(anchor?.[idPropertyName] ?? "");

    function setRenderedItems(
        nextItems: ValhallaAnchor[],
        { clone = false, skipMeasure = false }: { clone?: boolean; skipMeasure?: boolean } = {},
    ) {
        const resolved = Array.isArray(nextItems)
            ? clone
                ? cloneItems(nextItems)
                : nextItems
            : [];
        renderedItems = resolved;
        renderedSignature = buildSignature(renderedItems);
        if (!skipMeasure) {
            scheduleMeasure({ force: true });
        }
    }

    $: if (version !== lastSyncedVersion) {
        lastSyncedVersion = version;
        syncRenderedItemsFromProps(true);
    }
    function syncRenderedItemsFromProps(force = false) {
        const incomingItems = itemsData ?? [];
        const incomingSignature = buildSignature(incomingItems);

        if (!force && isDragging) {
            previousPropSignature = incomingSignature;
            return;
        }

        if (!force && awaitingPropSync) {
            if (incomingSignature === previousPropSignature) {
                return;
            }
            awaitingPropSync = false;
        }

        if (!force && incomingSignature === renderedSignature) {
            previousPropSignature = incomingSignature;
            return;
        }

        previousPropSignature = incomingSignature;
        setRenderedItems(incomingItems, { clone: true });
    }

    function handleConsider(e: CustomEvent<{ items: ValhallaAnchor[] }>) {
        isDragging = true;
        setRenderedItems(e.detail.items, { skipMeasure: true });
    }
    function handleFinalize(e: CustomEvent<{ items: ValhallaAnchor[] }>) {
        isDragging = false;
        awaitingPropSync = true;
        const nextItems = e.detail.items;
        setRenderedItems(nextItems);
        onDrop(nextItems);
    }

    function handleAnchorDelete(detail: { index: number; anchor: ValhallaAnchor }) {
        const { index } = detail;
        const nextItems = renderedItems.slice();
        nextItems.splice(index, 1);
        awaitingPropSync = true;
        setRenderedItems(nextItems);
        onDrop(nextItems);
    }

    let sectionEl: HTMLElement | null = null;
    let firstItemEl: HTMLElement | null = null;
    let cardHeight = 0;
    let gapPx = 0;

    let cachedSizing: {
        minHeight: number;
        maxHeight: number;
        gapPx: number;
        cardHeight: number;
        firstItemNode: HTMLElement | null;
        firstAnchorId: string | null;
    } | null = null;
    let needsFreshMeasurement = true;
    let observedFirstItem: Element | null = null;
    let lastMeasuredFirstAnchorId: string | null = null;
    let lastSectionWidth: number | null = null;

    let ro: ResizeObserver | null = null;
    
    let measurePromise: Promise<void> | null = null;
    let pendingForce = false;

    function parsePx(v: string | number | null | undefined) {
        if (!v) return 0;
        const n = Number(String(v).replace('px',''));
        return isNaN(n) ? 0 : n;
    }

    function clearSectionSizing() {
        cachedSizing = null;
        needsFreshMeasurement = true;
        lastMeasuredFirstAnchorId = null;

        if (!sectionEl) return;

        sectionEl.style.minHeight = "";
        sectionEl.style.maxHeight = "";
        sectionEl.style.overflowY = "";
    }

    function applySizingFromCache() {
        if (!cachedSizing || !sectionEl) return;

        sectionEl.style.minHeight = `${cachedSizing.minHeight}px`;
        sectionEl.style.maxHeight = `${cachedSizing.maxHeight}px`;
        sectionEl.style.overflowY = "auto";
    }

    function setObservedFirstItem(node: Element | null) {
        if (!ro || observedFirstItem === node) return;

        if (observedFirstItem) {
            try {
                ro.unobserve(observedFirstItem);
            } catch {
                // ignore
            }
        }

        observedFirstItem = node || null;
        if (observedFirstItem) {
            ro.observe(observedFirstItem);
        }
    }

    function handleSectionResize(entry: ResizeObserverEntry) {
        if (!sectionEl || entry.target !== sectionEl) return;

        const width = entry?.contentRect?.width;
        if (typeof width !== "number") return;

        const rounded = Math.round(width);
        if (lastSectionWidth === null) {
            lastSectionWidth = rounded;
            return;
        }

        if (Math.abs(rounded - lastSectionWidth) >= 1) {
            lastSectionWidth = rounded;
            needsFreshMeasurement = true;
            scheduleMeasure({ force: true });
        }
    }

    function handleFirstItemResize(entry: ResizeObserverEntry) {
        if (!observedFirstItem || entry.target !== observedFirstItem) return;

        const height = entry?.contentRect?.height;
        if (typeof height !== "number") {
            needsFreshMeasurement = true;
            scheduleMeasure({ force: true });
            return;
        }

        const roundedHeight = Math.round(height);
        if (cardHeight <= 0 || Math.abs(roundedHeight - Math.round(cardHeight)) >= 1) {
            needsFreshMeasurement = true;
            scheduleMeasure({ force: true });
        }
    }

    function computeSizing() {
        if (!sectionEl) return null;

        const cs = getComputedStyle(sectionEl);
        let computedGap = parsePx(cs.rowGap || cs.gap) || 0;
        const firstChild = sectionEl.firstElementChild as HTMLElement | null;
        firstItemEl = firstChild || null;
        const height = firstItemEl ? firstItemEl.getBoundingClientRect().height : 0;
        cardHeight = height;
        if ((!computedGap || isNaN(computedGap)) && sectionEl.children.length >= 2 && firstItemEl) {
            const secondEl = sectionEl.children[1] as HTMLElement;
            const r1 = firstItemEl.getBoundingClientRect();
            const r2 = secondEl.getBoundingClientRect();
            const measuredGap = Math.max(0, Math.round(r2.top - r1.bottom));
            if (measuredGap >= 0 && measuredGap < Math.max(8, cardHeight * 2)) {
                computedGap = measuredGap;
            }
        }

        if (cardHeight <= 0) return null;

        const minRows = 2;
        const maxRows = 6;
        const verticalPadding = parsePx(cs.paddingTop) + parsePx(cs.paddingBottom);
        const minHeight = cardHeight * minRows + computedGap * Math.max(0, minRows - 1) + verticalPadding;
        const maxHeight = cardHeight * maxRows + computedGap * Math.max(0, maxRows - 1) + verticalPadding;

        return {
            minHeight: Math.round(minHeight),
            maxHeight: Math.round(maxHeight),
            gapPx: computedGap,
            cardHeight,
            firstItemNode: firstItemEl,
            firstAnchorId: getAnchorId(renderedItems?.[0]) || null,
        };
    }

    function measureAndApplyHeights({ force = false } = {}) {
        if (!sectionEl) return;

        if (!renderedItems || renderedItems.length === 0) {
            clearSectionSizing();
            setObservedFirstItem(null);
            return;
        }

        if (force || needsFreshMeasurement || !cachedSizing) {
            const sizing = computeSizing();
            if (!sizing) return;

            cachedSizing = sizing;
            gapPx = sizing.gapPx;
            cardHeight = sizing.cardHeight;
            needsFreshMeasurement = false;
            lastMeasuredFirstAnchorId = sizing.firstAnchorId;
            setObservedFirstItem(sizing.firstItemNode);
        }

        applySizingFromCache();
    }

    async function scheduleMeasure({ force = false } = {}) {
        const hasItemsBeforeTick = Array.isArray(renderedItems) && renderedItems.length > 0;

        if (hasItemsBeforeTick) {
            const currentFirstId = getAnchorId(renderedItems[0]) || null;
            if (currentFirstId !== lastMeasuredFirstAnchorId) {
                needsFreshMeasurement = true;
            }
        }

        if (!force && hasItemsBeforeTick && !needsFreshMeasurement && !measurePromise) {
            return;
        }

        pendingForce = pendingForce || force;
        if (measurePromise) {
            return measurePromise;
        }

        measurePromise = (async () => {
            //await tick();
            const hasItemsAfterTick = Array.isArray(renderedItems) && renderedItems.length > 0;
            if (hasItemsAfterTick) {
                const currentFirstId = getAnchorId(renderedItems[0]) || null;
                if (currentFirstId !== lastMeasuredFirstAnchorId) {
                    needsFreshMeasurement = true;
                }
            }
            measureAndApplyHeights({ force: pendingForce });
            pendingForce = false;
        })().finally(() => {
            measurePromise = null;
        });

        return measurePromise;
    }

    function handleWindowResize() {
        needsFreshMeasurement = true;
        scheduleMeasure({ force: true });
    }

    onMount(() => {
        ro = new ResizeObserver((entries) => {
            for (const entry of entries) {
                if (sectionEl && entry.target === sectionEl) {
                    handleSectionResize(entry);
                } else {
                    handleFirstItemResize(entry);
                }
            }
        });

        if (sectionEl) ro.observe(sectionEl);
        if (firstItemEl) setObservedFirstItem(firstItemEl);
        
        scheduleMeasure({ force: true });
        window.addEventListener("resize", handleWindowResize);
    });

    onDestroy(() => {
        try {
            ro && ro.disconnect();
        } catch {
            // ignore
        }
        
        observedFirstItem = null;
        lastSectionWidth = null;
        window.removeEventListener("resize", handleWindowResize);
    });

    //$: (itemsData?.length, itemsData?.[0]?.[idPropertyName], scheduleMeasure());
</script>

<section
	use:dndzone={{ items: renderedItems, flipDurationMs }}
    on:consider={handleConsider} 
    on:finalize={handleFinalize}
    class:empty={!renderedItems || renderedItems.length === 0}
    bind:this={sectionEl}
>
    {#each renderedItems as item, i (getAnchorId(item) || i)}
        <div animate:flip={{duration: flipDurationMs}}>
            <TrailAnchorCard anchor={item} index={i} isFirst={i == 0} isLast={i == renderedItems.length - 1} onDelete={handleAnchorDelete}></TrailAnchorCard>
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
