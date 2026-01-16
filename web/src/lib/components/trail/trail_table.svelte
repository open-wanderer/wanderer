<script lang="ts">
    import type { Trail, TrailFilter } from "$lib/models/trail";
    import {
        formatDistance,
        formatElevation,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import { _ } from "svelte-i18n";
    import type { SelectItem } from "../base/select.svelte";
    import { goto } from "$app/navigation";
    import { getFileURL } from "$lib/util/file_util";
    import ShareInfo from "../share_info.svelte";
    
    interface Props {
        tableHeader: SelectItem[];
        trails?: Trail[] | null;
        selection: Set<Trail> | undefined;
        filter?: TrailFilter | null;
        items: number;
        onsort?: (value: any) => void
        onTrailSelect?: (value: any) => void
    }

    let { tableHeader, trails = null, selection, filter = null, items, onsort, onTrailSelect: onselect }: Props = $props();

    function getColumnWidth(columnValue: string): string {
        switch (columnValue) {
            case "select":
                return "w-[2%]";
            case "name":
                return "w-[25%]";
            case "distance":
                return "w-[15%]";
            case "duration":
            case "difficulty":
                return "w-[11%]";
            case "elevation_gain":
            case "created":
            case "date":
                return "w-[5%]";
            default:
                return "";
        }
    }

    function setSelectedTrail(e: Event, trail: Trail) {
        e.stopPropagation()

        if (trail !== undefined) {
            if (onselect !== undefined) {
                onselect(trail)
            }
            else {
                console.error("undefined event handler")
            }   
        }
    }

    function setSelectedAllTrails(e: Event) {
        onselect?.(undefined)
    }

    function isSelected(trail: Trail): boolean {
        if (selection === undefined) {
            return false;
        }
        
        if (trail !== undefined) {
            for (const strail of selection) {
                if (strail !== undefined && strail.id === trail.id)
                    return true;
            }
        }

        return false;
    }

    function allSelected(): boolean {
        if (selection === undefined || trails === undefined || trails === null) {
            return false;
        }

        return selection.size === trails.length;
    }

</script>

<div
    class="table-container w-full border border-input-border rounded-xl overflow-x-auto overflow-y-clip"
>
    <table class="w-full">
        <thead>
            <tr class="bg-secondary-hover">
                <th
                    class="p-4 text-left text-sm font-medium {getColumnWidth("select")}"
                >
                    <div class="flex items-center">
                        <input
                            id="trail-selected"
                            type="checkbox"
                            class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                            onclick={setSelectedAllTrails}
                            checked={allSelected()}
                        />
                    </div>
                </th>
                {#each tableHeader as column}
                    <th
                        class="p-4 text-left text-sm font-medium {getColumnWidth(
                            column.value,
                        )}"
                        onclick={() => onsort?.(column.value)}
                    >
                        <div class="cursor-pointer">
                            {column.text}
                            {#if filter && filter.sort === column.value}
                                <i
                                    id="sort-order-btn"
                                    class="fa fa-arrow-up"
                                    class:rotated={filter.sortOrder == "-"}
                                ></i>
                            {/if}
                        </div>
                    </th>
                {/each}
            </tr>
        </thead>
        <tbody>
            {#if trails}
                {#each trails as trail}
                    <tr
                        class="border-t border-input-border cursor-pointer hover:bg-secondary-hover transition-colors"
                        onclick={() =>
                            goto(
                                `/trail/view/@${trail.author}${
                                    trail.domain ? `@${trail.domain}` : ""
                                }/${trail.id}`,
                            )}
                    >
                        <td class="p-4 text-sm">
                            <div class="flex items-center">
                                <input
                                    type="checkbox"
                                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                                    checked={isSelected(trail)}
                                    onclick={(e: Event) => setSelectedTrail(e, trail)}
                                />
                            </div>
                        </td>
                        <td
                            class="flex justify-between items-center text-sm relative"
                        >
                            <div class="p-4 w-[75%] line-clamp-2 wrap-anywhere" title={trail.name}>
                                {trail.name}
                            </div>
                            <div class="flex flex-col items-center">
                                {#if trail.expand && trail.expand.author}
                                    <div class="author-icon">
                                        <img
                                            title={`${trail.public ? $_("public") + " " : ""}${$_("by")} @${trail.expand.author.preferred_username}${trail.expand.author.isLocal ? "" : "@" + trail.expand.author.domain}`}
                                            class="rounded-full w-5 aspect-square mx-1 inline"
                                            src={trail.expand.author.icon ||
                                                `https://api.dicebear.com/7.x/initials/svg?seed=${trail.expand.author.preferred_username}&backgroundType=gradientLinear`}
                                            alt="avatar"
                                        />
                                    </div>
                                {/if}
                                <div class="flex gap-x-1">
                                    {#if trail.public}
                                        <div
                                            class="public-icon tooltip"
                                            data-title={$_("public")}
                                        >
                                            <i class="fa fa-globe"></i>
                                        </div>
                                    {/if}
                                    {#if trail.expand?.trail_share_via_trail?.length}
                                        <ShareInfo type="trail" subject={trail}
                                        ></ShareInfo>
                                    {/if}
                                </div>
                            </div>
                        </td>
                        <td class="p-4 text-sm">
                            {formatDistance(trail.distance)}
                        </td>
                        <td class="p-4 text-sm">
                            {formatTimeHHMM(trail.duration)}
                        </td>
                        <td class="p-4 text-sm">
                            {$_(trail.difficulty ?? "easy")}
                        </td>
                        <td class="p-4 text-sm">
                            {formatElevation(trail.elevation_gain)}
                        </td>
                        <td class="p-4 text-sm">
                            {trail.like_count}
                        </td>
                        <td class="p-4 text-sm">
                            {#if trail.created}
                                {new Date(trail.created).toLocaleDateString(
                                    undefined,
                                    {
                                        month: "2-digit",
                                        day: "2-digit",
                                        year: "numeric",
                                        timeZone: "UTC",
                                    },
                                )}
                            {/if}
                        </td>
                        <td class="p-4 text-sm">
                            {#if trail.date}
                                {new Date(trail.date).toLocaleDateString(
                                    undefined,
                                    {
                                        month: "2-digit",
                                        day: "2-digit",
                                        year: "numeric",
                                        timeZone: "UTC",
                                    },
                                )}
                            {/if}
                        </td>
                    </tr>
                {/each}
            {:else}
                {#each { length: items } as _, index}
                    <tr
                        class="border-t border-input-border cursor-pointer bg-secondary-hover transition-colors animate-pulse"
                    >
                        {#each { length: tableHeader.length } as _, index}
                            <td class="p-4 text-sm"
                                ><div
                                    class="h-4 bg-menu-item-background-focus rounded w-3/4"
                                ></div></td
                            >
                        {/each}
                    </tr>
                {/each}
            {/if}
        </tbody>
    </table>
</div>

<style>
    .table-container {
        container-type: inline-size;
    }

    @container (max-width: 660px) {
        th:nth-last-child(-n + 2),
        td:nth-last-child(-n + 2) {
            display: none;
        }
    }
</style>
