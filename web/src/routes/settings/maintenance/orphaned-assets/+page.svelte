<script lang="ts">
    import OrphanedAssetsPanel from "$lib/components/settings/maintenance/orphaned_assets_panel.svelte";
    import { _ } from "svelte-i18n";
</script>

<svelte:head>
    <title>{$_("orphaned-assets-maintenance-title")} | wanderer</title>
</svelte:head>

<div class="space-y-6">
    <div class="space-y-2">
        <h1 class="text-3xl font-bold">{$_("orphaned-assets-maintenance-title")}</h1>
        <p class="max-w-3xl text-sm text-gray-500">
            {$_("orphaned-assets-maintenance-description")}
        </p>
    </div>

    <OrphanedAssetsPanel />
</div>
