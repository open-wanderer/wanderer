<script lang="ts">
    import { _ } from "svelte-i18n";
    import DoubleSlider from "../base/double_slider.svelte";

    interface Props {
        label: string;
        icon?: string;
        min: number;
        max: number;
        limit: number;
        formatValue: (value: number) => string;
    }
    let { label, icon, min = $bindable(), max = $bindable(), limit, formatValue }: Props = $props();
</script>

<fieldset>
    <legend>{#if icon}<i class={`fa ${icon}`} aria-hidden="true"></i>{/if}{label}</legend>
    <div class="range-track">
        <DoubleSlider minValue={0} maxValue={limit} bind:currentMin={min} bind:currentMax={max}
            minLabel={`${label} · ${$_("trail-search-min")}`} maxLabel={`${label} · ${$_("trail-search-max")}`}
            {formatValue} />
    </div>
    <div class="range-values">
        <span>{formatValue(min)}</span>
        <span>{formatValue(max)}{max >= limit ? "+" : ""}</span>
    </div>
</fieldset>

<style>
    fieldset { border: 0; margin: 0; max-width: 36rem; min-width: 0; padding: 0; width: 100%; }
    legend { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; border: 0; }
    legend i { font-size: .75rem; margin-right: .5rem; opacity: .55; }
    .range-track { padding: .25rem .85rem; }
    .range-values { display: flex; font-size: .85rem; font-variant-numeric: tabular-nums; justify-content: space-between; }
</style>
