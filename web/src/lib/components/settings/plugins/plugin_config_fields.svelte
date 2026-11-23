<script lang="ts">
    import Datepicker from "#lib/components/base/datepicker.svelte";
    import Select, { type SelectItem } from "#lib/components/base/select.svelte";
    import TextField from "#lib/components/base/text_field.svelte";
    import Toggle from "#lib/components/base/toggle.svelte";
    import type { ConfigField } from "#lib/models/plugin_provider.ts";
    import { _ } from "svelte-i18n";

    interface Props {
        fields?: ConfigField[];
        config?: Record<string, any>;
        fieldLabel: (field: ConfigField) => string;
        fieldHint: (field: ConfigField) => string | undefined;
        fieldError: (field: ConfigField) => string;
        selectItems: (field: ConfigField) => SelectItem[];
    }

    let {
        fields = [],
        config = $bindable<Record<string, any>>({}),
        fieldLabel,
        fieldHint,
        fieldError,
        selectItems,
    }: Props = $props();
</script>

{#each fields as field}
    {#if field.type === "select"}
        <Select
            label={fieldLabel(field)}
            items={selectItems(field)}
            bind:value={config[field.key] as string}
            error={fieldError(field)}
        ></Select>
    {:else if field.type === "boolean"}
        <Toggle
            bind:value={config[field.key]}
            label={fieldLabel(field)}
            error={fieldError(field)}
        ></Toggle>
        {@const hint = fieldHint(field)}
        {#if hint}
            <p class="text-xs text-gray-500 max-w-lg">{hint}</p>
        {/if}
    {:else if field.type === "date"}
        {@const hint = fieldHint(field)}
        {#if hint}
            <p
                class="text-xs text-gray-500 max-w-lg pt-4 pb-1 border-t border-input-border"
            >
                {hint}
            </p>
        {/if}
        <div class="flex items-end relative gap-x-2">
            <Datepicker
                label={fieldLabel(field)}
                bind:value={config[field.key]}
                error={fieldError(field)}
            ></Datepicker>
            <button
                class="btn-icon mb-[10px]"
                type="button"
                onclick={() => {
                    config[field.key] = undefined;
                }}
                aria-label={$_("clear")}
            ><i class="fa fa-close"></i></button>
        </div>
    {:else if field.type === "text" || field.type === "url"}
        <TextField
            label={fieldLabel(field)}
            bind:value={config[field.key]}
            name={field.key}
            type={field.type === "url" ? "url" : "text"}
            error={fieldError(field)}
        ></TextField>
    {/if}
{/each}
