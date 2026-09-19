<script lang="ts">
    import noUiSlider from "nouislider";
    import "nouislider/dist/nouislider.css";
    import { onMount } from "svelte";
    import { type Options as SliderOptions } from "nouislider";

    interface Props {
        minValue?: number;
        maxValue?: number;
        currentMin?: any;
        currentMax?: any;
        onset?: (data: [number, number]) => void;
        onupdate?: (data: [number, number]) => void;
    }

    let {
        minValue = 0,
        maxValue = 100,
        currentMin = $bindable(minValue),
        currentMax = $bindable(maxValue),
        onset,
        onupdate,
        ...sliderOptions
    }: Props & Partial<SliderOptions> = $props();

    let sliderContainer: any = $state();

    onMount(() => {
        const updateValues = (values: string[]) => {
            currentMin = parseFloat(values[0]);
            currentMax = parseFloat(values[1]);
            onupdate?.([currentMin, currentMax]);
        };

        noUiSlider.create(sliderContainer, {
            start: [currentMin, currentMax],
            connect: true,
            ...sliderOptions,
            range: {
                min: minValue,
                max: maxValue,
            },
        });

        sliderContainer.noUiSlider.on("update", updateValues);

        sliderContainer.noUiSlider.on("set", () => {
            onset?.([currentMin, currentMax]);
        });
    });
</script>

<div class="my-4" id="slider" bind:this={sliderContainer}></div>

<style lang="postcss">
    @reference "tailwindcss";
    @reference "../../../css/app.css";

    :global(.noUi-target) {
        @apply border border-input-border bg-input-background shadow-none;
    }

    :global(.noUi-horizontal) {
        height: 6px;
    }
    :global(.noUi-connect) {
        @apply bg-input-border-focus;
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
