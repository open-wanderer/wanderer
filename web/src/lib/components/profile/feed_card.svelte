<script lang="ts">
    import type { FeedItem } from "$lib/models/feed";
    import type { List } from "$lib/models/list";
    import type { SummitLog } from "$lib/models/summit_log";
    import type { Trail } from "$lib/models/trail";
    import { getFileURL, isVideoURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatHTMLAsTextPreview,
        formatTimeHHMM,
        formatTimeSince,
    } from "$lib/util/format_util";
    import {
        displayCategoryIcon,
        displayCategoryName,
        displaySubcategoryIcon,
        displaySubcategoryLabel,
    } from "$lib/util/category_util";
    import { _, locale } from "svelte-i18n";
    import TrailDropdown from "../trail/trail_dropdown.svelte";
    interface Props {
        feedItem: FeedItem;
    }

    let { feedItem }: Props = $props();

    let fullDescription = $state(false);

    const DESCRIPTION_PREVIEW_LENGTH = 100;

    const timeSince = $derived(
        formatTimeSince(new Date(feedItem.created ?? "")),
    );

    const trail = $derived(
        feedItem.type === "trail"
            ? (feedItem.expand.item as Trail)
            : undefined,
    );
    const list = $derived(
        feedItem.type === "list" ? (feedItem.expand.item as List) : undefined,
    );
    const summitLog = $derived(
        feedItem.type === "summit_log"
            ? (feedItem.expand.item as SummitLog)
            : undefined,
    );
    const activity = $derived(trail ?? summitLog);

    const photos = $derived(activity?.photos ?? []);
    const location = $derived(trail?.location);
    const category = $derived(trail?.expand?.category);
    const subcategory = $derived(trail?.expand?.subcategory);
    const trails = $derived(list?.trails);

    const author = $derived(feedItem.expand.item.expand?.author);
    const actorHandle = $derived(
        `${author?.preferred_username}@${author?.domain}`,
    );
    const itemTitle = $derived(
        trail?.name ??
            list?.name ??
            summitLog?.expand?.trail?.name ??
            $_("summit-log", { values: { n: 1 } }),
    );
    const itemDescription = $derived(
        trail?.description ?? list?.description ?? summitLog?.text ?? "",
    );
    const descriptionPreview = $derived(
        formatHTMLAsTextPreview(itemDescription, DESCRIPTION_PREVIEW_LENGTH),
    );
    const itemHref = $derived(
        feedItem.type === "trail"
            ? `/trail/view/@${actorHandle}/${feedItem.item}`
            : feedItem.type === "list"
              ? `/lists/@${actorHandle}/${feedItem.item}`
              : `/profile/${actorHandle}/stats`,
    );
    const photoCollection = $derived(
        feedItem.type === "summit_log" ? "summit_logs" : "trails",
    );

    function feedCategoryIcon() {
        if (subcategory) {
            return displaySubcategoryIcon(subcategory, category);
        }

        return displayCategoryIcon(category);
    }
</script>

<div class="feed-card px-6 py-4 rounded-xl border border-input-border">
    <p class="mb-2 text-gray-500 text-sm">
        {#if feedItem.type === "trail"}
            <i class="fa fa-route mr-2"></i>{$_("trail", { values: { n: 1 } })}
        {:else if feedItem.type === "list"}
            <i class="fa fa-layer-group mr-2"></i>{$_("list", {
                values: { n: 1 },
            })}
        {:else if feedItem.type === "summit_log"}
            <i class="fa fa-mountain mr-2"></i>{$_("summit-log", {
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
        href={itemHref}
    >
        <div class="feed-card-body">
            <h3 class="text-2xl font-semibold mb-2">
                {itemTitle}
            </h3>
            <div class="flex flex-wrap gap-x-8 gap-y-1">
                {#if category}
                    <p>
                        <i
                            class="fa {feedCategoryIcon()} mr-3"
                        ></i>{displayCategoryName(
                            category,
                            $locale,
                        )}
                        {#if subcategory}
                            <span class="text-gray-500">
                                / {displaySubcategoryLabel(
                                    subcategory,
                                    $locale,
                                )}
                            </span>
                        {/if}
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
                        activity?.distance,
                    )}</span
                >
                <span
                    ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                        activity?.duration,
                    )}</span
                >
                <span
                    ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(
                        activity?.elevation_gain,
                    )}</span
                >
                <span
                    ><i class="fa fa-arrow-trend-down mr-2"
                    ></i>{formatElevation(
                        activity?.elevation_loss,
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
                                        collectionId: photoCollection,
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
                                        collectionId: photoCollection,
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
            {#if itemDescription.length}
                <p class="text-sm whitespace-pre-wrap mt-6">
                    {!fullDescription
                        ? descriptionPreview.text
                        : formatHTMLAsText(itemDescription)}
                    {#if descriptionPreview.truncated && !fullDescription}
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
