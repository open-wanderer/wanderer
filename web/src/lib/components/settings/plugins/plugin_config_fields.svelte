<script lang="ts">
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import Select, { type SelectItem } from "$lib/components/base/select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import type { ConfigField } from "$lib/models/plugin_provider";
    import { _ } from "svelte-i18n";

    interface Props {
        fields?: ConfigField[];
        config?: Record<string, any>;
        fieldLabel: (field: ConfigField) => string;
        fieldHint: (field: ConfigField) => string | undefined;
        fieldError: (field: ConfigField) => string;
        selectItems: (field: ConfigField) => SelectItem[];
        bordered?: boolean;
        columns?: 1 | 2;
    }

    let {
        fields = [],
        config = $bindable<Record<string, any>>({}),
        fieldLabel,
        fieldHint,
        fieldError,
        selectItems,
        bordered = true,
        columns = 2,
    }: Props = $props();
</script>

{#if fields.length > 0}
    <div
        class={`grid grid-cols-1 gap-3 ${columns === 2 ? "md:grid-cols-2" : ""} ${bordered ? "pt-4 border-t border-input-border" : ""}`}
    >
        {#each fields as field}
            <div class={field.type === "boolean" ? "md:col-span-2" : ""}>
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
                        <p class="text-xs text-gray-500 max-w-lg">{hint}</p>
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
                {:else if field.type === "number" || field.type === "text" || field.type === "url"}
                    <TextField
                        label={fieldLabel(field)}
                        bind:value={config[field.key]}
                        name={field.key}
                        type={field.type === "number" ? "number" : field.type === "url" ? "url" : "text"}
                        min={field.min}
                        max={field.max}
                        step={field.step}
                        error={fieldError(field)}
                    ></TextField>
                {/if}
            </div>
        {/each}
    </div>
{/if}
