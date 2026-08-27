<script lang="ts">
    import PhotoMetadataPanel from "$lib/components/settings/maintenance/photo_metadata_panel.svelte";
    import { _ } from "svelte-i18n";
</script>

<svelte:head>
    <title>{$_("photo-metadata-maintenance-title")} | wanderer</title>
</svelte:head>

<div class="space-y-6">
    <div class="space-y-2">
        <h1 class="text-3xl font-bold">{$_("photo-metadata-maintenance-title")}</h1>
        <p class="max-w-3xl text-sm text-gray-500">
            {$_("photo-metadata-maintenance-description")}
        </p>
    </div>

    <PhotoMetadataPanel />
</div>
