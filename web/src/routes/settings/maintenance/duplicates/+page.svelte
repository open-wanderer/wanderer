<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import Tabs from "$lib/components/base/tabs.svelte";
    import SimilarAssetsPanel from "$lib/components/settings/duplicates/similar_assets_panel.svelte";
    import SimilarTrailsPanel from "$lib/components/settings/duplicates/similar_trails_panel.svelte";
    import { _ } from "svelte-i18n";

    let activeTab = $state(page.url.searchParams.get("tab") === "assets" ? 1 : 0);
    let duplicateTabs = $derived([
        $_("duplicate-maintenance-trails-tab"),
        $_("duplicate-maintenance-assets-tab"),
    ]);

    $effect(() => {
        const selectedTab = activeTab === 1 ? "assets" : "trails";
        if (page.url.searchParams.get("tab") === selectedTab) {
            return;
        }

        const nextUrl = new URL(page.url);
        nextUrl.searchParams.set("tab", selectedTab);
        goto(`${nextUrl.pathname}?${nextUrl.searchParams.toString()}`, {
            keepFocus: true,
            noScroll: true,
            replaceState: true,
        });
    });
</script>

<svelte:head>
    <title>{$_("duplicate-maintenance-title")} | wanderer</title>
</svelte:head>

<div class="space-y-6">
    <div class="space-y-2">
        <h1 class="text-3xl font-bold">{$_("duplicate-maintenance-title")}</h1>
        <p class="text-sm text-gray-500 max-w-3xl">
            {$_("duplicate-maintenance-description")}
        </p>
    </div>

    <Tabs tabs={duplicateTabs} bind:activeTab />

    <div class:hidden={activeTab !== 0}>
        <SimilarTrailsPanel />
    </div>
    <div class:hidden={activeTab !== 1}>
        <SimilarAssetsPanel />
    </div>
</div>
