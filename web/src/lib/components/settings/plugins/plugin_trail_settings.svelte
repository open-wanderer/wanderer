<script lang="ts">
    import Select, { type SelectItem } from "$lib/components/base/select.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        hasTourKindChoice?: boolean;
        supportsSourcePrivacy?: boolean;
        planned?: boolean;
        completed?: boolean;
        privacy?: string;
        privacySelectItems?: SelectItem[];
    }

    let {
        hasTourKindChoice = false,
        supportsSourcePrivacy = false,
        planned = $bindable(true),
        completed = $bindable(true),
        privacy = $bindable("original"),
        privacySelectItems = [],
    }: Props = $props();
</script>

{#if hasTourKindChoice}
    <div class="flex flex-wrap gap-x-4">
        <Toggle
            bind:value={planned}
            label={$_("planned-tours", { values: { n: 2 } })}
        ></Toggle>
        <Toggle
            bind:value={completed}
            label={$_("completed-tours", { values: { n: 2 } })}
        ></Toggle>
    </div>
{/if}

{#if supportsSourcePrivacy}
    <Select
        label={$_("privacy")}
        items={privacySelectItems}
        bind:value={privacy}
    ></Select>
    <p class="text-xs text-gray-500 max-w-lg">
        {#if privacy == "original"}
            {$_("plugin-privacy-hint-original")}
        {:else}
            {$_("plugin-privacy-hint-user")}
        {/if}
    </p>
{/if}
