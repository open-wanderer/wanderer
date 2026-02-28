<script lang="ts">
    import { page } from "$app/state";
    import Search, {
        type SearchItem,
    } from "$lib/components/base/search.svelte";
    import Select, {
        type SelectItem,
    } from "$lib/components/base/select.svelte";

    import TextField from "$lib/components/base/text_field.svelte";
    import {
        searchLocations,
        type LocationSearchResult,
    } from "$lib/stores/search_store";
    import { settings_update } from "$lib/stores/settings_store";
    import { currentUser } from "$lib/stores/user_store";
    import { getIconForLocation } from "$lib/util/icon_util";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";
    import Toggle from "$lib/components/base/toggle.svelte";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";

    let settings = $derived(page.data.settings);

    let allowAutoGeolocate = $state(
        page.data.settings.behavior?.allowAutoGeolocate ?? false,
    );

    async function handleAllowAutoGeolocateChange() {
        if (!settings) {
            return;
        }

        try {
            if (!settings.behavior) {
                settings.behavior = { allowAutoGeolocate: allowAutoGeolocate };
            } else {
                settings.behavior.allowAutoGeolocate = allowAutoGeolocate;
            }

            await settings_update(settings);
        } catch (e) {
            show_toast({
                type: "error",
                icon: "close",
                text: "Error updating behavior settings",
            });
            console.error(e);
        }
    }

    const mapFocus: SelectItem[] = [
        { text: $_("trail", { values: { n: 2 } }), value: "trails" },
        { text: $_("location"), value: "location" },
    ];

    let selectedLanguage = "en";
    let selectedMapFocus = $state("trails");

    let searchDropdownItems: SearchItem[] = $state([]);
    let citySearchQuery: string = $state("");

    let customTilesetName: string = $state("");
    let customTilesetURL: string = $state("");
    let terrainURL: string = $state("");
    let hillshadingURL: string = $state("");

    onMount(() => {
        citySearchQuery = settings?.location?.name ?? "";
        selectedLanguage = settings?.language || "en";
        selectedMapFocus = settings?.mapFocus ?? "trails";

        terrainURL = settings?.terrain?.terrain ?? "";
        hillshadingURL = settings?.terrain?.hillshading ?? "";
    });

    async function searchCities(q: string) {
        const r = await searchLocations(q, 5);

        searchDropdownItems = r.map((h: LocationSearchResult) => ({
            text: h.name,
            description: h.description,
            value: h,
            icon: getIconForLocation(h),
        }));
    }

    async function handleSearchClick(item: SearchItem) {
        citySearchQuery = item.text;
        await settings_update({
            id: settings?.id!,
            location: {
                name: item.value.name,
                lat: item.value.lat,
                lon: item.value.lon,
            },
        });
    }

    async function handleMapFocusSelection(value: "trails" | "location") {
        await settings_update({
            id: settings!.id,
            mapFocus: value,
        });
    }

    async function handleTilesetAdd() {
        if (!customTilesetName || !customTilesetURL) {
            return;
        }
        await settings_update({
            id: settings!.id,
            tilesets: [
                ...(settings.tilesets ?? []),
                { name: customTilesetName, url: customTilesetURL },
            ],
        });
        customTilesetName = "";
        customTilesetURL = "";
    }

    async function handleTilesetDelete(index: number) {
        settings.tilesets.splice(index, 1);
        await settings_update({
            id: settings!.id,
            tilesets: settings.tilesets,
        });
    }

    async function handleTerrainAdd() {
        await settings_update({
            id: settings!.id,
            terrain: { terrain: terrainURL, hillshading: hillshadingURL },
        });
    }

    let terrainSaveEnabled = $derived(
        terrainURL !== settings?.terrain?.terrain ||
            hillshadingURL !== settings?.terrain?.hillshading,
    );
</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>
<h2 class="text-2xl font-semibold">{$_("map")}</h2>
<hr class="mt-4 mb-6 border-input-border" />
{#if $currentUser}
    <div class="space-y-12">
        <div>
            <h4 class="text-xl font-medium mb-2">{$_("focus-map-on")}</h4>
            <Select
                items={mapFocus}
                bind:value={selectedMapFocus}
                onchange={handleMapFocusSelection}
            ></Select>
            {#if selectedMapFocus == "location"}
                <div class="mt-3">
                    <Search
                        items={searchDropdownItems}
                        placeholder="{$_('search-for-trails-places')}..."
                        clearAfterSelect={false}
                        bind:value={citySearchQuery}
                        onupdate={searchCities}
                        onclick={handleSearchClick}
                    ></Search>
                </div>
            {/if}
            <div
                class="mt-4 grid gap-4"
                style="grid-template-columns: 1fr min-content ;"
            >
                <p>{$_("allow-auto-geolocate")}</p>
                <div>
                    <Toggle
                        bind:value={allowAutoGeolocate}
                        onchange={handleAllowAutoGeolocateChange}
                    ></Toggle>
                </div>
            </div>
        </div>
        <div>
            <h4 class="text-xl font-medium mb-2">{$_("tilesets")}</h4>

            {#each settings.tilesets ?? [] as tileset, i}
                <div
                    class="flex items-center justify-between px-4 py-2 border border-input-border rounded-xl mb-2"
                >
                    <div>
                        <p>{tileset.name}</p>
                        <p class="text-sm text-gray-500">{tileset.url}</p>
                    </div>
                    <button
                        aria-label="Delete tileset"
                        class="btn-icon"
                        onclick={() => handleTilesetDelete(i)}
                        ><i class="fa fa-trash text-red-500"></i></button
                    >
                </div>
            {/each}
            <div class="flex gap-4 items-center">
                <TextField
                    label={$_("name")}
                    bind:value={customTilesetName}
                    placeholder={$_("name")}
                ></TextField>
                <div class="flex items-center basis-full gap-2">
                    <div class="flex-grow">
                        <TextField
                            label="URL"
                            bind:value={customTilesetURL}
                            placeholder="https://.../style.json"
                        ></TextField>
                    </div>
                    <button
                        aria-label="Add tileset"
                        class="btn-icon mt-6"
                        onclick={handleTilesetAdd}
                        ><i class="fa fa-plus"></i></button
                    >
                </div>
            </div>
        </div>

        <div>
            <h4 class="text-xl font-medium mb-2">{$_("Terrain")}</h4>
            <div class="flex items-center gap-2">
                <div class="basis-full">
                    <TextField
                        label="Terrain URL"
                        bind:value={terrainURL}
                        placeholder="https://.../tiles.json"
                    ></TextField>
                </div>
                <div class="basis-full">
                    <TextField
                        label="Hillshading URL"
                        bind:value={hillshadingURL}
                        placeholder="https://.../tiles.json"
                    ></TextField>
                </div>
                <button
                    aria-label="Save terrain settings"
                    disabled={!terrainSaveEnabled}
                    class="btn-icon mt-6"
                    class:hover:!bg-background={!terrainSaveEnabled}
                    onclick={handleTerrainAdd}
                    class:text-gray-500={!terrainSaveEnabled}
                    ><i class="fa fa-save"></i></button
                >
            </div>
        </div>
    </div>
{/if}
