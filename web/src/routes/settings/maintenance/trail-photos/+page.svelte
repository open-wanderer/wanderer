<script lang="ts">
    import TrailPhotoMaintenancePanel from "$lib/components/settings/maintenance/trail_photo_maintenance_panel.svelte";
    import { _ } from "svelte-i18n";
</script>

<svelte:head>
    <title>{$_("trail-photo-maintenance-title")} | wanderer</title>
</svelte:head>

<div class="space-y-6">
    <div class="space-y-2">
        <h1 class="text-3xl font-bold">{$_("trail-photo-maintenance-title")}</h1>
        <p class="max-w-3xl text-sm text-gray-500">
            {$_("trail-photo-maintenance-description")}
        </p>
    </div>

    <TrailPhotoMaintenancePanel />
</div>
