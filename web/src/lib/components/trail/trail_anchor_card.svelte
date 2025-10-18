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
        onmouseenter?: MouseEventHandler<HTMLDivElement>;
        onmouseleave?: MouseEventHandler<HTMLDivElement>;
    }

    let {
        anchor,
        onmouseenter,
        onmouseleave,
    }: Props = $props();

</script>

<div
    class="trail-anchor-card relative rounded-2xl border border-input-border min-w-72 h-[386px] cursor-pointer flex flex-col"
    {onmouseenter}
    {onmouseleave}
    role="listitem"
>
    <div class="p-4">
        <div>
            <div class="flex gap-x-4">
                {#if anchor.locationName}
                    <h5 class="text-overflow-ellipsis overflow-hidden whitespace-nowrap">
                        <i class="fa fa-location-dot mr-3"></i>{anchor.locationName}
                    </h5>
                {:else}
                    <h5>
                        <i class="fa fa-location-dot mr-3"></i>{anchor.lat.toFixed(4)},{anchor.lon.toFixed(4)}
                    </h5>
                {/if}
            </div>
        </div>
        <div
            class="grid grid-cols-3 mt-2 gap-1 text-sm text-gray-500 whitespace-nowrap"
        >
            <span
                ><i class="fa fa-left-right mr-2"></i>{formatDistance(anchor.distance)}</span
            >
            <span
                ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(
                    300,
                )}</span
            >
            <span
                ><i class="fa fa-arrow-trend-down mr-2"></i>{formatElevation(
                    200,
                )}</span
            >
        </div>
    </div>
</div>
