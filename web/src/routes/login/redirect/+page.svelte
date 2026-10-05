<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { oauth_login } from "$lib/stores/user_store";
    import type { AuthProviderInfo } from "pocketbase";
    import { onMount } from "svelte";

    let error = $state(page.url.searchParams.get("error"));
    let errorDescription = $state(page.url.searchParams.get("error_description"));
    const errorURI = page.url.searchParams.get("error_uri");

    const oauthState = page.url.searchParams.get("state");
    const code = page.url.searchParams.get("code");
    onMount(async () => {
        if (error || !oauthState || !code) {
            return;
        }

        const providerData = localStorage.getItem("provider");

        if (!providerData) {
            error = "missing_provider";
            errorDescription =
                "No OAuth provider was specified in local storage.";
            return;
        }

        const provider: AuthProviderInfo = JSON.parse(providerData);

        if (provider.state !== oauthState) {
            error = "mismacthed_provider";
            errorDescription =
                "OAuth provider does not match the one defined in local storage.";
            return;
        }

        oauth_login({
            name: provider.name,
            code: code,
            codeVerifier: provider.codeVerifier,
        })
            .then(() => {
                window.location.replace("/");
            })
            .catch((e) => {
                error = "oauth_error";
                errorDescription = e.toString();
            });
    });
</script>

<main
    class="flex items-center justify-center"
    style="min-height: calc(100vh - 388px)"
>
    {#if error}
        <div
            class="rounded-xl bg-input-background-error border border-red-400 p-6 max-w-xl space-y-4"
        >
            <h5 class="text-xl font-semibold">{error}</h5>
            <p>{errorDescription}</p>
            {#if errorURI}
                <p><a class="underline" href={errorURI}>More Info</a></p>
            {/if}
        </div>
    {:else}
        <div class="max-w-fit space-y-4 text-center">
            <div class="spinner spinner-dark"></div>
            <h5 class="text-xl font-semibold">Authenticating...</h5>
        </div>
    {/if}
</main>
