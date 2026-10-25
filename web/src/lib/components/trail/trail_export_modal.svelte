<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";
    import Select from "../base/select.svelte";

    interface Props {
        title?: string;
        onexport?: (settings: typeof exportSettings) => void;
    }

    let { title = $_("export"), onexport }: Props = $props();

    const uid = $props.id();
    const modalId = `export-modal-${uid}`;

    let modal: Modal;

    export function openModal() {
        modal.openModal();
    }

    const fileFormats = [
        { text: "GPX", value: "gpx" },
        { text: "GeoJSON", value: "json" },
    ];

    const exportSettings: {
        fileFormat: "gpx" | "json";
        photos: boolean;
        summitLog: boolean;
    } = $state({
        fileFormat: "gpx",
        photos: false,
        summitLog: false,
    });

    function exportTrail() {
        onexport?.(exportSettings);
        modal.closeModal();
    }
</script>

<Modal id={modalId} {title} size="md:min-w-sm" bind:this={modal}>
    {#snippet content()}
        <div>
            <div class="mb-3">
                <Select
                    bind:value={exportSettings.fileFormat}
                    items={fileFormats}
                    label={$_("file-format")}
                ></Select>
            </div>
            <div class="mb-2">
                <input
                    id={`${modalId}-include-photos-checkbox`}
                    type="checkbox"
                    bind:checked={exportSettings.photos}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for={`${modalId}-include-photos-checkbox`} class="ms-2 text-sm"
                    >{$_("photos")}</label
                >
            </div>
            <div class="mb-2">
                <input
                    id={`${modalId}-include-summit-log-checkbox`}
                    type="checkbox"
                    bind:checked={exportSettings.summitLog}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                />
                <label for={`${modalId}-include-summit-log-checkbox`} class="ms-2 text-sm"
                    >{$_("summit-book")}</label
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
                onclick={exportTrail}
                name="save">{$_("export")}</button
            >
        </div>
    {/snippet}</Modal
>
