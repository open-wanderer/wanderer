<script lang="ts">
    import * as noUiSlider from "nouislider";
    import "nouislider/dist/nouislider.css";
    import { onMount, untrack } from "svelte";

    interface Props {
        minValue?: number;
        maxValue?: number;
        currentMin?: number;
        currentMax?: number;
        minLabel?: string;
        maxLabel?: string;
        formatValue?: (value: number) => string;
        onset?: (data: [number, number]) => void;
        onupdate?: (data: [number, number]) => void;
    }

    let {
        minValue = 0,
        maxValue = 100,
        currentMin = $bindable(minValue),
        currentMax = $bindable(maxValue),
        minLabel,
        maxLabel,
        formatValue,
        onset,
        onupdate,
    }: Props = $props();

    const id = $props.id();
    let sliderContainer: HTMLDivElement;
    let slider = $state.raw<noUiSlider.API>();

    // Keep the handles in sync when a parent resets its draft or updates limits.
    $effect(() => {
        const api = slider;
        const lower = minValue;
        const upper = Math.max(lower + 1, maxValue);
        const values = [currentMin, currentMax];
        if (!api) return;
        untrack(() => {
            if (api.options.range.min !== lower || api.options.range.max !== upper) {
                api.updateOptions({ range: { min: lower, max: upper }, start: values }, false);
            } else {
                const previous = api.get(true) as number[];
                if (values.some((value, index) => Math.abs(value - previous[index]) > 1e-7)) api.set(values, false);
            }
        });
    });

    onMount(() => {
        const updateValues = (_values: (number | string)[], _handle: number, values: number[]) => {
            currentMin = values[0];
            currentMax = values[1];
            onupdate?.([currentMin, currentMax]);
        };

        const api = noUiSlider.create(sliderContainer, {
            start: [currentMin, currentMax],
            connect: true,
            range: {
                min: minValue,
                max: Math.max(minValue + 1, maxValue),
            },
            format: { to: String, from: Number },
            ariaFormat: { to: value => formatValue?.(value) ?? String(value) },
            handleAttributes: [minLabel ? { "aria-label": minLabel } : {}, maxLabel ? { "aria-label": maxLabel } : {}],
        });

        api.on("update", updateValues);

        api.on("set", () => {
            onset?.([currentMin, currentMax]);
        });
        slider = api;
        return () => { api.destroy(); slider = undefined; };
    });
</script>

<div class="my-4" {id} bind:this={sliderContainer}></div>

<style lang="postcss">
    @reference "tailwindcss";
    @reference "../../../css/app.css";

    :global(.noUi-target) {
        @apply border border-input-border shadow-none;
    }
    :global(.noUi-horizontal) {
        height: 6px;
    }
    :global(.noUi-connect) {
        @apply bg-primary;
    }

    :global(.noUi-horizontal .noUi-handle) {
        @apply rounded-full w-7 h-7 -top-3;
    }

    :global(.noUi-handle::before) {
        @apply content-none;
    }

    :global(.noUi-handle::after) {
        @apply content-none;
    }
</style>
