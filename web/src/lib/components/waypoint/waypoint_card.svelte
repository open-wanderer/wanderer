<script lang="ts">
    import { sanitizeHTML } from "#lib/util/sanitize_html.ts";
    import type { Waypoint } from "#lib/models/waypoint.ts";
    import {
        getFileURL,
        isVideoURL,
    } from "#lib/util/file_util.ts";
    import { _ } from "svelte-i18n";
    import Dropdown, { type DropdownItem } from "../base/dropdown.svelte";
    import { browser } from "$app/env";
    import PhotoGallery from "../photo/photo_gallery.svelte";

    interface Props {
        waypoint: Waypoint;
        mode?: "show" | "edit";
        onchange?: (item: DropdownItem) => void;
    }

    let { waypoint, mode = "show", onchange }: Props = $props();

    let gallery: PhotoGallery | undefined = $state();

    import { waypointFilePreviews } from "#lib/util/photo_preview_util.ts";

    let imgSrc: string[] = $state([]);
    $effect(() => {
        let cancelled = false;
        if (waypoint.photos?.length) {
            imgSrc = waypoint.photos
                .filter((_, i) => i < 3)
                .reverse()
                .map((p) => getFileURL(waypoint, p, "300x300"));
        } else if (waypoint._photos?.length && browser) {
            waypointFilePreviews(waypoint._photos).then((previews) => {
                if (!cancelled) imgSrc = previews;
            }).catch(() => {});
        } else {
            imgSrc = [];
        }
        return () => { cancelled = true; };
    });

    const dropdownItems = [
        { text: $_("edit"), value: "edit" },
        { text: $_("delete"), value: "delete" },
    ];
</script>

<div
    class="flex gap-4 p-4 outline outline-1 outline-input-border rounded-md my-2 hover:outline-2 items-start"
>
    {#if imgSrc.length}
        {#if mode == "show"}
            <PhotoGallery
                photos={waypoint.photos.map((p) => getFileURL(waypoint, p))}
                bind:this={gallery}
            ></PhotoGallery>
        {/if}
        <button
            class="relative basis-16 aspect-square ml-2 mb-3 shrink-0"
            type="button"
            onclick={mode == "show" ? () => gallery?.openGallery() : undefined}
        >
            {#each imgSrc as img, i}
                {#if isVideoURL(img)}
                    <!-- svelte-ignore a11y_media_has_caption -->
                    <video
                        controls={false}
                        loop
                        class="absolute h-full rounded-xl object-cover aspect-square"
                        style="top: {6 * i}px; right: {6 *
                            i}px; transform: rotate(-{i * 5}deg)"
                        onmouseenter={(e) => (e.target as any).play()}
                        onmouseleave={(e) => (e.target as any).pause()}
                        src={img}
                    ></video>
                {:else}
                    <img
                        class="absolute h-full rounded-xl object-cover aspect-square"
                        style="top: {6 * i}px; right: {6 *
                            i}px; transform: rotate(-{i * 5}deg)"
                        src={img}
                        alt="waypoint"
                    />
                {/if}
            {/each}
        </button>
    {/if}
    <div class="basis-full">
        <div class="flex justify-between items-center mb-2">
            <h5>
                <i class="fa fa-{waypoint.icon} mr-2"></i>{waypoint.name}
            </h5>
            {#if mode == "edit"}
                <Dropdown items={dropdownItems} {onchange}></Dropdown>
            {/if}
        </div>

        {#if waypoint.description}
            <div>{@html sanitizeHTML(waypoint.description)}</div>
        {/if}

        <span class="text-sm text-gray-500"
            >{waypoint.lat.toFixed(5)}, {waypoint.lon.toFixed(5)}</span
        >
    </div>
</div>
