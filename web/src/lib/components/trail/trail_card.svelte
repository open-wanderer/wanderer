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
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import type { MouseEventHandler } from "svelte/elements";
    import Chip from "../base/chip.svelte";

    interface Props {
        trail: Trail;
        fullWidth?: boolean;
        selected: boolean;
        hovered: boolean;
        onmouseenter?: MouseEventHandler<HTMLDivElement>;
        onmouseleave?: MouseEventHandler<HTMLDivElement>;
        onTrailSelect?: () => void;
    }

    let {
        trail,
        fullWidth = false,
        selected = false,
        hovered = false,
        onmouseenter,
        onmouseleave,
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

    let trailIsShared = $derived(
        (trail.expand?.trail_share_via_trail?.length ?? 0) > 0,
    );

    function handleInputClick(e: Event) {
        e.stopPropagation();
        onTrailSelect?.();
        hovered = true;
    }
    // expand and collapse the tags
    let expandedTags = $state(false);

    function toggleExpandTags(e: MouseEvent) {
        e.preventDefault();
        e.stopPropagation();
        expandedTags = !expandedTags;
    }
</script>

<div
    class="trail-card relative rounded-2xl border border-input-border min-w-72 h-[386px] bg-background {fullWidth
        ? ''
        : 'lg:w-72'} cursor-pointer flex flex-col"
    class:bg-secondary-hover={selected}
    {onmouseenter}
    {onmouseleave}
    role="listitem"
>
    <div
        class="relative w-full basis-full max-h-48 overflow-hidden rounded-t-2xl"
    >
        {#if isVideoURL(thumbnail)}
            <!-- svelte-ignore a11y_media_has_caption -->
            <video
                id="header-img"
                class="w-full h-full object-cover"
                autoplay
                loop
                src={thumbnail}
            ></video>
        {:else}
            <img
                loading="lazy"
                class="w-full h-full"
                id="header-img"
                src={thumbnail}
                alt=""
            />
        {/if}
    </div>
    {#if hovered || selected}
        <div
            class="flex absolute top-4 left-4 w-8 h-8 rounded-full items-center justify-center bg-background text-content"
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
    {#if (trail.public || trailIsShared) && $currentUser}
        <div
            class="flex absolute top-4 right-4 {trail.public && trailIsShared
                ? 'w-14'
                : 'w-8'} h-8 rounded-full items-center justify-center bg-background text-content"
        >
            {#if trail.public && $currentUser}
                <span
                    class="tooltip"
                    class:mr-2={trail.public && trailIsShared}
                    data-title={$_("public")}
                >
                    <i class="fa fa-globe"></i>
                </span>
            {/if}
            {#if trail.expand?.trail_share_via_trail?.length}
                <span class="tooltip" data-title={$_("shared")}>
                    <i class="fa fa-share-nodes"></i>
                </span>
            {/if}
        </div>
    {/if}
    {#if $currentUser && trail.like_count > 0}
        <div
            class="flex absolute items-center justify-center {trailIsShared ||
            trail.public
                ? 'top-14'
                : 'top-4'} right-4 bg-background w-8 h-8 rounded-full"
        >
            <span class="tooltip" data-title={$_("likes")}>
                <i class="fa fa-heart"></i>
            </span>
            <div
                class="absolute pointer-events-none left-5 -top-1 text-xs rounded-full bg-menu-background px-1 text-center"
            >
                {trail.like_count}
            </div>
        </div>
    {/if}
    <div class="p-4">
        <div>
            <h4 class="font-semibold text-lg line-clamp-2 wrap-anywhere">{trail.name}</h4>
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
                    {trail.expand.author.preferred_username}{trail.expand.author
                        .isLocal
                        ? ""
                        : "@" + trail.expand.author.domain}
                </p>
            {/if}
            {#if trail.tags.length}
                <div class="flex flex-wrap gap-1 mb-3 items-center">
                    {#each expandedTags ? trail.tags : trail.tags.slice(0, 2) as t}
                        <Chip text={t} closable={false} primary={false}></Chip>
                    {/each}

                    {#if trail.tags.length > 2}
                        <button
                            onclick={toggleExpandTags}
                            class="text-sm text-gray-500 hover:underline focus:outline-none"
                            type="button"
                        >
                            {#if expandedTags}
                                {$_("show-less")}
                            {:else}
                                +{trail.tags.length - 2} {$_("more")}
                            {/if}
                        </button>
                    {/if}
                </div>
            {/if}
            <div class="flex gap-x-4 gap-y-1 text-base flex-wrap">
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
                    <i class="fa fa-gauge mr-3"></i>{$_(
                        trail.difficulty ?? "?",
                    )}
                </p>
            </div>
        </div>
        <div
            class="grid grid-cols-2 mt-2 gap-1 text-sm text-gray-500 whitespace-nowrap"
        >
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
    </div>
</div>

<style>
    .trail-card #header-img {
        object-fit: cover;
        transition: 0.25s ease;
    }

    .trail-card:hover #header-img {
        scale: 1.075;
    }
</style>
