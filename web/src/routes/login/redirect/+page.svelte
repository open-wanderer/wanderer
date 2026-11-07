<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { oauth_login } from "#lib/stores/user_store.ts";
    import { appCallbackUrl, httpUrlOrNull } from "#lib/util/oauth_app_util.ts";
    import type { AuthProviderInfo } from "pocketbase";
    import { onMount } from "svelte";
    import { _ } from "svelte-i18n";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

    let error = $state(page.url.searchParams.get("error"));
    let errorDescription = $state(
        page.url.searchParams.get("error_description"),
    );
    const errorURI = httpUrlOrNull(page.url.searchParams.get("error_uri"));

    const oauthState = page.url.searchParams.get("state");
    const code = page.url.searchParams.get("code");

    // Set when this login belongs to the app. Rendered as a link, because a
    // state only proves that this server issued it, and any client can ask for
    // one: the user has to confirm that the login is theirs.
    let appCallback: string | null = $state(null);

    onMount(async () => {
        if (data.appLoginRefused) {
            error = "app_login_refused";
            errorDescription = $_("app-login-refused");
            return;
        }

        if (data.appLogin) {
            const params = new URLSearchParams();
            if (code) params.set("code", code);
            if (oauthState) params.set("state", oauthState);
            if (error) params.set("error", error);
            if (errorDescription)
                params.set("error_description", errorDescription);
            appCallback = appCallbackUrl(params, navigator.userAgent);

            // Without a code there is nothing to protect, and the app is
            // waiting to hear that the login failed.
            if (!code) {
                window.location.href = appCallback;
            }
            return;
        }

        // Web flow: complete login using the provider stashed in localStorage
        // by the login page.
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
    {:else if appCallback && code}
        <div class="max-w-xl space-y-4 p-6">
            <h5 class="text-xl font-semibold">{$_("app-login-confirm-title")}</h5>
            <p>
                {$_("app-login-confirm-text", {
                    values: {
                        provider: data.appLogin?.provider,
                        host: page.url.host,
                    },
                })}
            </p>
            <p class="font-semibold">{$_("app-login-confirm-warning")}</p>
            <a class="btn-primary inline-block" href={appCallback}
                >{$_("app-login-confirm-action")}</a
            >
        </div>
    {:else}
        <div class="max-w-fit space-y-4 text-center">
            <div class="spinner spinner-dark dark:spinner-light"></div>
            <h5 class="text-xl font-semibold">Authenticating...</h5>
        </div>
    {/if}
</main>
