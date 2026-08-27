<script lang="ts">
    import { _ } from "svelte-i18n";

    interface Props {
        id: string;
        label: string;
        bins: number[];
        min?: number;
        max: number;
        step?: number;
        currentMin?: number;
        currentMax?: number;
        formatValue?: (value: number) => string;
        capped?: boolean;
    }

    let {
        id,
        label,
        bins,
        min = 0,
        max,
        step = 1,
        currentMin = $bindable(min),
        currentMax = $bindable(max),
        formatValue = (value) => value.toString(),
        capped = true,
    }: Props = $props();

    let span = $derived(Math.max(max - min, step));
    let minPosition = $derived(((currentMin - min) / span) * 100);
    let maxPosition = $derived(((currentMax - min) / span) * 100);
    let highestBin = $derived(Math.max(...bins, 1));
    let formattedMinimum = $derived(formatValue(currentMin));
    let formattedMaximum = $derived(
        `${formatValue(currentMax)}${capped && currentMax === max ? "+" : ""}`,
    );

    function setMinimum(event: Event) {
        const next = Number((event.currentTarget as HTMLInputElement).value);
        currentMin = Math.min(next, currentMax - step);
    }

    function setMaximum(event: Event) {
        const next = Number((event.currentTarget as HTMLInputElement).value);
        currentMax = Math.max(next, currentMin + step);
    }

    function binIsSelected(index: number) {
        const center = min + ((index + 0.5) / bins.length) * span;
        return center >= currentMin && center <= currentMax;
    }
</script>

<div class="space-y-2">
    <div class="flex items-start justify-between gap-3">
        <label for={`${id}-minimum`} class="text-sm font-medium">{label}</label>
        <output
            id={`${id}-values`}
            class="shrink-0 text-sm font-semibold text-content"
        >
            {formattedMinimum} – {formattedMaximum}
        </output>
    </div>

    <div class="histogram-range relative h-16 pt-1">
        <div
            class="absolute inset-x-0 top-0 flex h-10 items-end gap-[2px]"
            aria-hidden="true"
        >
            {#each bins as bin, index}
                <span
                    class="min-w-0 flex-1 rounded-t-sm bg-input-border transition-colors"
                    class:bg-input-border-focus={binIsSelected(index)}
                    class:opacity-75={binIsSelected(index)}
                    style={`height: ${Math.max(10, (bin / highestBin) * 100)}%`}
                ></span>
            {/each}
        </div>

        <div
            class="absolute inset-x-0 bottom-[7px] h-1 rounded-full bg-input-border"
            aria-hidden="true"
        ></div>
        <div
            class="absolute bottom-[7px] h-1 rounded-full bg-input-border-focus"
            style={`left: ${minPosition}%; right: ${100 - maxPosition}%`}
            aria-hidden="true"
        ></div>

        <input
            id={`${id}-minimum`}
            class="range-input"
            type="range"
            {min}
            {max}
            {step}
            value={currentMin}
            aria-label={`${label}: ${$_("filter-preview-minimum")}`}
            aria-valuetext={formattedMinimum}
            aria-describedby={`${id}-values`}
            oninput={setMinimum}
        />
        <input
            id={`${id}-maximum`}
            class="range-input"
            type="range"
            {min}
            {max}
            {step}
            value={currentMax}
            aria-label={`${label}: ${$_("filter-preview-maximum")}`}
            aria-valuetext={formattedMaximum}
            aria-describedby={`${id}-values`}
            oninput={setMaximum}
        />
    </div>
</div>

<style>
    .range-input {
        appearance: none;
        background: transparent;
        bottom: 0;
        height: 1.25rem;
        left: 0;
        margin: 0;
        pointer-events: none;
        position: absolute;
        width: 100%;
    }

    .range-input::-webkit-slider-runnable-track {
        background: transparent;
        border: 0;
        height: 0.25rem;
    }

    .range-input::-webkit-slider-thumb {
        appearance: none;
        background: rgba(var(--background));
        border: 2px solid rgba(var(--input-border-focus));
        border-radius: 9999px;
        box-shadow: 0 1px 4px rgb(0 0 0 / 0.2);
        height: 1.125rem;
        margin-top: -0.4375rem;
        pointer-events: auto;
        width: 1.125rem;
    }

    .range-input::-moz-range-track {
        background: transparent;
        border: 0;
        height: 0.25rem;
    }

    .range-input::-moz-range-thumb {
        background: rgba(var(--background));
        border: 2px solid rgba(var(--input-border-focus));
        border-radius: 9999px;
        box-shadow: 0 1px 4px rgb(0 0 0 / 0.2);
        height: 1.125rem;
        pointer-events: auto;
        width: 1.125rem;
    }

    .range-input:focus-visible::-webkit-slider-thumb {
        box-shadow: 0 0 0 4px rgba(var(--input-ring));
        outline: none;
    }

    .range-input:focus-visible::-moz-range-thumb {
        box-shadow: 0 0 0 4px rgba(var(--input-ring));
        outline: none;
    }
</style>
