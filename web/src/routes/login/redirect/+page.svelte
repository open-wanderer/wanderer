<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { oauth_login } from "$lib/stores/user_store";
    import type { AuthProviderInfo } from "pocketbase";
    import { onMount } from "svelte";

    let error = $state(page.url.searchParams.get("error"));
    let errorDescription = $state(
        page.url.searchParams.get("error_description"),
    );
    const errorURI = page.url.searchParams.get("error_uri");

    const oauthState = page.url.searchParams.get("state");
    const code = page.url.searchParams.get("code");

    // Appended to `state` by the app before opening the authorization URL, so
    // an app-originated flow can be told apart from a plain web login by
    // looking at the returned `state` value itself, rather than inferring it
    // from the absence of a web-only signal (localStorage) — browsers can
    // legitimately lack that too. PocketBase's OAuth2 code exchange never
    // inspects `state` server-side (it's a purely client-side CSRF check), so
    // tagging it is safe. Keep in sync with the identical constant in
    // app/lib/provider/auth_provider.dart.
    const APP_STATE_MARKER = ".wanderer-app";

    // Set when relaying to the app. Not every browser reliably completes a
    // JS-initiated navigation to a custom URI scheme, so this also renders a
    // real, tappable fallback link as a backstop.
    let appCallbackUrl: string | null = $state(null);

    onMount(async () => {
        const isAppFlow = !!oauthState && oauthState.endsWith(APP_STATE_MARKER);

        if (isAppFlow) {
            const params = new URLSearchParams();
            if (code) params.set("code", code);
            if (oauthState) params.set("state", oauthState);
            if (error) params.set("error", error);
            if (errorDescription)
                params.set("error_description", errorDescription);
            const callbackUrl = `wanderer://oauth-callback?${params.toString()}`;
            appCallbackUrl = callbackUrl;
            window.location.href = callbackUrl;

            // If this really is the app, the browser/webview navigates away (or
            // closes) right about now and none of this executes. If it doesn't,
            // this tab has no way to know whether the app actually completed the
            // login — on some browsers (notably Firefox on Android) the OS still
            // hands control back to the app and completes login successfully,
            // it just doesn't dismiss this leftover tab automatically the way
            // Chrome does. So there's nothing to declare as failed here: the tap
            // link plus a "safe to close" hint (rendered below) is the correct,
            // honest fallback rather than guessing at an error.
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
                window.location.href = "/";
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
            <div class="spinner spinner-dark dark:spinner-light"></div>
            <h5 class="text-xl font-semibold">Authenticating...</h5>
            {#if appCallbackUrl}
                <p class="text-sm border border-input-border rounded-md mt-8">
                    If you are not redirected automatically, <a
                        class="underline"
                        href={appCallbackUrl}>tap here to return to the app</a
                    >.
                </p>
            {/if}
        </div>
    {/if}
</main>
