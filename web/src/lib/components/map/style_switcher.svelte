<script lang="ts">
    import type { StyleSwitcherControlOptions } from "$lib/vendor/maplibre-style-switcher/style-switcher-control";
    import { _ } from "svelte-i18n";
    import RadioGroup, { type RadioItem } from "../base/radio_group.svelte";
    import { defaultMapState } from "$lib/vendor/maplibre-layer-manager/layers";

    interface Props {
        settings: StyleSwitcherControlOptions;
    }

    let { settings }: Props = $props();

    let mapState = $state(settings.state);

    let container: HTMLDivElement;

    let styleItems: RadioItem[] = $derived(
        Object.entries(settings.styles).map<RadioItem>(([name, style]) => ({
            text: name,
            value: style,
        })),
    );

    let selectedStyle: number = $derived(
        styleItems.findIndex((i) => i.text === mapState.base),
    );

    let showSwitcher: boolean = $state(false);

    function toggleSwitcher() {
        showSwitcher = !showSwitcher;
    }

    function hideSwitcher() {
        showSwitcher = false;
    }

    export function getElement() {
        return container;
    }
</script>

<div
    bind:this={container}
    class="maplibregl-ctrl maplibregl-ctrl-group relative z-10"
>
    <button onclick={toggleSwitcher} aria-label="toggle style switcher">
        <i class="fa fa-layer-group text-black"></i>
    </button>

    <div
        class="absolute bg-menu-background rounded-lg px-4 py-2 mt-2 shadow-xl space-y-3 max-h-96 overflow-y-scroll min-w-48"
        class:hidden={!showSwitcher}
        style="transform: translateX(calc(-100% + 29px))"
    >
        <div class="flex items-center justify-between">
            <span class="font-semibold whitespace-nowrap text-base"
                >{$_("map-style")}</span
            >
            <button
                class="fa fa-close !rounded-full"
                onclick={hideSwitcher}
                aria-label="close style switcher"
            ></button>
        </div>
        <RadioGroup
            name="completed"
            items={styleItems}
            selected={selectedStyle}
            onchange={(style) => {
                mapState.base = style.text;
                settings.onchange(mapState);
            }}
        ></RadioGroup>
        <hr class="border-input-border" />
        <span class="font-semibold whitespace-nowrap text-base"
            >{$_("layer", { values: { n: 2 } })}</span
        >
        {#each Object.keys(defaultMapState.overlays) as overlay}
            <div class="flex items-center mt-2 mb-4">
                <input
                    id="{overlay}-layer-checkbox"
                    type="checkbox"
                    bind:checked={mapState.overlays[overlay]}
                    class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                    onchange={(e) => {
                        settings.onchange(mapState);
                    }}
                />

                <label for="{overlay}-layer-checkbox" class="ms-2 text-sm"
                    >{$_(overlay)}</label
                >
            </div>
        {/each}
        <hr class="border-input-border" />
        <p class="font-semibold whitespace-nowrap text-base">{$_("pois")}</p>
        {#each Object.entries(defaultMapState.pois) as [category, pois]}
            <p class="text-sm font-medium whitespace-nowrap">{$_(category)}</p>
            {#each Object.keys(pois) as poi}
                <div class="flex items-center mt-2 mb-4 ml-2">
                    <input
                        id="{poi}-poi-checkbox"
                        type="checkbox"
                        bind:checked={mapState.pois[category][poi]}
                        class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
                        onchange={(e) => {
                            settings.onchange(mapState);
                        }}
                    />

                    <label for="{poi}-poi-checkbox" class="ms-2 text-sm"
                        >{$_(poi)}</label
                    >
                </div>
            {/each}
        {/each}
    </div>
</div>
