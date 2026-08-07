<script lang="ts">
    import emptyStateTrailDark from "$lib/assets/svgs/empty_states/empty_state_trail_dark.svg";
    import emptyStateTrailLight from "$lib/assets/svgs/empty_states/empty_state_trail_light.svg";
    import type { List } from "$lib/models/list";
    import type { Trail } from "$lib/models/trail";
    import { theme } from "$lib/stores/theme_store";
    import { currentUser } from "$lib/stores/user_store";
    import { getFileURL } from "$lib/util/file_util";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsTextPreview,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import Dropdown, { type DropdownItem } from "../base/dropdown.svelte";
    import ShareInfo from "../share_info.svelte";
    import TrailListItem from "../trail/trail_list_item.svelte";
    import { handleFromRecordWithIRI } from "$lib/util/activitypub_util";

    const DESCRIPTION_PREVIEW_LENGTH = 100;

    interface Props {
        list: List;
        onclick?: (data: { trail: Trail; index: number }) => void;
        onmouseenter?: (data: { trail: Trail; index: number }) => void;
        onmouseleave?: (data: { trail: Trail; index: number }) => void;
        onchange?: (item: DropdownItem) => void;
    }

    let { list, onclick, onmouseenter, onmouseleave, onchange }: Props =
        $props();

    let cumulativeDistance = $derived(
        list.expand?.trails?.reduce((s, b) => s + b.distance!, 0),
    );

    let cumulativeElevationGain = $derived(
        list.expand?.trails?.reduce((s, b) => s + b.elevation_gain!, 0),
    );

    let cumulativeElevationLoss = $derived(
        list.expand?.trails?.reduce((s, b) => s + b.elevation_loss!, 0),
    );

    let cumulativeDuration = $derived(
        list.expand?.trails?.reduce((s, b) => s + b.duration!, 0),
    );

    let allowEdit = $derived(
        list.author == $currentUser?.actor ||
            list.expand?.list_share_via_list?.some(
                (s) => s.permission == "edit",
            ),
    );

    let allowShare = $derived(
        list.author == $currentUser?.actor &&
            !list.expand?.trails?.some(
                (t) => !t.public && t.author !== $currentUser?.actor,
            ),
    );

    let dropdownItems = $derived([
        ...(allowShare
            ? [{ text: $_("share"), value: "share", icon: "share" }]
            : []),
        ...(allowEdit
            ? [{ text: $_("edit"), value: "edit", icon: "pen" }]
            : []),
        ...(list.author == $currentUser?.actor
            ? [{ text: $_("delete"), value: "delete", icon: "trash" }]
            : []),
    ]);

    let listIsShared = $derived(
        (list.expand?.list_share_via_list?.length ?? 0) > 0,
    );

    let fullDescription: boolean = $state(false);

    let descriptionPreview = $derived(
        formatHTMLAsTextPreview(list.description, DESCRIPTION_PREVIEW_LENGTH),
    );

    function handleTrailSelect(trail: Trail, index: number) {
        onclick?.({ trail, index });
    }

    function handleTrailMouseEnter(trail: Trail, index: number) {
        onmouseenter?.({ trail, index });
    }

    function handleTrailMouseLeave(trail: Trail, index: number) {
        onmouseleave?.({ trail, index });
    }
</script>

<div class="relative">
    {#if (list.public || listIsShared) && $currentUser}
        <div
            class="flex absolute top-4 left-6 {list.public && listIsShared
                ? 'w-16'
                : 'w-8'} h-8 rounded-full items-center justify-center bg-white text-primary"
        >
            {#if list.public}
                <span
                    class="tooltip"
                    class:mr-3={list.public && listIsShared}
                    data-title={$_("public")}
                >
                    <i class="fa fa-globe"></i>
                </span>
            {/if}
            {#if listIsShared}
                <ShareInfo type="list" subject={list}></ShareInfo>
            {/if}
        </div>
    {/if}
    {#if dropdownItems.length}
        <div class="absolute bottom-8 right-8">
            <Dropdown items={dropdownItems} {onchange}
                >{#snippet children({ toggleMenu: openDropdown })}
                    <button
                        aria-label="Open dropdown"
                        class="rounded-full bg-white text-black hover:bg-gray-200 focus:ring-4 ring-gray-100/50 transition-colors h-12 w-12"
                        onclick={openDropdown}
                    >
                        <i class="fa fa-ellipsis-vertical"></i>
                    </button>
                {/snippet}
            </Dropdown>
        </div>
    {/if}
    <img
        class="w-full object-cover max-h-64"
        src={list.avatar
            ? getFileURL(list, list.avatar)
            : $theme === "light"
              ? emptyStateTrailLight
              : emptyStateTrailDark}
        alt="avatar"
    />
</div>

<div class="p-4 md:p-6">
    <h4 class="text-2xl font-semibold mb-4">{list.name}</h4>
    {#if list.expand?.author}
        <p class="my-3 text-gray-500 text-sm">
            {$_("by")}
            <img
                class="rounded-full w-8 aspect-square mx-1 inline"
                src={list.expand.author.icon ||
                    `https://api.dicebear.com/7.x/initials/svg?seed=${list.expand.author.preferred_username}&backgroundType=gradientLinear`}
                alt="avatar"
            />
            <a class="underline" href="/profile/{handleFromRecordWithIRI(list)}"
                >{handleFromRecordWithIRI(list)}</a
            >
        </p>
    {/if}
    <hr />
    <div
        class="grid grid-cols-2 my-4 gap-4 font-semibold whitespace-nowrap flex-wrap justify-around"
    >
        <span
            ><i class="fa fa-left-right mr-2"></i>{formatDistance(
                cumulativeDistance,
            )}</span
        >
        <span
            ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                cumulativeDuration,
            )}</span
        >
        <span
            ><i class="fa fa-arrow-trend-up mr-2"></i>{formatElevation(
                cumulativeElevationGain,
            )}</span
        >
        <span
            ><i class="fa fa-arrow-trend-down mr-2"></i>{formatElevation(
                cumulativeElevationLoss,
            )}</span
        >
    </div>
    <hr class="mb-4" />
    {#if list.description}
        <div class="text-gray-500 whitespace-pre-wrap">
            {#if descriptionPreview.truncated && !fullDescription}
                <div>{descriptionPreview.text}</div>
                <button type="button" onclick={() => (fullDescription = true)}>
                    ... <span class="underline">{$_("read-more")}</span></button
                >
            {:else}
                <div class="prose dark:prose-invert">
                    {@html list.description}
                </div>
            {/if}
        </div>
    {/if}
    <h5 class="text-xl font-semibold my-4">
        {list.trails?.length ?? 0}
        {$_("trail", { values: { n: list.trails?.length ?? 0 } })}
    </h5>
    <div class="space-y-2">
        {#each list.expand?.trails ?? [] as trail, i}
            <div
                role="presentation"
                onclick={() => handleTrailSelect(trail, i)}
                onmouseenter={() => handleTrailMouseEnter(trail, i)}
                onmouseleave={() => handleTrailMouseLeave(trail, i)}
            >
                <TrailListItem selected={false} hovered={false} {trail} showDescription={false}></TrailListItem>
            </div>
        {/each}
    </div>
</div>
