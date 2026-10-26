<script lang="ts">
    import * as noUiSlider from "nouislider";
    import "nouislider/dist/nouislider.css";
    import { onMount, untrack } from "svelte";
    import { type Options as SliderOptions } from "nouislider";

    interface Props {
        minValue?: number;
        maxValue?: number;
        currentValue?: any;
        label?: string;
        formatValue?: (value: number) => string;
        onset?: (value: number) => void;
    }

    let {
        minValue = 0,
        maxValue = 100,
        currentValue = $bindable(maxValue / 2),
        label,
        formatValue,
        onset,
        ...sliderOptions
    }: Props & Partial<SliderOptions> = $props();

    const id = $props.id();
    let sliderContainer: HTMLDivElement;
    let slider = $state.raw<noUiSlider.API>();

    $effect(() => {
        const api = slider;
        const lower = minValue;
        const upper = Math.max(lower + 1, maxValue);
        const value = currentValue;
        if (!api) return;
        untrack(() => {
            if (api.options.range.min !== lower || api.options.range.max !== upper) {
                api.updateOptions({ range: { min: lower, max: upper }, start: value }, false);
            } else if (Math.abs(value - Number(api.get(true))) > 1e-7) {
                api.set(value, false);
            }
        });
    });

    onMount(() => {
        const api = noUiSlider.create(sliderContainer, {
            start: currentValue,
            connect: [true, false],
            ariaFormat: { to: value => formatValue?.(value) ?? String(value) },
            handleAttributes: [label ? { "aria-label": label } : {}],
            ...sliderOptions,
            range: {
                min: minValue,
                max: Math.max(minValue + 1, maxValue),
            },
        });

        api.on("update", (values) => {
            currentValue = parseFloat(String(values[0]));
        });
        api.on("set", () => {
            onset?.(currentValue);
        });
        slider = api;
        return () => { api.destroy(); slider = undefined; };
    });

    export function set(value: number) {
        slider?.set(value);
    }
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
