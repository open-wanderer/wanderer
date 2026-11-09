<script lang="ts">
    import { _ } from "svelte-i18n";

    let { progress }: { progress?: { file: string; current: number; total: number } } = $props();
</script>

<p role="status" class="mt-2 text-sm text-gray-500">
    {#if progress}{$_("photo-conversion-progress", { values: progress })}{/if}
</p>
