<script lang="ts">
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import type { List } from "$lib/models/list";
    import { theme } from "$lib/stores/theme_store";
    import { getFileURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import ShareInfo from "../share_info.svelte";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";
    import { currentUser } from "$lib/stores/user_store";

    interface Props {
        list: List;
        active?: boolean;
    }

    let { list, active = false }: Props = $props();

    let listIsShared = $derived(
        (list.expand?.list_share_via_list?.length ?? 0) > 0,
    );
</script>

<div
    class="flex items-start gap-6 p-4 hover:bg-menu-item-background-hover rounded-xl transition-colors cursor-pointer"
    class:bg-menu-item-background-hover={active}
>
    <img
        class="w-16 md:w-20 aspect-square rounded-full object-cover"
        src={list.avatar
            ? getFileURL(list, list.avatar)
            : $theme === "light"
              ? emptyStateTrailLight
              : emptyStateTrailDark}
        alt="list avatar"
    />

    <div class="self-start min-w-0 basis-full transition-transform">
        <div class="flex items-center gap-3">
            <h5 class="text-xl font-semibold overflow-hidden overflow-ellipsis shrink-0 max-w-3/4 whitespace-nowrap">
                {list.name}
            </h5>
            <div class="basis-full"></div>
            {#if $currentUser}
                {#if list.public}
                    <span class="tooltip" data-title={$_("public")}>
                        <i class="fa fa-globe"></i>
                    </span>
                {/if}
                {#if list.expand?.list_share_via_list?.length}
                    <span class="tooltip" data-title={$_("shared")}>
                        <i class="fa fa-share-nodes"></i>
                    </span>
                {/if}
            {/if}
        </div>
        {#if list.expand?.author}
            <p class="text-xs text-gray-500 my-2">
                {$_("by")}
                <img
                    class="rounded-full w-5 aspect-square mx-1 inline"
                    src={list.expand.author.icon ||
                        `https://api.dicebear.com/7.x/initials/svg?seed=${list.expand.author.preferred_username}&backgroundType=gradientLinear`}
                    alt="author avatar"
                />
                {handleFromRecordWithIRI(list)}
            </p>
        {/if}
        <div
            class="grid grid-cols-2 mt-1 mb-2 gap-x-4 gap-y-1 text-sm text-gray-500 whitespace-nowrap flex-wrap"
        >
            <span
                ><i class="fa fa-left-right mr-2"></i>{formatDistance(
                    list.distance,
                )}</span
            >
            <span
                ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                    list.duration,
                )}</span
            >
            <span
                ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(
                    list.elevation_gain,
                )}</span
            >
            <span
                ><i class="fa fa-arrow-trend-down mr-2"></i>{formatElevation(
                    list.elevation_loss,
                )}</span
            >
        </div>
        <p class="text-sm text-gray-500 mb-2">
            {list.trails?.length ?? 0}
            {$_("trail", {
                values: { n: list.expand?.trails?.length ?? 0 },
            })}
        </p>
        <p
            class="text-gray-500 text-sm mr-8 whitespace-pre-wrap {active
                ? ''
                : 'max-h-24 overflow-hidden text-ellipsis'}"
        >
            {formatHTMLAsText(
                !active
                    ? list.description?.substring(0, 100)
                    : list.description,
            )}
            {#if (list.description?.length ?? 0) > 100 && !active}
                ...
            {/if}
        </p>
    </div>
</div>
