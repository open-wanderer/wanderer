<script lang="ts">
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import type { SummitLog } from "$lib/models/summit_log";
    import { _ } from "svelte-i18n";
    import Dropdown, { type DropdownItem } from "../base/dropdown.svelte";

    import { theme } from "$lib/stores/theme_store";
    import {
        getFileURL,
        isVideoURL,
        readAsDataURLAsync,
    } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";

    let thumbnail: string = $state(
        $theme === "light" ? emptyStateTrailLight : emptyStateTrailDark,
    );

    interface Props {
        log: SummitLog;
        mode?: "show" | "edit";
        onchange?: (item: DropdownItem) => void;
    }

    let { log, mode = "show", onchange }: Props = $props();

    const dropdownItems: DropdownItem[] = [
        { text: $_("edit"), value: "edit" },
        { text: $_("delete"), value: "delete" },
    ];
    $effect(() => fetchPhotos(log._photos ?? []));

    function fetchPhotos(photos: File[]) {
        Promise.all(
            photos.map(async (f) => {
                return await readAsDataURLAsync(f);
            }),
        ).then((v) => {
            if (log.photos.length) {
                thumbnail = getFileURL(log, log.photos[0], "600x0");
            } else if (v.length) {
                thumbnail = v[0];
            } else {
                thumbnail =
                    $theme === "light"
                        ? emptyStateTrailLight
                        : emptyStateTrailDark;
            }
        });
    }
</script>

<div class="p-4 my-2 border border-input-border rounded-xl">
    <div class="flex items-center gap-x-4">
        <div class="h-24 aspect-square shrink-0 rounded-xl overflow-hidden">
            {#if isVideoURL(thumbnail)}
                <!-- svelte-ignore a11y_media_has_caption -->
                <video
                    controls={false}
                    loop
                    class="object-cover h-full w-full"
                    onmouseenter={(e) => (e.target as any).play()}
                    onmouseleave={(e) => (e.target as any).pause()}
                    src={thumbnail}
                ></video>
            {:else}
                <img
                    id="header-img"
                    class="object-cover h-full w-full"
                    src={thumbnail}
                    alt=""
                />
            {/if}
        </div>
        <div class="basis-full">
            <div
                class="flex justify-between items-center"
                class:mb-2={log.text}
            >
                <h5 class="font-medium mr-2">
                    {new Date(log.date).toLocaleDateString(undefined, {
                        month: "2-digit",
                        day: "2-digit",
                        year: "numeric",
                        timeZone: "UTC",
                    })}
                </h5>

                {#if mode == "edit"}
                    <Dropdown items={dropdownItems} {onchange}></Dropdown>
                {/if}
            </div>
            {#if log.expand?.author}
                <p class="text-xs text-gray-500 mb-3">
                    {$_("by")}
                    <img
                        class="rounded-full w-5 aspect-square mx-1 inline"
                        src={log.expand.author.icon ||
                            `https://api.dicebear.com/7.x/initials/svg?seed=${log.expand.author.preferred_username}&backgroundType=gradientLinear`}
                        alt="avatar"
                    />
                    {handleFromRecordWithIRI(log)}
                </p>
            {/if}
            {#if log.distance || log.elevation_gain || log.elevation_loss || log.duration}
                <div
                    class="flex mt-1 gap-x-4 text-sm text-gray-500 flex-wrap mb-2"
                >
                    <span
                        ><i class="fa fa-left-right mr-2"></i>{formatDistance(
                            log.distance,
                        )}</span
                    >
                    <span
                        ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                            log.duration ? log.duration : undefined,
                        )}</span
                    >
                    <span
                        ><i class="fa fa-arrow-trend-up mr-2"
                        ></i>{formatElevation(log.elevation_gain)}</span
                    >
                    <span
                        ><i class="fa fa-arrow-trend-down mr-2"
                        ></i>{formatElevation(log.elevation_loss)}</span
                    >
                </div>
            {/if}
            <span class="whitespace-pre-wrap">{formatHTMLAsText(log.text)}</span>
        </div>
    </div>
</div>
