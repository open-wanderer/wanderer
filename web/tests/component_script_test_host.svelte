<script lang="ts">
    import { untrack } from "svelte";
    let { setup }: { setup: () => void } = $props();
    untrack(() => setup());
</script>
