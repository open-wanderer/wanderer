<script lang="ts">
    import Slider from "$lib/components/base/slider.svelte";
    import Select from "$lib/components/base/select.svelte";
    import { ROUND_TRIP_COMPASS_DIRECTIONS } from "$lib/util/routing_round_trip_util";
    import { _ } from "svelte-i18n";
    import { slide } from "svelte/transition";

    interface Props {
        loading?: boolean;
        disabled?: boolean;
        onGenerate: (
            targetDistanceMeters: number,
            direction?: number,
        ) => void | Promise<void>;
    }

    let {
        loading = false,
        disabled = false,
        onGenerate,
    }: Props = $props();

    let targetDistanceKm = $state(40);
    let direction = $state("");
    let expanded = $state(false);

    function directionItems() {
        return [
            {
                text: $_("routing-round-trip-direction-auto"),
                value: "",
            },
            ...ROUND_TRIP_COMPASS_DIRECTIONS.map((compassDirection) => ({
                text: $_(compassDirection.labelKey),
                value: `${compassDirection.bearing}`,
            })),
        ];
    }

    async function generate() {
        await onGenerate(
            targetDistanceKm * 1000,
            direction === "" ? undefined : Number(direction),
        );
        expanded = false;
    }
</script>

<section class="w-full">
    <div
        class="flex w-full items-center rounded-lg border border-input-border transition-colors"
        class:rounded-b-none={expanded}
        class:opacity-60={disabled}
    >
        <div class="flex min-w-0 flex-1 items-center gap-3 px-4 py-3">
            <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-input-background text-content"
            >
                <i class="fa fa-arrows-spin" aria-hidden="true"></i>
            </span>
            <span class="min-w-0 flex-1 font-medium">{$_("routing-round-trip")}</span>
        </div>
        <button
            type="button"
            class="btn-icon mr-2 shrink-0"
            aria-controls="round-trip-controls-panel"
            aria-expanded={expanded}
            aria-label={$_("routing-round-trip")}
            {disabled}
            onclick={() => (expanded = !expanded)}
        >
            <i
                class="fa fa-chevron-{expanded ? 'up' : 'down'} text-xs text-gray-500"
                aria-hidden="true"
            ></i>
        </button>
    </div>

    {#if expanded}
        <div
            id="round-trip-controls-panel"
            class="space-y-4 rounded-b-lg border-x border-b border-input-border p-3"
            transition:slide
        >
            <div class="text-sm">
                <div class="flex items-center justify-between gap-3">
                    <span class="font-medium">{$_("routing-round-trip-distance")}</span>
                    <output class="shrink-0 tabular-nums">
                        {Math.round(targetDistanceKm)} km
                    </output>
                </div>
                <Slider
                    minValue={1}
                    maxValue={300}
                    step={1}
                    bind:currentValue={targetDistanceKm}
                ></Slider>
            </div>

            <Select
                name="round-trip-direction"
                label={$_("routing-round-trip-direction")}
                items={directionItems()}
                bind:value={direction}
                {disabled}
            ></Select>

            <button
                class="btn-primary w-full"
                type="button"
                disabled={disabled || loading}
                onclick={generate}
            >
                {#if loading}
                    <i class="fa fa-spinner fa-spin mr-2"></i>
                {/if}
                {$_("routing-round-trip-generate")}
            </button>
        </div>
    {/if}
</section>
