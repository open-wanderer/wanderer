<script module lang="ts">
    import type { Waypoint } from "$lib/models/waypoint";

    export type WaypointMerge = {
        incoming: Waypoint;
        existing: Waypoint;
    };

    export type WaypointMergeOptions = {
        photos: boolean;
        title: boolean;
        description: boolean;
        icon: boolean;
    };
</script>

<script lang="ts">
    import { browser } from "$app/environment";
    import { _ } from "svelte-i18n";
    import Modal from "../base/modal.svelte";
    import RadioGroup, { type RadioItem } from "../base/radio_group.svelte";

    interface Props {
        merge?: WaypointMerge;
        oncreate?: () => void;
        onmerge?: (options: WaypointMergeOptions) => void;
        oncancel?: () => void;
    }

    let { merge, oncreate, onmerge, oncancel }: Props = $props();

    const waypointMergeActionItems: RadioItem[] = [
        {
            text: $_("create-waypoint-anyway"),
            value: "create",
        },
        {
            text: $_("add-to-existing-waypoint"),
            value: "merge",
        },
    ];

    let modal: Modal;
    let waypointMergeAction: "merge" | "create" = $state("create");
    let waypointMergeActionIndex = $derived(
        waypointMergeActionItems.findIndex(
            (item) => item.value === waypointMergeAction,
        ),
    );
    let appendWaypointPhotos = $state(true);
    let appendWaypointTitle = $state(true);
    let appendWaypointDescription = $state(true);
    let appendWaypointIcon = $state(false);

    export function openModal() {
        loadWaypointMergePreferences();
        modal.openModal();
    }

    export function closeModal() {
        modal.closeModal();
    }

    function saveDecision() {
        if (waypointMergeAction === "create") {
            oncreate?.();
            return;
        }

        onmerge?.({
            photos: appendWaypointPhotos,
            title: appendWaypointTitle,
            description: appendWaypointDescription,
            icon: appendWaypointIcon,
        });
    }

    function loadWaypointMergePreferences() {
        waypointMergeAction = getWaypointMergePreference("action", false)
            ? "merge"
            : "create";
        appendWaypointPhotos = getWaypointMergePreference("photos", true);
        appendWaypointTitle = getWaypointMergePreference("title", true);
        appendWaypointDescription = getWaypointMergePreference(
            "description",
            true,
        );
        appendWaypointIcon = getWaypointMergePreference("icon", false);
    }

    function getWaypointMergePreference(key: string, fallback: boolean) {
        if (!browser) {
            return fallback;
        }

        const value = localStorage.getItem(`waypoint-merge-${key}`);
        if (value == null) {
            return fallback;
        }

        return value === "true";
    }

    function setWaypointMergePreference(key: string, value: boolean) {
        if (browser) {
            localStorage.setItem(`waypoint-merge-${key}`, value.toString());
        }
    }
</script>

<Modal
    id="waypoint-merge-modal"
    title={$_("nearby-waypoint-found")}
    bind:this={modal}
>
    {#snippet content()}
        {#if merge}
            <div class="space-y-4">
                <div
                    class="flex items-center gap-3 rounded-md border border-input-border p-3"
                >
                    <div
                        class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-surface"
                    >
                        <i class="fa fa-{merge.existing.icon ?? 'circle'}"></i>
                    </div>
                    <div>
                        <p class="font-medium">
                            {merge.existing.name ||
                                $_("waypoints", { values: { n: 1 } })}
                        </p>
                        <p class="text-sm text-gray-500">
                            {merge.existing.lat.toFixed(5)},
                            {merge.existing.lon.toFixed(5)}
                        </p>
                    </div>
                </div>
                <p>
                    {$_("nearby-waypoint-found-text", {
                        values: {
                            name:
                                merge.existing.name ||
                                $_("waypoints", { values: { n: 1 } }),
                        },
                    })}
                </p>
                <RadioGroup
                    name="waypoint-merge-action"
                    items={waypointMergeActionItems}
                    selected={waypointMergeActionIndex}
                    onchange={(item) => {
                        waypointMergeAction = item.value as "merge" | "create";
                        setWaypointMergePreference(
                            "action",
                            waypointMergeAction === "merge",
                        );
                    }}
                ></RadioGroup>
                {#if waypointMergeAction === "merge"}
                    <div class="space-y-2 pl-6">
                        {#if merge.incoming._photos?.length}
                            <label class="flex items-center gap-2">
                                <input
                                    type="checkbox"
                                    bind:checked={appendWaypointPhotos}
                                    onchange={() =>
                                        setWaypointMergePreference(
                                            "photos",
                                            appendWaypointPhotos,
                                        )}
                                />
                                <span>{$_("append-waypoint-photos")}</span>
                            </label>
                        {/if}
                        {#if merge.incoming.name?.trim()}
                            <label class="flex items-center gap-2">
                                <input
                                    type="checkbox"
                                    bind:checked={appendWaypointTitle}
                                    onchange={() =>
                                        setWaypointMergePreference(
                                            "title",
                                            appendWaypointTitle,
                                        )}
                                />
                                <span>
                                    {merge.existing.name?.trim()
                                        ? $_("append-waypoint-title")
                                        : $_("use-waypoint-title")}
                                </span>
                            </label>
                        {/if}
                        {#if merge.incoming.description?.trim()}
                            <label class="flex items-center gap-2">
                                <input
                                    type="checkbox"
                                    bind:checked={appendWaypointDescription}
                                    onchange={() =>
                                        setWaypointMergePreference(
                                            "description",
                                            appendWaypointDescription,
                                        )}
                                />
                                <span>
                                    {merge.existing.description?.trim()
                                        ? $_("append-waypoint-description")
                                        : $_("use-waypoint-description")}
                                </span>
                            </label>
                        {/if}
                        {#if merge.incoming.icon &&
                            merge.incoming.icon !== merge.existing.icon}
                            <label class="flex items-center gap-2">
                                <input
                                    type="checkbox"
                                    bind:checked={appendWaypointIcon}
                                    onchange={() =>
                                        setWaypointMergePreference(
                                            "icon",
                                            appendWaypointIcon,
                                        )}
                                />
                                <span>
                                    {$_("use-new-waypoint-icon")}
                                    <i
                                        class="fa fa-{merge.incoming.icon} ml-1"
                                    ></i>
                                </span>
                            </label>
                        {/if}
                    </div>
                {/if}
            </div>
        {/if}
    {/snippet}
    {#snippet footer()}
        <div class="flex flex-wrap items-center gap-4">
            <button class="btn-secondary" type="button" onclick={oncancel}
                >{$_("cancel")}</button
            >
            <button class="btn-primary" type="button" onclick={saveDecision}
                >{waypointMergeAction === "create"
                    ? $_("save")
                    : $_("continue")}</button
            >
        </div>
    {/snippet}
</Modal>
