<script lang="ts">
    import { page } from "$app/state";
    import RadioGroup, {
        type RadioItem,
    } from "$lib/components/base/radio_group.svelte";
    import Select, {
        type SelectItem,
    } from "$lib/components/base/select.svelte";
    import type { Language } from "$lib/models/settings";
    import { settings_update } from "$lib/stores/settings_store";
    import { _ } from "svelte-i18n";

    const settings = page.data.settings;

    const languages: SelectItem[] = [
        { text: $_("czech"), value: "cs" },
        { text: $_("english"), value: "en" },
        { text: $_("german"), value: "de" },
        { text: $_("spanish"), value: "es" },
        { text: $_("basque"), value: "eu" },
        { text: $_("french"), value: "fr" },
        { text: $_("hungarian"), value: "hu" },
        { text: $_("italian"), value: "it" },
        { text: $_("dutch"), value: "nl" },
        { text: $_("norwegian"), value: "no" },
        { text: $_("polish"), value: "pl" },
        { text: $_("portuguese"), value: "pt" },
        { text: $_("russian"), value: "ru" },
        { text: $_("chinese"), value: "zh" },
    ];

    const units: RadioItem[] = [
        { text: $_("metric"), value: "metric" },
        { text: $_("imperial"), value: "imperial" },
    ];

    let selectedLanguage = $state(settings?.language ?? "en");

    async function handleLanguageSelection(value: Language) {
        await settings_update({
            id: settings!.id,
            language: value,
        });
        window.location.reload();
    }

    async function handleUnitSelection(e: RadioItem) {
        await settings_update({
            id: settings!.id,
            unit: e.value as "imperial" | "metric",
        });
    }
</script>

<svelte:head>
    <title>{$_("settings")} | {page.data.appTitle}</title>
</svelte:head>
<h2 class="text-2xl font-semibold">{$_("language")} & {$_("units")}</h2>
<hr class="mt-4 mb-6 border-input-border" />
<h4 class="text-xl font-medium mb-2">{$_("language")}</h4>
<Select
    items={languages}
    bind:value={selectedLanguage}
    onchange={handleLanguageSelection}
></Select>
<h4 class="text-xl font-medium mt-6 mb-2">{$_("units")}</h4>
<RadioGroup
    name="unit"
    items={units}
    selected={settings?.unit == "metric" ? 0 : 1}
    onchange={handleUnitSelection}
></RadioGroup>
