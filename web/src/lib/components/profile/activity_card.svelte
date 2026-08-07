<script lang="ts">
    import type { Actor } from "$lib/models/activitypub/actor";
    import type { TimelineItem } from "$lib/models/timeline";
    import { getFileURL, isVideoURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatHTMLAsTextPreview,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    interface Props {
        activity: TimelineItem;
        actor: Actor;
    }

    let { activity, actor }: Props = $props();

    let fullDescription: boolean = $state(false);

    const DESCRIPTION_PREVIEW_LENGTH = 100;

    let descriptionPreview = $derived(
        formatHTMLAsTextPreview(
            activity.description,
            DESCRIPTION_PREVIEW_LENGTH,
        ),
    );
</script>

<div class="activity-card p-6 ounded-xl border border-input-border">
    <div class="flex gap-x-4 items-start">
        <img
            class="rounded-full w-10 aspect-square overflow-hidden"
            src={actor.icon ||
                `https://api.dicebear.com/7.x/initials/svg?seed=${actor.preferred_username}&backgroundType=gradientLinear`}
            alt="avatar"
        />
        <div>
            <span class="font-semibold">{actor.preferred_username}</span>
            {activity.type === "trail"
                ? $_("planned-a-trail")
                : $_("completed-a-trail")}
            <p class="text-xs text-gray-500 mb-3">
                {new Date(activity.created).toLocaleDateString(undefined, {
                    month: "long",
                    day: "2-digit",
                    year: "numeric",
                    timeZone: "UTC",
                })}
            </p>
        </div>
    </div>
    <h3 class="text-2xl font-semibold !mt-2">{activity.name}</h3>
    <div class="flex flex-wrap mt-1 gap-x-4 gap-y-2 text-sm text-gray-500">
        <span
            ><i class="fa fa-left-right mr-2"></i>{formatDistance(
                activity.distance,
            )}</span
        >
        <span
            ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                activity.duration,
            )}</span
        >
        <span
            ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(
                activity.elevation_gain,
            )}</span
        >
        <span
            ><i class="fa fa-arrow-trend-down mr-2"></i>{formatElevation(
                activity.elevation_loss,
            )}</span
        >
    </div>
    {#if activity.photos.length}
        <div
            class="grid gap-[1px] {activity.photos.length > 1
                ? 'grid-cols-[8fr_5fr]'
                : 'grid-cols-1'} mt-6"
        >
            {#each activity.photos.slice(0, 3) as photo, i}
                {#if isVideoURL(photo)}
                    <!-- svelte-ignore a11y_media_has_caption -->
                    <video
                        class="object-cover h-full max-h-80 w-full"
                        autoplay
                        loop
                        src={getFileURL(
                            {
                                collectionId: activity.type + "s",
                                id: activity.id,
                            },
                            photo,
                        )}
                    ></video>
                {:else}
                    <img
                        class="object-cover h-full max-h-80 w-full"
                        class:row-span-2={i == 0 && activity.photos.length > 2}
                        src={getFileURL(
                            {
                                collectionId: activity.type + "s",
                                id: activity.id,
                            },
                            photo,
                        )}
                        alt=""
                    />
                {/if}
            {/each}
        </div>
    {/if}
    {#if activity.description.length}
        <p class="text-sm whitespace-pre-wrap mt-6">
            {!fullDescription
                ? descriptionPreview.text
                : formatHTMLAsText(activity.description)}
            {#if descriptionPreview.truncated && !fullDescription}
                <button
                    onclick={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                        fullDescription = true;
                    }}
                >
                    ... <span class="underline">{$_("read-more")}</span></button
                >
            {/if}
        </p>
    {/if}
</div>
