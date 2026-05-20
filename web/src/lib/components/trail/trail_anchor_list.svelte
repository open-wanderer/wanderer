<script lang="ts">
    import { tick } from "svelte";
    import { _ } from "svelte-i18n";
    import GpxMetricsComputation from "$lib/models/gpx/gpx-metrics-computation";
    import type TrackSegment from "$lib/models/gpx/track-segment";
    import type { ValhallaAnchor } from "$lib/models/valhalla";
    import { searchLocationReverse } from "$lib/stores/search_store";
    import {
        formatDistance,
        formatElevation,
    } from "$lib/util/format_util";

    interface Props {
        anchors: ValhallaAnchor[];
        segments?: TrackSegment[];
        disabled?: boolean;
        onMove: (fromIndex: number, toIndex: number) => void | Promise<void>;
        onDelete: (index: number) => void;
    }

    let {
        anchors,
        segments = [],
        disabled = false,
        onMove,
        onDelete,
    }: Props = $props();

    const anchorCoordinates = (anchor: ValhallaAnchor) =>
        `${anchor.lat.toFixed(4)}, ${anchor.lon.toFixed(4)}`;

    function fallbackAnchorTitle(index: number) {
        if (index === 0) {
            return $_("start");
        }
        if (index === anchors.length - 1) {
            return $_("finish");
        }
        return `${$_("route-point")} #${index + 1}`;
    }

    const locationCache = new Map<string, string>();
    const pendingLocationRequests = new Set<string>();
    let locationNames = $state<Record<string, string>>({});
    let locationAbortController: AbortController | null = null;

    function locationCacheKey(anchor: ValhallaAnchor) {
        return `${anchor.lat.toFixed(5)},${anchor.lon.toFixed(5)}`;
    }

    async function loadAnchorLocation(anchor: ValhallaAnchor, signal: AbortSignal) {
        const key = locationCacheKey(anchor);
        if (locationNames[key] || pendingLocationRequests.has(key)) {
            return;
        }

        const cached = locationCache.get(key);
        if (cached) {
            locationNames = { ...locationNames, [key]: cached };
            return;
        }

        pendingLocationRequests.add(key);
        try {
            const locationName = await searchLocationReverse(anchor.lat, anchor.lon, {
                includeRoad: true,
                signal,
            });
            if (locationName) {
                locationCache.set(key, locationName);
                locationNames = { ...locationNames, [key]: locationName };
            }
        } catch (error) {
            if (!(error instanceof DOMException && error.name === "AbortError")) {
                console.error("Failed to resolve anchor location", error);
            }
        } finally {
            pendingLocationRequests.delete(key);
        }
    }

    async function loadAnchorLocations(nextAnchors: ValhallaAnchor[]) {
        locationAbortController?.abort();
        const controller = new AbortController();
        locationAbortController = controller;

        for (const anchor of nextAnchors) {
            if (controller.signal.aborted) return;
            await loadAnchorLocation(anchor, controller.signal);
        }
    }

    $effect(() => {
        void loadAnchorLocations(anchors);
        return () => locationAbortController?.abort();
    });

    function anchorTitle(anchor: ValhallaAnchor, index: number) {
        return locationNames[locationCacheKey(anchor)] || fallbackAnchorTitle(index);
    }

    function handleItemMouseEnter(e: MouseEvent) {
        const element = e.currentTarget as HTMLElement;
        const title = element.querySelector<HTMLElement>(".anchor-title-viewport");
        const text = element.querySelector<HTMLElement>(".anchor-title-text");
        if (!title || !text) {
            return;
        }

        text.style.maxWidth = "none";
        text.style.overflow = "visible";
        text.style.textOverflow = "clip";

        const overflow = Math.max(0, text.scrollWidth - title.clientWidth);

        text.style.removeProperty("max-width");
        text.style.removeProperty("overflow");
        text.style.removeProperty("text-overflow");

        if (overflow <= 0) {
            title.classList.remove("is-overflowing");
            title.style.removeProperty("--title-scroll-distance");
            title.style.removeProperty("--title-scroll-duration");
            return;
        }

        const duration = Math.min(7, Math.max(1.8, overflow / 24));

        title.style.setProperty("--title-scroll-distance", `-${overflow}px`);
        title.style.setProperty("--title-scroll-duration", `${duration}s`);
        title.classList.add("is-overflowing");
    }

    function handleItemMouseLeave(e: MouseEvent) {
        const title = (e.currentTarget as HTMLElement).querySelector<HTMLElement>(
            ".anchor-title-viewport",
        );
        if (!title) {
            return;
        }

        title.classList.remove("is-overflowing");
        title.style.removeProperty("--title-scroll-distance");
        title.style.removeProperty("--title-scroll-duration");
    }

    function anchorIcon(index: number) {
        if (index === 0) {
            return "fa-bullseye";
        }
        if (index === anchors.length - 1) {
            return "fa-flag-checkered";
        }
        return "fa-location-dot";
    }

    interface SegmentMetrics {
        distance: number;
        elevationGain: number;
        elevationLoss: number;
    }

    function computeSegmentMetrics(segment: TrackSegment): SegmentMetrics {
        const metrics = new GpxMetricsComputation(5, 5);
        for (const point of segment.trkpt ?? []) {
            metrics.addAndFilter(point);
        }

        return {
            distance: metrics.totalDistance,
            elevationGain: metrics.totalElevationGainSmoothed,
            elevationLoss: metrics.totalElevationLossSmoothed,
        };
    }

    const allSegmentMetrics = $derived(segments.map(computeSegmentMetrics));

    const allCumulativeMetrics = $derived(
        allSegmentMetrics.reduce<SegmentMetrics[]>((totals, metrics) => {
            const previous = totals[totals.length - 1] ?? {
                distance: 0,
                elevationGain: 0,
                elevationLoss: 0,
            };
            totals.push({
                distance: previous.distance + metrics.distance,
                elevationGain: previous.elevationGain + metrics.elevationGain,
                elevationLoss: previous.elevationLoss + metrics.elevationLoss,
            });
            return totals;
        }, []),
    );

    function segmentMetrics(index: number) {
        if (index === 0) return null;
        return allSegmentMetrics[index - 1] ?? null;
    }

    function cumulativeMetrics(index: number) {
        if (index === 0) return null;
        return allCumulativeMetrics[index - 1] ?? null;
    }

    let listElement: HTMLOListElement;
    let dragIndex = $state<number | null>(null);
    let insertBefore = $state<number | null>(null);
    let pointerId: number | null = null;

    function isValidInsert(pos: number): boolean {
        return (
            dragIndex !== null &&
            insertBefore === pos &&
            pos !== dragIndex &&
            pos !== dragIndex + 1
        );
    }

    function getInsertPosition(clientY: number): number {
        const items = Array.from(
            listElement.querySelectorAll<HTMLElement>("li[data-anchor-index]"),
        );

        for (const item of items) {
            const itemIndex = Number(item.dataset.anchorIndex);
            const rect = item.getBoundingClientRect();
            if (clientY < rect.top + rect.height / 2) {
                return itemIndex;
            }
        }

        return anchors.length;
    }

    function clearDragState() {
        dragIndex = null;
        insertBefore = null;
        pointerId = null;
    }

    async function handleKeyDown(e: KeyboardEvent, index: number) {
        if (disabled) return;
        let toIndex: number | null = null;
        if (e.key === "ArrowUp" && index > 0) {
            e.preventDefault();
            toIndex = index - 1;
        } else if (e.key === "ArrowDown" && index < anchors.length - 1) {
            e.preventDefault();
            toIndex = index + 1;
        }
        if (toIndex === null) return;
        await onMove(index, toIndex);
        await tick();
        const handles = listElement.querySelectorAll<HTMLElement>("li[data-anchor-index] button.drag-handle");
        handles[toIndex]?.focus();
    }

    function handlePointerDown(e: PointerEvent, index: number) {
        if (disabled || e.button !== 0) {
            return;
        }

        e.preventDefault();
        dragIndex = index;
        insertBefore = index;
        pointerId = e.pointerId;
        (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    }

    function handlePointerMove(e: PointerEvent) {
        if (pointerId !== e.pointerId || dragIndex === null) {
            return;
        }

        e.preventDefault();
        insertBefore = getInsertPosition(e.clientY);
    }

    function handlePointerUp(e: PointerEvent) {
        if (pointerId !== e.pointerId) {
            return;
        }

        e.preventDefault();
        if (dragIndex !== null && insertBefore !== null) {
            const toIndex =
                dragIndex < insertBefore ? insertBefore - 1 : insertBefore;
            if (toIndex !== dragIndex) {
                onMove(dragIndex, toIndex);
            }
        }

        clearDragState();
    }
</script>

<ol
    bind:this={listElement}
    class="flex max-h-96 shrink-0 flex-col gap-2 overflow-y-auto py-2"
>
    {#each anchors as anchor, i (anchor.id)}
        {@const metrics = segmentMetrics(i)}
        {@const cumulative = cumulativeMetrics(i)}
        {@const showCumulative = i > 1 && cumulative != null}
        <li
            data-anchor-index={i}
            class="rounded-lg border border-input-border transition-colors hover:bg-secondary-hover"
            class:p-3={i !== 0}
            class:px-3={i === 0}
            class:pt-2={i === 0}
            class:pb-2={i === 0}
            class:has-cumulative={showCumulative}
            class:opacity-50={dragIndex === i}
            class:drop-above={isValidInsert(i)}
            class:drop-below={i === anchors.length - 1 && isValidInsert(anchors.length)}
            onmouseenter={handleItemMouseEnter}
            onmouseleave={handleItemMouseLeave}
        >
            <div class="grid grid-cols-[2.25rem_minmax(0,1fr)] gap-x-3 gap-y-1">
                <button
                    class="drag-handle relative row-start-1 flex h-9 w-9 items-center justify-center self-start rounded-full p-0 text-xl text-content hover:bg-secondary-hover disabled:cursor-not-allowed disabled:opacity-50"
                    type="button"
                    disabled={disabled}
                    aria-label={$_("move-route-point")}
                    aria-keyshortcuts="ArrowUp ArrowDown"
                    onkeydown={(e) => handleKeyDown(e, i)}
                    onpointerdown={(e) => handlePointerDown(e, i)}
                    onpointermove={handlePointerMove}
                    onpointerup={handlePointerUp}
                    onpointercancel={clearDragState}
                    onlostpointercapture={clearDragState}
                >
                    <i class="fa {anchorIcon(i)}"></i>
                    {#if i > 0 && i < anchors.length - 1}
                        <span
                            class="absolute -bottom-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-background px-1 text-[0.65rem] font-semibold leading-none text-gray-500"
                        >
                            {i + 1}
                        </span>
                    {/if}
                </button>

                <div class="anchor-title min-w-0">
                    <div
                        class="anchor-title-viewport"
                        role="presentation"
                        title={anchorTitle(anchor, i)}
                    >
                        <span class="anchor-title-text font-medium leading-5">
                            {anchorTitle(anchor, i)}
                        </span>
                    </div>
                    <p class="truncate text-xs leading-4 text-gray-500">
                        {anchorCoordinates(anchor)}
                    </p>
                </div>

                {#if metrics}
                    {#if showCumulative}
                        <span
                            class="cumulative-indicator col-start-1 row-start-2 flex h-5 items-center justify-end pr-1 text-sm font-semibold leading-5 text-gray-500"
                            title={$_("cumulative")}
                            aria-label={$_("cumulative")}
                        >
                            Σ
                        </span>
                    {/if}
                    <div class="col-start-2 row-start-2 min-w-0">
                        <div
                            class="segment-stats flex h-5 min-w-0 items-center gap-x-3 overflow-hidden text-sm leading-5 text-gray-500"
                        >
                            <span class="flex shrink-0 items-center whitespace-nowrap">
                                <i class="fa fa-left-right mr-1 w-4 text-center"></i>{formatDistance(metrics.distance, { compact: true })}
                            </span>
                            <span class="flex shrink-0 items-center whitespace-nowrap">
                                <i class="fa fa-arrow-trend-up mr-1 w-4 text-center"></i>{formatElevation(metrics.elevationGain)}
                            </span>
                            <span class="flex shrink-0 items-center whitespace-nowrap">
                                <i class="fa fa-arrow-trend-down mr-1 w-4 text-center"></i>{formatElevation(metrics.elevationLoss)}
                            </span>
                        </div>
                        {#if showCumulative}
                            <div
                                class="cumulative-stats flex h-5 min-w-0 items-center gap-x-3 overflow-hidden text-sm leading-5 text-gray-500"
                            >
                                <span class="flex shrink-0 items-center whitespace-nowrap">
                                    <i class="fa fa-left-right mr-1 w-4 text-center"></i>{formatDistance(cumulative.distance, { compact: true })}
                                </span>
                                <span class="flex shrink-0 items-center whitespace-nowrap">
                                    <i class="fa fa-arrow-trend-up mr-1 w-4 text-center"></i>{formatElevation(cumulative.elevationGain)}
                                </span>
                                <span class="flex shrink-0 items-center whitespace-nowrap">
                                    <i class="fa fa-arrow-trend-down mr-1 w-4 text-center"></i>{formatElevation(cumulative.elevationLoss)}
                                </span>
                            </div>
                        {/if}
                    </div>
                {/if}
            </div>

            <button
                class="delete-button btn-icon text-xs text-gray-400 hover:text-red-500"
                type="button"
                disabled={disabled}
                title={$_("delete")}
                aria-label={$_("delete-route-point")}
                onclick={() => onDelete(i)}
            >
                <i class="fa fa-trash"></i>
            </button>
        </li>
    {/each}
</ol>

<style>
    li {
        position: relative;
    }
    .delete-button {
        position: absolute;
        top: 0.5rem;
        right: 0.5rem;
        height: 1.75rem;
        font-size: 0.75rem;
    }
    .anchor-title {
        --delete-button-space: 2rem;
        --title-end-space: var(--delete-button-space);
    }
    .anchor-title-viewport {
        width: calc(100% - var(--title-end-space));
        min-width: 0;
        overflow: hidden;
        white-space: nowrap;
    }
    .anchor-title-text {
        display: inline-block;
        max-width: 100%;
        overflow: hidden;
        text-overflow: ellipsis;
        vertical-align: bottom;
        white-space: nowrap;
    }
    .drag-handle {
        cursor: grab;
        touch-action: none;
    }
    .drag-handle:active {
        cursor: grabbing;
    }
    .cumulative-indicator,
    .cumulative-stats {
        display: none;
    }
    li.drop-above::before,
    li.drop-below::after {
        content: "";
        position: absolute;
        left: 0;
        right: 0;
        height: 2px;
        border-radius: 9999px;
        background: rgba(var(--content));
    }
    li.drop-above::before {
        top: -6px;
    }
    li.drop-below::after {
        bottom: -6px;
    }

    @media (hover: hover) and (pointer: fine) {
        .anchor-title {
            --title-end-space: 0rem;
        }

        li:hover .anchor-title,
        li:focus-within .anchor-title {
            --title-end-space: var(--delete-button-space);
        }

        :global(.anchor-title-viewport.is-overflowing) .anchor-title-text {
            max-width: none;
            overflow: visible;
            text-overflow: clip;
            animation: anchor-title-marquee var(--title-scroll-duration, 3s)
                ease-in-out 0.15s infinite alternate;
        }

        .delete-button {
            opacity: 0;
            pointer-events: none;
            transition: opacity 0.15s ease-in-out;
        }

        li:hover .delete-button,
        li:focus-within .delete-button {
            opacity: 1;
            pointer-events: auto;
        }

        li.has-cumulative:hover .segment-stats,
        li.has-cumulative:focus-within .segment-stats {
            display: none;
        }

        li.has-cumulative:hover .cumulative-indicator,
        li.has-cumulative:focus-within .cumulative-indicator,
        li.has-cumulative:hover .cumulative-stats,
        li.has-cumulative:focus-within .cumulative-stats {
            display: flex;
        }
    }

    @keyframes anchor-title-marquee {
        from {
            transform: translateX(0);
        }
        to {
            transform: translateX(var(--title-scroll-distance, 0));
        }
    }

    @media (prefers-reduced-motion: reduce) {
        :global(.anchor-title-viewport.is-overflowing) .anchor-title-text {
            animation: none;
        }
    }
</style>
