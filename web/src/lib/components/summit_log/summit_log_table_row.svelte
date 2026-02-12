<script lang="ts">
    import type { SummitLog } from "$lib/models/summit_log";

    import { getFileURL, isVideoURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import PhotoGallery from "../photo_gallery.svelte";
    import Dropdown, { type DropdownItem } from "../base/dropdown.svelte";

    interface Props {
        log: SummitLog;
        handle: string;
        showCategory?: boolean;
        showTrail?: boolean;
        showRoute?: boolean;
        showAuthor?: boolean;
        showDescription?: boolean;
        showPhotos?: boolean;
        showMenu?: boolean;
        ontext?: (summitLog: SummitLog) => void;
        onopen?: (summitLog: SummitLog) => void;
        ondelete?: (summitLog: SummitLog) => void;
        onedit?: (summitLog: SummitLog) => void;
    }

    let {
        log,
        handle,
        showCategory = false,
        showTrail = false,
        showRoute = false,
        showAuthor = false,
        showDescription = false,
        showPhotos = false,
        showMenu = false,
        onopen,
        ontext,
        ondelete,
        onedit,
    }: Props = $props();

    let gallery: PhotoGallery | undefined = $state();

    let imgSrc: string[] = $state([]);

    let dropdownItems: DropdownItem[] = [
        {
            text: $_("edit"),
            value: "edit",
        },
        {
            text: $_("delete"),
            value: "delete",
        },
    ];
    $effect(() => {
        if (log.photos?.length) {
            imgSrc = log.photos
                .filter((_, i) => i < 3)
                .reverse()
                .map((p) => getFileURL(log, p));
        } else {
            imgSrc = [];
        }
    });

    function openText() {
        ontext?.(log);
    }

    function openRoute() {
        onopen?.(log);
    }

    function colCount() {
        return [
            showCategory,
            showTrail,
            showAuthor,
            showRoute,
            showDescription,
        ].reduce((b, v) => (v ? b + 1 : b), 7);
    }

    function handleDropdownClick(item: DropdownItem): void {
        if (item.value == "edit") {
            onedit?.(log);
        } else if (item.value == "delete") {
            ondelete?.(log);
        }
    }
</script>

<tr class="text-sm">
    {#if showPhotos}
        <td>
            {#if imgSrc.length}
                <PhotoGallery
                    photos={log.photos.map((p) => getFileURL(log, p))}
                    bind:this={gallery}
                ></PhotoGallery>
                <button
                    class="relative w-16 aspect-square ml-2 mb-3 shrink-0"
                    type="button"
                    onclick={() => gallery?.openGallery()}
                >
                    {#each imgSrc as img, i}
                        {#if isVideoURL(img)}
                            <!-- svelte-ignore a11y_media_has_caption -->
                            <video
                                controls={false}
                                loop
                                class="absolute h-full w-full rounded-xl object-cover"
                                style="top: {4 * i}px; right: {4 *
                                    i}px; transform: rotate(-{i * 5}deg)"
                                onmouseenter={(e) => (e.target as any).play()}
                                onmouseleave={(e) => (e.target as any).pause()}
                                src={img}
                            ></video>
                        {:else}
                            <img
                                class="absolute h-full rounded-xl object-cover"
                                style="top: {4 * i}px; right: {4 *
                                    i}px; transform: rotate(-{i * 5}deg)"
                                src={img}
                                alt="waypoint"
                            />
                        {/if}
                    {/each}
                </button>
            {/if}
        </td>
    {/if}
    <td class:py-4={!log.expand?.gpx_data}
        >{new Date(log.date).toLocaleDateString(undefined, {
            month: "2-digit",
            day: "2-digit",
            year: "numeric",
            timeZone: "UTC",
        })}</td
    >
    <td>
        {formatDistance(log.distance)}
    </td>

    <td>
        {formatElevation(log.elevation_gain)}
    </td>
    <td>
        {formatElevation(log.elevation_loss)}
    </td>
    <td>
        {formatTimeHHMM(log.duration ? log.duration : undefined)}
    </td>
    {#if showCategory}
        <td>
            {$_(log.expand?.trail?.expand?.category?.name ?? "-")}
        </td>
    {/if}
    {#if showTrail}
        <td>
            <a
                aria-label="Go to trail"
                class="btn-icon aspect-square"
                href="/trail/view/{handle}/{log.expand?.trail?.id ?? ''}"
                ><i class="fa fa-arrow-up-right-from-square px-[3px]"></i></a
            >
        </td>
    {/if}
    {#if showDescription}
        <td>
            {#if log.text}
                <button onclick={openText}
                    ><p
                        class="rounded-full bg-menu-item-background-hover hover:bg-menu-item-background-focus text-ellipsis max-w-28 whitespace-nowrap overflow-hidden px-3 py-1"
                    >
                        {formatHTMLAsText(log.text)}
                    </p></button
                >
            {/if}
        </td>
    {/if}
    {#if showAuthor && log.expand?.author}
        <td>
            <p
                class="tooltip flex justify-center"
                data-title="@{log.expand.author.preferred_username}{log.expand.author
                    .isLocal
                    ? ''
                    : '@' + log.expand.author.domain}"
            >
                <a
                    href="/profile/@{log.expand.author.preferred_username?.toLowerCase()}{log
                        .expand.author.isLocal
                        ? ''
                        : '@' + log.expand.author.domain}"
                >
                    <img
                        class="rounded-full w-7 aspect-square"
                        src={log.expand.author.icon ||
                            `https://api.dicebear.com/7.x/initials/svg?seed=${log.expand.author.preferred_username}&backgroundType=gradientLinear`}
                        alt="avatar"
                    />
                </a>
            </p>
        </td>
    {/if}
    {#if showRoute}
        <td>
            {#if log.gpx}
                <button
                    aria-label="Open route"
                    onclick={openRoute}
                    class="btn-icon"
                >
                    <i class="fa fa-map-location-dot px-[3px] text-xl"
                    ></i></button
                >
            {/if}
        </td>
    {/if}
    {#if showMenu}
        <td>
            <Dropdown onchange={handleDropdownClick} items={dropdownItems}>
                {#snippet children({ toggleMenu: openDropdown })}
                    <button
                        aria-label="Open dropdown"
                        class="rounded-full bg-white text-black hover:bg-gray-200 focus:ring-4 ring-gray-100/50 transition-colors h-6 w-6"
                        onclick={openDropdown}
                    >
                        <i class="fa fa-ellipsis-vertical"></i>
                    </button>
                {/snippet}
            </Dropdown>
        </td>
    {/if}
</tr>
<tr>
    <td colspan={colCount()}> <hr class="border-input-border" /> </td>
</tr>

<style>
    td {
        padding: 0.3rem 0.5rem;
    }
</style>
