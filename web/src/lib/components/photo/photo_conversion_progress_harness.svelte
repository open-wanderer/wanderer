<script lang="ts">
    import PhotoConversionProgress from "./photo_conversion_progress.svelte";

    let progress = $state<{ file: string; current: number; total: number } | undefined>();
    export function setProgress(value: typeof progress) { progress = value; }
</script>

<PhotoConversionProgress {progress} />
