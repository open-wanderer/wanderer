<script lang="ts">
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import type { Trail } from "$lib/models/trail";
    import { theme } from "$lib/stores/theme_store";
    import { currentUser } from "$lib/stores/user_store";
    import { getFileURL, isVideoURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import ShareInfo from "../share_info.svelte";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import Chip from "../base/chip.svelte";

    interface Props {
        trail: Trail;
        showDescription?: boolean;
        selected: boolean;
        hovered: boolean;
        onTrailSelect?: () => void;
    }

    let {
        trail,
        showDescription = true,
        selected = false,
        hovered = false,
        onTrailSelect,
    }: Props = $props();

    let thumbnail = $derived(
        trail.photos.length
            ? getFileURL(
                  trail,
                  trail.photos.at(trail.thumbnail ?? 0) ?? trail.photos[0],
                  "600x0",
              )
            : $theme === "light"
              ? emptyStateTrailLight
              : emptyStateTrailDark,
    );

    function handleInputClick(e: Event) {
        e.stopPropagation();
        onTrailSelect?.();
        hovered = true;
    }

    let expandedTags = $state(false);

    function toggleExpandTags(e: MouseEvent) {
        e.preventDefault();
        e.stopPropagation();
        expandedTags = !expandedTags;
    }
</script>

<li
    class="flex gap-8 p-4 rounded-xl border border-input-border cursor-pointer hover:bg-secondary-hover transition-colors items-center"
    class:bg-secondary-hover={selected}
>
    <div class="shrink-0">
        {#if isVideoURL(thumbnail)}
            <!-- svelte-ignore a11y_media_has_caption -->
            <video
                class="h-28 w-28 object-cover rounded-xl"
                autoplay
                loop
                src={thumbnail}
            ></video>
        {:else}
            <img
                class="h-28 w-28 object-cover rounded-xl"
                src={thumbnail}
                alt=""
            />
        {/if}
    </div>
    <div class="min-w-0 basis-full relative">
        <div class="flex items-center justify-between">
            <h4 class="font-semibold text-lg line-clamp-2 wrap-anywhere">
                {trail.name}
            </h4>
            <div class="flex items-center shrink-0 gap-3">
                {#if trail.public && $currentUser}
                    <span class="tooltip" data-title={$_("public")}>
                        <i class="fa fa-globe"></i>
                    </span>
                {/if}
                {#if trail.expand?.trail_share_via_trail?.length}
                    <ShareInfo type="trail" subject={trail}></ShareInfo>
                {/if}
                {#if $currentUser && trail.like_count > 0}
                    <div class="flex items-start gap-x-[2px]">
                        <span class="tooltip" data-title={$_("likes")}>
                            <i class="fa fa-heart"></i>
                        </span>
                        <div
                            class="pointer-events-none left-4 -top-1 text-xs rounded-full bg-content text-content-inverse px-1 text-center"
                        >
                            {trail.like_count}
                        </div>
                    </div>
                {/if}
            </div>
        </div>
        {#if trail.date}
            <p class="text-xs text-gray-500 mb-3">
                {new Date(trail.date).toLocaleDateString(undefined, {
                    month: "long",
                    day: "2-digit",
                    year: "numeric",
                    timeZone: "UTC",
                })}
            </p>
        {/if}
        {#if trail.expand?.author}
            <p class="text-xs text-gray-500 mb-3">
                {$_("by")}
                <img
                    class="rounded-full w-5 aspect-square mx-1 inline"
                    src={trail.expand.author.icon ||
                        `https://api.dicebear.com/7.x/initials/svg?seed=${trail.expand.author.preferred_username}&backgroundType=gradientLinear`}
                    alt="avatar"
                />
                {handleFromRecordWithIRI(trail)}
            </p>
        {/if}
        {#if trail.tags.length && trail.expand?.tags}
            <div class="flex flex-wrap gap-1 mb-3 items-center">
                {#each expandedTags ? trail.expand.tags : trail.expand.tags.slice(0, 2) as tag}
                    <Chip text={tag.name} closable={false} primary={false}></Chip>
                {/each}

                {#if trail.expand.tags.length > 2}
                    <button
                        onclick={toggleExpandTags}
                        class="text-sm text-gray-500 hover:underline focus:outline-none"
                        type="button"
                    >
                        {#if expandedTags}
                            Show less
                        {:else}
                            +{trail.expand.tags.length - 2} more
                        {/if}
                    </button>
                {/if}
            </div>
        {/if}
        <div class="flex flex-wrap gap-x-8 gap-y-1">
            {#if trail.expand?.category?.name || trail.category}
                <p>
                    <i class="fa fa-shapes mr-3"> </i>{$_(trail.expand?.category?.name ?? trail.category ?? "-")}
                </p>
            {/if}
            {#if trail.location}
                <p>
                    <i class="fa fa-location-dot mr-3"></i>{trail.location}
                </p>
            {/if}
            <p class="whitespace-nowrap">
                <i class="fa fa-gauge mr-3"></i>{$_(trail.difficulty ?? "?")}
            </p>
        </div>

        <div class="flex flex-wrap mt-1 gap-x-4 gap-y-2 text-sm text-gray-500">
            <span
                ><i class="fa fa-left-right mr-2"></i>{formatDistance(
                    trail.distance,
                )}</span
            >
            <span
                ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                    trail.duration,
                )}</span
            >
            <span
                ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(
                    trail.elevation_gain,
                )}</span
            >
            <span
                ><i class="fa fa-arrow-trend-down mr-2"></i>{formatElevation(
                    trail.elevation_loss,
                )}</span
            >
        </div>
        {#if showDescription}
            <p
                class="mt-3 text-sm whitespace-nowrap min-w-0 max-w-full overflow-hidden text-ellipsis basis-full"
            >
                {formatHTMLAsText(trail.description ?? "")}
            </p>
        {/if}
        {#if hovered || selected}
            <div
                class="flex absolute bottom-0 right-0 w-8 h-8 rounded-full items-center justify-center bg-background text-content"
            >
                <input
                    id="trail-selected"
                    type="checkbox"
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                    bind:checked={selected}
                    onclick={(e) => handleInputClick(e)}
                />
            </div>
        {/if}
    </div>
</li>
