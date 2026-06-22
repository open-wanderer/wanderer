<script lang="ts">
    import Toggle from "$lib/components/base/toggle.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        prefix: string;
        value?: boolean;
    }

    let { prefix, value = $bindable(false) }: Props = $props();
</script>

<div class="pt-4 border-t border-input-border space-y-2">
    <Toggle
        name={`${prefix}.enabled`}
        bind:value
        label={$_("plugin-auto-merge-label")}
    />
    <p class="text-xs text-gray-500 max-w-lg">
        {$_("plugin-auto-merge-hint")}
    </p>
</div>
