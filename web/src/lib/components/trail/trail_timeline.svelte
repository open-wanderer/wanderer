<script lang="ts">
    import type { Trail } from "$lib/models/trail";
    import { getFileURL, isVideoURL } from "$lib/util/file_util";
    import { formatDistance } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import PhotoGallery from "../photo_gallery.svelte";

    type Props = {
        trail: Trail;
        onmouseenter?: (index: number) => void;
        onmouseleave?: (index: number) => void;
    };

    let gallery: PhotoGallery[] = $state([]);

    let { trail, onmouseenter, onmouseleave }: Props = $props();
</script>

<div
    class="trail-timeline relative grid grid-cols-[auto_1fr] items-start gap-x-4 gap-y-6"
>
    <div class="bg-background">
        <i class="fa fa-bullseye"></i>
    </div>
    <div class="">
        <p class="font-semibold">{$_("start")}</p>
    </div>
    {#each trail.expand?.waypoints_via_trail ?? [] as wp, i}
        <div
            class="bg-background cursor-pointer"
            role="presentation"
            onmouseenter={() => onmouseenter?.(i)}
            onmouseleave={() => onmouseleave?.(i)}
        >
            <i class="fa fa-{wp.icon ?? 'circle'}"></i>
            {#if wp.distance_from_start}
                <p class="text-sm text-gray-500">
                    {formatDistance(wp.distance_from_start)}
                </p>
            {/if}
        </div>

        <div class="border border-input-border rounded-xl overflow-hidden">
            {#if wp.photos.length}
                <PhotoGallery
                    photos={wp.photos.map((p) => getFileURL(wp, p))}
                    bind:this={gallery[i]}
                ></PhotoGallery>
                <div
                    class="grid gap-[1px] {wp.photos.length > 1
                        ? 'grid-cols-[8fr_5fr]'
                        : 'grid-cols-1'} cursor-pointer"
                >
                    {#each wp.photos.slice(0,3) as photo, j}
                        {#if isVideoURL(photo)}
                            <!-- svelte-ignore a11y_media_has_caption -->
                            <video
                                controls={false}
                                loop
                                class="w-full object-cover {j == 0 &&
                                wp.photos.length > 2
                                    ? 'row-span-2 h-80'
                                    : 'h-[159.5px]'}"
                                onclick={() => gallery[i].openGallery(j)}
                                onmouseenter={(e) => (e.target as any).play()}
                                onmouseleave={(e) => (e.target as any).pause()}
                                src={getFileURL(wp, photo)}
                            ></video>
                        {:else}
                            <img
                                onclick={() => gallery[i].openGallery(j)}
                                role="presentation"
                                class="w-full object-cover {j == 0 &&
                                wp.photos.length > 2
                                    ? 'row-span-2 h-80'
                                    : 'h-[159.5px]'}"
                                src={getFileURL(wp, photo)}
                                alt=""
                            />
                        {/if}
                    {/each}
                </div>
            {/if}
            <div class="p-4">
                <h5 class="text-xl font-semibold">{wp.name}</h5>
                <span class="text-sm text-gray-500"
                    ><i class="fa fa-location-dot mr-1"></i>
                    {wp.lat.toFixed(5)}, {wp.lon.toFixed(5)}</span
                >
                <p class="whitespace-pre-line">
                    {@html wp.description}
                </p>
            </div>
        </div>
    {/each}
    <div class="bg-background">
        <i class="fa fa-flag-checkered"></i>
        {#if trail.distance}
            <p class="text-sm text-gray-500">
                {formatDistance(trail.distance)}
            </p>
        {/if}
    </div>
    <div class="">
        <p class="font-semibold">{$_("finish")}</p>
    </div>
</div>

<style>
    .trail-timeline::before {
        border-right: 2px dotted rgba(var(--input-border));
        bottom: 0;
        content: "";
        left: 7.2px;
        position: absolute;
        top: 0;
        width: 1px;
        z-index: -1;
    }
</style>
