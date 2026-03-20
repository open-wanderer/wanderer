<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        title?: string;
        onmerge?: (settings: MergeSettings) => void;
    }

    let { title = $_("link-as-summit-log"), onmerge: onmerge }: Props = $props();

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    export interface MergeSettings {
        summitLog: boolean;
        photos: boolean;
        comments: boolean;
        delete: boolean;
        tags: boolean;
        likes: boolean;
    }
    
    const settings: MergeSettings = $state({
        summitLog: true,
        photos: true,
        comments: true,
        delete: true,
        tags: true,
        likes: true,
    });

    function mergeTrail() {
        onmerge?.(settings);
        modal.closeModal();
    }
</script>

<Modal id="merge-modal" {title} size="min-w-md" bind:this={modal}>
    {#snippet content()}
        <div>
            <h4 class="font-semibold mb-2">{$_("copy-include-elements")}</h4>
            <div class="mb-2">
                <input
                    id="include-summit-log-checkbox"
                    type="checkbox"
                    bind:checked={settings.summitLog}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-summit-log-checkbox" class="ms-2 text-sm"
                    >{$_("summit-log", { values: { n: 2 } })}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-photos-checkbox"
                    type="checkbox"
                    bind:checked={settings.photos}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-photos-checkbox" class="ms-2 text-sm"
                    >{$_("photos")}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-comments-checkbox"
                    type="checkbox"
                    bind:checked={settings.comments}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-comments-checkbox" class="ms-2 text-sm"
                    >{$_("comment", { values: { n: 2 } })}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-tags-checkbox"
                    type="checkbox"
                    bind:checked={settings.tags}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-tags-checkbox" class="ms-2 text-sm"
                    >{$_("tags")}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id="include-likes-checkbox"
                    type="checkbox"
                    bind:checked={settings.likes}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-likes-checkbox" class="ms-2 text-sm"
                    >{$_("likes")}</label
                >
            </div>
            <h4 class="font-semibold mt-4 mb-2">{$_("linked-trails")}</h4>
            <div class="mb-2">
                <input
                    id="include-trail-delete-checkbox"
                    type="checkbox"
                    bind:checked={settings.delete}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for="include-trail-delete-checkbox" class="ms-2 text-sm"
                    >{$_("delete-linked-trails")}</label
                >
            </div>
        </div>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
            <button
                class="btn-primary"
                type="button"
                onclick={mergeTrail}
                name="save">{$_("link")}</button
            >
        </div>
    {/snippet}</Modal
>
