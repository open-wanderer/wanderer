<script lang="ts">
    import { page } from "$app/state";
    import Toggle from "$lib/components/base/toggle.svelte";
    import { normalizeTrailSearchFilterSections, trailSearchFilters, type TrailSearchFilterId, type TrailSearchFilterSections } from "$lib/models/trail_search_settings";
    import { settings_update } from "$lib/stores/settings_store";
    import { onMount, untrack } from "svelte";
    import { _ } from "svelte-i18n";

    let sections = $state(normalizeTrailSearchFilterSections(untrack(() => page.data.settings?.behavior?.searchFilterSections)));
    let saving = $state(false);
    let saveError = $state(false);
    let ready = $state(false);

    onMount(() => { ready = true; });

    $effect(() => {
        const stored = page.data.settings?.behavior?.searchFilterSections;
        untrack(() => { if (!saving) sections = normalizeTrailSearchFilterSections(stored); });
    });

    async function save(next: TrailSearchFilterSections) {
        const settings = page.data.settings;
        if (!settings?.id || saving) return;
        const previous = sections;
        sections = normalizeTrailSearchFilterSections(next);
        saving = true;
        saveError = false;
        try {
            await settings_update({
                id: settings.id,
                behavior: { ...settings.behavior, allowAutoGeolocate: settings.behavior?.allowAutoGeolocate ?? false, searchFilterSections: sections },
            });
        } catch {
            sections = previous;
            saveError = true;
        } finally { saving = false; }
    }

    function change(filter: TrailSearchFilterId, advanced: boolean) {
        void save({ ...sections, [filter]: advanced ? "advanced" : "general" });
    }
</script>

<svelte:head><title>{$_("settings-search")} | wanderer</title></svelte:head>
<h2 class="text-2xl font-semibold">{$_("settings-search")}</h2>
<p class="mt-2 text-sm text-gray-500">{$_("settings-search-help")}</p>
<hr class="mt-4 mb-6 border-input-border" />
<div class="search-filter-settings grid items-center gap-4" style="grid-template-columns: 1fr max-content;"
    role="group" aria-label={$_("settings-search-placement")} aria-busy={saving}>
    <div></div>
    <span class="text-sm font-medium whitespace-nowrap">{$_("settings-search-advanced")}</span>
    {#each trailSearchFilters as filter}
        <p>{$_(filter.label)}</p>
        <div class="justify-self-end">
            <Toggle name={`search-filter-${filter.id}`}
                ariaLabel={`${$_(filter.label)}: ${$_("settings-search-advanced")}`}
                value={sections[filter.id] === "advanced"} disabled={!ready || saving}
                onchange={(enabled) => change(filter.id, enabled)} />
        </div>
    {/each}
</div>
{#if saveError}<p role="alert" class="save-error">{$_("settings-search-error")}</p>{/if}

<style>
    .save-error { font-size: .85rem; margin-top: 1rem; }
</style>
