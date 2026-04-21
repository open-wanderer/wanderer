<script lang="ts">
    import Toggle from "$lib/components/base/toggle.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        prefix: string;
    }

    let { prefix }: Props = $props();
</script>

<div class="pt-4 border-t border-input-border space-y-2">
    <Toggle
        name={`${prefix}.enabled`}
        label={$_("integration-auto-merge-label")}
    />
    <p class="text-xs text-gray-500 max-w-lg">
        {$_("integration-auto-merge-hint")}
    </p>
</div>
