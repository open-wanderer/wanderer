<script lang="ts">
    import { plugin_oauth_callback } from "$lib/stores/plugin_instance_store";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";

    let error = $state("");

    onMount(async () => {
        const code = page.url.searchParams.get("code") ?? "";
        const state = page.url.searchParams.get("state") ?? "";
        const providerError = page.url.searchParams.get("error");
        const instanceId = sessionStorage.getItem(`wanderer_plugin_oauth_${state}`) ?? "";
        sessionStorage.removeItem(`wanderer_plugin_oauth_${state}`);

        if (providerError) {
            error = providerError;
            return;
        }
        if (!code || !state || !instanceId) {
            error = "invalid_oauth_callback";
            return;
        }

        try {
            await plugin_oauth_callback({ instanceId, code, state });
            await goto("/settings/plugins");
        } catch (e) {
            error = e instanceof Error ? e.message : "oauth_error";
        }
    });
</script>

<svelte:head>
    <title>{$_("plugins")} | wanderer</title>
</svelte:head>

<div class="mx-auto max-w-md py-12">
    {#if error}
        <h3 class="text-xl font-semibold">{$_("error")}</h3>
        <p class="mt-3 text-sm text-gray-500">{error}</p>
        <a class="btn-secondary mt-6 inline-block" href="/settings/plugins">{$_("plugins")}</a>
    {:else}
        <h3 class="text-xl font-semibold">{$_("plugins")}</h3>
    {/if}
</div>
