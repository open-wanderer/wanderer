<script lang="ts">
    import type { PhotoLibraryCandidate } from "$lib/models/photo_library";
    import type { PluginProvider } from "$lib/models/plugin_provider";
    import PhotoLibraryPickerModal from "../photo/photo_library_picker_modal.svelte";

    interface Props {
        lat: number;
        lon: number;
        assetPluginIds?: string[];
        assetPluginProviders?: PluginProvider[];
        onselect?: (candidates: PhotoLibraryCandidate[]) => void | Promise<void>;
    }

    let { lat, lon, assetPluginIds = [], assetPluginProviders = [], onselect }: Props = $props();
    let picker: PhotoLibraryPickerModal = $state()!;

    export function openModal() {
        picker.openModal();
    }
</script>

<PhotoLibraryPickerModal
    bind:this={picker}
    id="asset-photo-picker-modal"
    {lat}
    {lon}
    {assetPluginIds}
    {assetPluginProviders}
    doubleRadius={false}
    {onselect}
/>
