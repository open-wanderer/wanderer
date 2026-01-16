<script lang="ts">
    import type { FeedItem } from "$lib/models/feed";
    import type { List } from "$lib/models/list";
    import type { Trail } from "$lib/models/trail";
    import { getFileURL, isVideoURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatTimeHHMM,
        formatTimeSince,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import TrailDropdown from "../trail/trail_dropdown.svelte";
    interface Props {
        feedItem: FeedItem;
    }

    let { feedItem }: Props = $props();

    let fullDescription = $state(false);

    const timeSince = $derived(
        formatTimeSince(new Date(feedItem.created ?? "")),
    );

    const photos = $derived((feedItem.expand.item as Trail).photos);
    const location = $derived((feedItem.expand.item as Trail).location);
    const category = $derived((feedItem.expand.item as Trail).expand?.category?.name);

    const trails = $derived((feedItem.expand.item as List).trails);

    const author = $derived(feedItem.expand.item.expand?.author);
</script>

<div class="feed-card px-6 py-4 rounded-xl border border-input-border">
    <p class="mb-2 text-gray-500 text-sm">
        {#if feedItem.type === "trail"}
            <i class="fa fa-route mr-2"></i>{$_("trail", { values: { n: 1 } })}
        {:else if feedItem.type === "list"}
            <i class="fa fa-layer-group mr-2"></i>{$_("list", {
                values: { n: 1 },
            })}
        {/if}
    </p>

    <a href="/profile/{author?.preferred_username}@{author?.domain}">
        <div class="feed-card-header flex gap-x-4 items-start">
            <img
                class="rounded-full w-10 aspect-square overflow-hidden shrink-0"
                src={author?.icon ||
                    `https://api.dicebear.com/7.x/initials/svg?seed=${author?.preferred_username}&backgroundType=gradientLinear`}
                alt="avatar"
            />
            <div>
                <span class="font-semibold">{author?.preferred_username}</span>
                <p class="text-sm text-gray-500 mb-3">
                    {author?.preferred_username}@{author?.domain}
                </p>
            </div>
            <div class="basis-full"></div>
            <p class="text-xs text-gray-500 shrink-0">
                {$_(`n-${timeSince.unit}-ago`, {
                    values: { n: timeSince.value },
                })}
            </p>
        </div>
    </a>
    <a
        class="block"
        href={(feedItem.type === "trail" ? "/trail/view/" : "/lists/") +
            `@${author?.preferred_username}@${author?.domain}/${feedItem.item}`}
    >
        <div class="feed-card-body">
            <h3 class="text-2xl font-semibold mb-2">
                {feedItem.expand.item.name}
            </h3>
            <div class="flex flex-wrap gap-x-8 gap-y-1">
                {#if category}
                    <p>
                        <i class="fa fa-shapes mr-3"> </i>{$_(category)}
                    </p>
                {/if}
                {#if location}
                    <p>
                        <i class="fa fa-location-dot mr-3"></i>{location}
                    </p>
                {/if}
            </div>
            <div
                class="flex flex-wrap mt-1 gap-x-4 gap-y-2 text-sm text-gray-500 mb-2"
            >
                <span
                    ><i class="fa fa-left-right mr-2"></i>{formatDistance(
                        feedItem.expand.item.distance,
                    )}</span
                >
                <span
                    ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                        feedItem.expand.item.duration,
                    )}</span
                >
                <span
                    ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(
                        feedItem.expand.item.elevation_gain,
                    )}</span
                >
                <span
                    ><i class="fa fa-arrow-trend-down mr-2"
                    ></i>{formatElevation(
                        feedItem.expand.item.elevation_loss,
                    )}</span
                >
            </div>
            {#if trails}
                <p class="text-sm text-gray-500">
                    {trails.length}
                    {$_("trail", {
                        values: { n: trails.length },
                    })}
                </p>
            {/if}
            {#if photos?.length}
                <div
                    class="grid gap-[1px] {photos.length > 1
                        ? 'grid-cols-[8fr_5fr]'
                        : 'grid-cols-1'} mt-4"
                >
                    {#each photos.slice(0, 3) as photo, i}
                        {#if isVideoURL(photo)}
                            <!-- svelte-ignore a11y_media_has_caption -->
                            <video
                                class="object-cover h-full max-h-80 w-full"
                                autoplay
                                loop
                                src={getFileURL(
                                    {
                                        collectionId: "trails",
                                        id: feedItem.item,
                                    },
                                    photo,
                                )}
                            ></video>
                        {:else}
                            <img
                                class="object-cover h-full max-h-80 w-full"
                                class:row-span-2={i == 0 && photos.length > 2}
                                src={getFileURL(
                                    {
                                        collectionId: "trails",
                                        id: feedItem.item,
                                    },
                                    photo,
                                )}
                                alt=""
                            />
                        {/if}
                    {/each}
                </div>
            {/if}
            {#if feedItem.expand.item.description?.length}
                <p class="text-sm whitespace-pre-wrap mt-6">
                    {formatHTMLAsText(
                        !fullDescription
                            ? feedItem.expand.item.description?.substring(
                                  0,
                                  100,
                              )
                            : feedItem.expand.item.description,
                    )}
                    {#if (feedItem.expand.item.description?.length ?? 0) > 100 && !fullDescription}
                        <button
                            onclick={(e) => {
                                e.stopPropagation();
                                e.preventDefault();
                                fullDescription = true;
                            }}
                        >
                            ... <span class="underline">{$_("read-more")}</span
                            ></button
                        >
                    {/if}
                </p>
            {/if}
        </div>
    </a>
    {#if feedItem.type == "trail"}
        <div class="feed-card-actions flex items-center justify-end mt-4">
            <TrailDropdown
                trails={new Set<Trail>([feedItem.expand.item as Trail])}
                mode="overview"
            >
                {#snippet toggle({ toggleMenu: openDropdown })}
                    <button
                        class="btn-icon"
                        onclick={openDropdown}
                        aria-label="Trail actions"
                        type="button"
                        ><i class="fa fa-ellipsis-vertical"></i></button
                    >
                {/snippet}</TrailDropdown
            >
        </div>
    {/if}
</div>
