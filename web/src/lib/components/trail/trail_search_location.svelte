<script lang="ts">
    import Slider from "../base/slider.svelte";
    import type { TrailFilter } from "$lib/models/trail";
    import { searchLocations, type LocationSearchResult } from "$lib/stores/search_store";
    import { formatDistance } from "$lib/util/format_util";
    import { tick } from "svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        near: TrailFilter["near"];
    }

    let { near = $bindable() }: Props = $props();
    let query = $state("");
    let places = $state<LocationSearchResult[]>([]);
    let loading = $state(false);
    let failed = $state(false);
    let searched = $state(false);
    let input = $state<HTMLInputElement>();
    const id = $props.id();
    let selected = $derived(Number.isFinite(near.lat) && Number.isFinite(near.lon));

    // Replacing the draft place filter also resets an unfinished place query.
    $effect(() => {
        query = near.name ?? "";
    });

    $effect(() => {
        const text = query.trim();
        places = [];
        failed = false;
        searched = false;
        loading = false;
        if (selected || text.length < 2) return;

        const controller = new AbortController();
        loading = true;
        const timer = setTimeout(async () => {
            try {
                const results = await searchLocations(text, 5, (url, options) =>
                    fetch(url, { ...options, signal: controller.signal }));
                if (!controller.signal.aborted) {
                    places = results;
                    searched = true;
                }
            } catch {
                if (!controller.signal.aborted) failed = true;
            } finally {
                if (!controller.signal.aborted) loading = false;
            }
        }, 350);

        return () => {
            clearTimeout(timer);
            controller.abort();
        };
    });

    function selectPlace(place: LocationSearchResult) {
        near = { ...near, lat: place.lat, lon: place.lon, name: place.name };
        query = "";
    }

    async function clearPlace() {
        near = { radius: near.radius };
        query = "";
        await tick();
        input?.focus();
    }
</script>

<div class="location-editor">
    {#if selected}
        <div class="selected-place">
            <i class="fa fa-location-dot" aria-hidden="true"></i>
            <strong>{near.name || `${near.lat?.toFixed(3)}, ${near.lon?.toFixed(3)}`}</strong>
            <button type="button" class="change-place" onclick={clearPlace}>{$_("trail-search-change-place")}</button>
            <button type="button" class="clear-place" aria-label={$_("trail-search-clear-place")} onclick={clearPlace}><i class="fa fa-xmark" aria-hidden="true"></i></button>
        </div>
    {:else}
        <label class="visually-hidden" for={`${id}-place`}>{$_("trail-search-place")}</label>
        <div class="place-input">
            <i class="fa fa-magnifying-glass" aria-hidden="true"></i>
            <input bind:this={input} id={`${id}-place`} type="search" bind:value={query}
                autocomplete="off" placeholder={$_("trail-search-place")}
                onkeydown={(event) => { if (event.key === "Enter") event.preventDefault(); }} />
            {#if loading}<i class="fa fa-spinner fa-spin" aria-hidden="true"></i>{/if}
        </div>
        {#if places.length}
            <ul class="place-results" aria-label={$_("trail-search-place")}>
                {#each places as place}
                    <li><button type="button" class="place-option" onclick={() => selectPlace(place)}>
                        <i class="fa fa-location-dot" aria-hidden="true"></i>
                        <span><strong>{place.name}</strong><small>{place.description}</small></span>
                    </button></li>
                {/each}
            </ul>
        {:else if failed}
            <p class="place-status" role="status">{$_("trail-search-location-error")}</p>
        {:else if searched && !loading}
            <p class="place-status" role="status">{$_("no-results")}</p>
        {/if}
    {/if}
    <div class="radius-control">
        <p><span>{$_("radius")}</span><strong>{formatDistance(near.radius)}</strong></p>
        <div class="radius-track">
            <Slider minValue={100} maxValue={Math.max(10000, near.radius)} step={100}
                label={$_("radius")} formatValue={formatDistance} bind:currentValue={near.radius} />
        </div>
    </div>
</div>

<style>
    .visually-hidden { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; border: 0; }
    .location-editor { display: grid; gap: .65rem; max-width: 36rem; min-width: 0; }
    .location-editor > label { font-size: .85rem; }
    .place-input {
        align-items: center;
        background: rgba(var(--input-background));
        border: 1px solid rgba(var(--input-border));
        border-radius: .375rem;
        display: flex;
        gap: .5625rem;
        min-height: 3.125rem;
        padding: .75rem .9375rem;
    }
    .place-input:focus-within { border-color: rgba(var(--input-border-focus)); }
    .place-input > i {
        color: rgba(var(--content), .6);
        flex-shrink: 0;
        font-size: 1rem;
        width: 1rem;
    }
    .place-input input { background: transparent; border: 0; flex: 1; font-size: .85rem; min-width: 0; outline: 0; }
    .place-results { list-style: none; margin: 0; padding: 0; }
    .place-option { align-items: center; border-radius: .5rem; display: flex; gap: .65rem; padding: .5rem .65rem; text-align: left; width: 100%; }
    .place-option:hover, .place-option:focus-visible { background: rgba(var(--secondary-hover)); }
    .place-option > i { color: rgba(var(--content), .6); flex-shrink: 0; }
    .place-option > span { display: grid; gap: .1rem; min-width: 0; overflow-wrap: anywhere; }
    .place-option strong { font-size: .85rem; font-weight: 500; }
    .place-option small, .place-status { color: rgba(var(--content), .65); font-size: .75rem; }
    .selected-place { align-items: center; display: flex; flex-wrap: wrap; gap: .65rem; min-width: 0; overflow-wrap: anywhere; }
    .selected-place > strong { font-size: .85rem; }
    .change-place { font-size: .75rem; margin-left: auto; text-decoration: underline; text-underline-offset: 3px; }
    .clear-place { align-items: center; border-radius: 50%; display: inline-flex; flex-shrink: 0; height: 2rem; justify-content: center; width: 2rem; }
    .radius-control { display: grid; gap: .7rem; margin-top: .4rem; }
    .radius-control > p { display: flex; font-size: .8rem; justify-content: space-between; margin: 0; }
    .radius-track { min-width: 0; padding: .25rem .85rem; }
</style>
