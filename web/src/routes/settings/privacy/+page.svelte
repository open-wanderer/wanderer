<script lang="ts">
    import { page } from "$app/state";
    import RadioGroup, {
        type RadioItem,
    } from "$lib/components/base/radio_group.svelte";
    import type { Settings } from "$lib/models/settings";
    import { settings_update } from "$lib/stores/settings_store";
    import { _ } from "svelte-i18n";

    let settings = $derived(page.data.settings as Settings);

    const accoutPrivacyItems: RadioItem[] = [
        {
            text: $_("public"),
            value: "public",
            description: $_("settings-privacy-account-public"),
        },
        {
            text: $_("private"),
            value: "private",
            description: $_("settings-privacy-account-private"),
        },
    ];

    const trailPrivacyItems: RadioItem[] = [
        {
            text: $_("public"),
            value: "public",
            description: $_("settings-privacy-trails-public"),
        },
        {
            text: $_("only-me"),
            value: "private",
            description: $_("settings-privacy-trails-private"),
        },
    ];

    const listPrivacyItems: RadioItem[] = [
        {
            text: $_("public"),
            value: "public",
            description: $_("settings-privacy-lists-public"),
        },
        {
            text: $_("only-me"),
            value: "private",
            description: $_("settings-privacy-lists-private"),
        },
    ];

    async function handleAccountSelection(e: RadioItem) {
        await settings_update({
            id: settings!.id,
            privacy: {
                trails: settings.privacy?.trails ?? "private",
                lists: settings.privacy?.lists ?? "private",
                account: e.value as "public" | "private",
            },
        });
    }

    async function handleTrailsSelection(e: RadioItem) {
        await settings_update({
            id: settings!.id,
            privacy: {
                trails: e.value as "public" | "private",
                lists: settings.privacy?.lists ?? "private",
                account: settings.privacy?.account ?? "public",
            },
        });
    }

    async function handleListsSelection(e: RadioItem) {
        await settings_update({
            id: settings!.id,
            privacy: {
                trails: settings.privacy?.trails ?? "private",
                lists: e.value as "public" | "private",
                account: settings.privacy?.account ?? "private",
            },
        });
    }
</script>

<svelte:head>
    <title>{$_("settings")} | {page.data.appTitle}</title>
</svelte:head>
<h2 class="text-2xl font-semibold">{$_("privacy")}</h2>
<hr class="mt-4 mb-6 border-input-border" />

<h4 class="text-xl font-medium mb-4">{$_("account-privacy")}</h4>
<RadioGroup
    name="account"
    items={accoutPrivacyItems}
    selected={settings?.privacy?.account == "private" ? 1 : 0}
    onchange={handleAccountSelection}
></RadioGroup>

<h4 class="text-xl font-medium mt-12 mb-4">
    {$_("trail", { values: { n: 2 } })}
</h4>
<RadioGroup
    name="trails"
    items={trailPrivacyItems}
    selected={settings?.privacy?.trails == "public" ? 0 : 1}
    onchange={handleTrailsSelection}
></RadioGroup>

<h4 class="text-xl font-medium mt-12 mb-4">
    {$_("list", { values: { n: 2 } })}
</h4>
<RadioGroup
    name="lists"
    items={listPrivacyItems}
    selected={settings?.privacy?.lists == "public" ? 0 : 1}
    onchange={handleListsSelection}
></RadioGroup>
