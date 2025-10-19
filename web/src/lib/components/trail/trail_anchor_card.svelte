<script lang="ts">
    import { type ValhallaAnchor } from "$lib/models/valhalla";
    import {
        formatDistance,
        formatElevation,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import type { MouseEventHandler } from "svelte/elements";

    interface Props {
        anchor: ValhallaAnchor;
        index: number;
        isLast?: boolean;
        onmouseenter?: MouseEventHandler<HTMLDivElement>;
        onmouseleave?: MouseEventHandler<HTMLDivElement>;
        onDelete?: (detail: { index: number; anchor: ValhallaAnchor }) => void;
    }

    let {
        anchor,
        index,
        isLast = false,
        onmouseenter,
        onmouseleave,
        onDelete,
    }: Props = $props();

    function handleDelete(e: MouseEvent) {
        e.stopPropagation();
        onDelete?.({ index, anchor });
    }

</script>

<div
    class="trail-anchor-card relative rounded-2xl border border-input-border min-w-72 cursor-pointer flex flex-col"
    {onmouseenter}
    {onmouseleave}
    role="listitem"
>
    <div class="p-4">
        <div>
            <div class="flex gap-x-4">
                <h5 class="text-overflow-ellipsis overflow-hidden whitespace-nowrap">
                    {#if index == 0}
                        <i class="fa fa-bullseye mr-3"></i>
                    {:else if isLast}
                        <i class="fa fa-flag-checkered mr-3"></i>
                    {:else}
                        <i class="fa fa-location-dot mr-3"></i>
                    {/if}
                    {#if anchor.locationName}
                        {anchor.locationName}
                    {:else}
                        {anchor.lat.toFixed(4)},{anchor.lon.toFixed(4)}
                    {/if}
                </h5>
            </div>
        </div>
        <div
            class="grid grid-cols-[auto_auto_auto_auto] mt-2 gap-1 text-sm text-gray-500 whitespace-nowrap"
        >
            <span>
                <!-- show default icon, and an alternate icon on hover -->
                <i class="fa fa-hashtag mr-2 icon-default" aria-hidden="true"></i>
                <i
                    class="fa fa-trash mr-2 icon-hover cursor-pointer"
                    aria-hidden="true"
                    title="Delete anchor"
                    onclick={handleDelete}
                ></i>
                {index + 1}
            </span>
            <span
                ><i class="fa fa-left-right mr-2"></i>{formatDistance(anchor.distance)}</span
            >
            <span
                ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(anchor.elevation_gain)}</span
            >
            <span
                ><i class="fa fa-arrow-trend-down mr-2"></i>{formatElevation(anchor.elevation_loss)}</span
            >
        </div>
    </div>
</div>


<style>
    .trail-anchor-card {
        object-fit: cover;
        transition: transform 0.25s ease;
        /* keep hover scale centered so it expands evenly */
        transform-origin: center center;
    }

    .trail-anchor-card:hover {
        scale: 1.02;
    }
    

    /* icon swap on card hover */
    .trail-anchor-card .icon-hover {
        display: none;
    }
    .trail-anchor-card:hover .icon-default {
        display: none;
    }
    .trail-anchor-card:hover .icon-hover {
        display: inline-block;
    }
</style>
