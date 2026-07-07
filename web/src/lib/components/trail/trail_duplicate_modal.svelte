<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import {
        defaultTrailDuplicateOptions,
        normalizeTrailDuplicateOptions,
        type Trail,
        type TrailDuplicateOptions,
    } from "$lib/models/trail";
    import { _ } from "svelte-i18n";

    interface Props {
        onduplicate?: (settings: TrailDuplicateOptions) => void;
    }

    let { onduplicate }: Props = $props();

    let modal: Modal;
    let sourceTrail: Trail | undefined = $state();
    let settings: TrailDuplicateOptions = $state({
        ...defaultTrailDuplicateOptions,
    });

    function duplicateSettings(includePhotosByDefault: boolean): TrailDuplicateOptions {
        return normalizeTrailDuplicateOptions({
            ...defaultTrailDuplicateOptions,
            trailPhotos: includePhotosByDefault,
            waypointPhotos: includePhotosByDefault,
            summitLogPhotos: false,
        });
    }

    export function openModal(trail: Trail, includePhotosByDefault = false) {
        sourceTrail = trail;
        settings = duplicateSettings(includePhotosByDefault);
        modal.openModal();
    }

    function duplicateTrail() {
        onduplicate?.({ ...settings });
        modal.closeModal();
    }

    $effect(() => {
        if (!settings.waypoints) {
            settings.waypointPhotos = false;
        }
        if (!settings.summitLogs) {
            settings.summitLogPhotos = false;
        }
    });
</script>

<Modal id="trail-duplicate-modal" title={$_("duplicate")} size="min-w-md" bind:this={modal}>
    {#snippet content()}
        <div>
            {#if sourceTrail?.name}
                <p class="mb-4 text-sm font-medium break-words">{sourceTrail.name}</p>
            {/if}

            <div
                class="grid items-center gap-x-4 gap-y-2 text-sm"
                style="grid-template-columns: minmax(0, 1fr) min-content min-content;"
            >
                <span></span>
                <span class="justify-self-center text-sm font-medium whitespace-nowrap"
                    >{$_("copy")}</span
                >
                <span class="justify-self-center text-sm font-medium whitespace-nowrap"
                    >{$_("include-photos")}</span
                >

                <p class="min-w-0">{$_("duplicate-route-and-metadata")}</p>
                <div class="justify-self-center">
                    <Toggle value={true} disabled></Toggle>
                </div>
                <div class="justify-self-center">
                    <Toggle bind:value={settings.trailPhotos}></Toggle>
                </div>

                <p class="min-w-0">{$_("duplicate-trail-waypoints")}</p>
                <div class="justify-self-center">
                    <Toggle bind:value={settings.waypoints}></Toggle>
                </div>
                <div class="justify-self-center" class:opacity-50={!settings.waypoints}>
                    <Toggle bind:value={settings.waypointPhotos} disabled={!settings.waypoints}
                    ></Toggle>
                </div>

                <p class="min-w-0">{$_("duplicate-trail-summit-logs")}</p>
                <div class="justify-self-center">
                    <Toggle bind:value={settings.summitLogs}></Toggle>
                </div>
                <div class="justify-self-center" class:opacity-50={!settings.summitLogs}>
                    <Toggle bind:value={settings.summitLogPhotos} disabled={!settings.summitLogs}
                    ></Toggle>
                </div>
            </div>
        </div>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center justify-end gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
            <button class="btn-primary" type="button" onclick={duplicateTrail}
                >{$_("duplicate")}</button
            >
        </div>
    {/snippet}
</Modal>
