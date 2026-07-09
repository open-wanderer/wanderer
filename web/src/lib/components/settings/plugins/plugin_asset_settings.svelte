<script lang="ts">
    import Select, { type SelectItem } from "$lib/components/base/select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import type { ConfigField } from "$lib/models/plugin_provider";
    import { _ } from "svelte-i18n";

    interface Props {
        photoMode?: string;
        photoModeSelectItems?: SelectItem[];
        importSizeField?: ConfigField;
        maxWaypointsField?: ConfigField;
        supportsPhotoLimits?: boolean;
        supportsMaxPhotosPerTrail?: boolean;
        maxPhotosPerTrail?: string | number;
        maxPhotosPerWaypoint?: string | number;
        autoAttachTrailPlugins?: boolean;
        autoAttachUpload?: boolean;
        config?: Record<string, any>;
        fieldLabel: (field: ConfigField) => string;
        fieldError: (field: ConfigField) => string;
        selectItems: (field: ConfigField) => SelectItem[];
    }

    let {
        photoMode = $bindable("copy"),
        photoModeSelectItems = [],
        importSizeField,
        maxWaypointsField,
        supportsPhotoLimits = false,
        supportsMaxPhotosPerTrail = false,
        maxPhotosPerTrail = $bindable("20"),
        maxPhotosPerWaypoint = $bindable("5"),
        autoAttachTrailPlugins = $bindable(true),
        autoAttachUpload = $bindable(true),
        config = $bindable<Record<string, any>>({}),
        fieldLabel,
        fieldError,
        selectItems,
    }: Props = $props();
</script>

<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
    <Select
        label={$_("plugin-photo-mode-label")}
        items={photoModeSelectItems}
        bind:value={photoMode}
    ></Select>
    {#if importSizeField && importSizeField.type === "select"}
        <Select
            label={fieldLabel(importSizeField)}
            items={selectItems(importSizeField)}
            bind:value={config[importSizeField.key] as string}
            error={fieldError(importSizeField)}
        ></Select>
    {/if}
</div>
<p class="text-xs text-gray-500">
    {$_("plugin-photo-mode-link-private-hint")}
</p>

<div class="space-y-2">
    <p class="text-sm font-medium">{$_("plugin-auto-attach-label")}</p>
    <div class="flex flex-wrap gap-x-4">
        <Toggle
            bind:value={autoAttachTrailPlugins}
            label={$_("plugin-auto-attach-trail-plugins")}
        ></Toggle>
        <Toggle
            bind:value={autoAttachUpload}
            label={$_("plugin-auto-attach-upload")}
        ></Toggle>
    </div>
</div>

{#if supportsPhotoLimits}
    <div class="space-y-3 pt-4 border-t border-input-border">
        <h4 class="text-sm font-medium">{$_("plugin-photo-limits-label")}</h4>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            {#if maxWaypointsField}
                <TextField
                    label={fieldLabel(maxWaypointsField)}
                    bind:value={config[maxWaypointsField.key]}
                    name={maxWaypointsField.key}
                    type="number"
                    min={maxWaypointsField.min}
                    max={maxWaypointsField.max}
                    step={maxWaypointsField.step}
                    error={fieldError(maxWaypointsField)}
                ></TextField>
            {/if}
            <TextField
                label={$_("plugin-max-photos-per-waypoint")}
                bind:value={maxPhotosPerWaypoint}
                name="maxPhotosPerWaypoint"
                type="number"
                min="1"
                step="1"
            ></TextField>
            {#if supportsMaxPhotosPerTrail}
                <TextField
                    label={$_("plugin-max-photos-per-trail")}
                    bind:value={maxPhotosPerTrail}
                    name="maxPhotosPerTrail"
                    type="number"
                    min="1"
                    step="1"
                ></TextField>
            {/if}
        </div>
    </div>
{/if}
