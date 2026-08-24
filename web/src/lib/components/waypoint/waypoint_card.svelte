<script lang="ts">
    import type { Waypoint } from "$lib/models/waypoint";
    import {
        getFileURL,
        isVideoURL,
        readAsDataURLAsync,
    } from "$lib/util/file_util";
    import { _ } from "svelte-i18n";
    import Dropdown, { type DropdownItem } from "../base/dropdown.svelte";
    import { browser } from "$app/environment";
    import PhotoGallery from "../photo/photo_gallery.svelte";

    interface Props {
        waypoint: Waypoint;
        mode?: "show" | "edit";
        onchange?: (item: DropdownItem) => void;
    }

    let { waypoint, mode = "show", onchange }: Props = $props();

    let gallery: PhotoGallery | undefined = $state();

    let imgSrc: string[] = $state([]);
    $effect(() => {
        const persistedPhotos = (waypoint.photos ?? [])
            .map((p) => getFileURL(waypoint, p));
        const assetPluginPhotos = (waypoint._assetCandidates ?? [])
            .flatMap((candidate) => {
                if (!candidate.pluginId) {
                    return [];
                }
                const plugin = encodeURIComponent(candidate.pluginId);
                return `/api/v1/plugins/assets/${plugin}/thumbnail/${encodeURIComponent(candidate.assetId)}`;
            });
        const immediatePhotos = [...persistedPhotos, ...assetPluginPhotos];

        imgSrc = immediatePhotos.slice(-3).reverse();

        if (waypoint._photos?.length && browser) {
            Promise.all(
                waypoint._photos
                    .map(async (f) => {
                        return await readAsDataURLAsync(f);
                    }),
            ).then((v) => {
                imgSrc = [...immediatePhotos, ...v].slice(-3).reverse();
            });
        }
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
    <div class="min-w-0 flex-1">
        <div class="flex min-w-0 items-center justify-between gap-3 mb-2">
            <h5 class="min-w-0 flex-1 truncate">
                <i class="fa fa-{waypoint.icon} mr-2"></i>{waypoint.name}
            </h5>
            {#if mode == "edit"}
                <div class="shrink-0">
                    <Dropdown items={dropdownItems} {onchange}></Dropdown>
                </div>
            {/if}
        </div>

        {#if waypoint.description}
            <p>{@html waypoint.description}</p>
        {/if}

        <span class="text-sm text-gray-500"
            >{waypoint.lat.toFixed(5)}, {waypoint.lon.toFixed(5)}</span
        >
    </div>
</div>
