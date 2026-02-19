<script lang="ts">
    import { page } from "$app/state";
    import { env } from "$env/dynamic/public";
    import Button from "$lib/components/base/button.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import LogoTextTwoLineDark from "$lib/components/logo/logo_text_two_line_dark.svelte";
    import LogoTextTwoLineLight from "$lib/components/logo/logo_text_two_line_light.svelte";
    import type { User } from "$lib/models/user";
    import { theme } from "$lib/stores/theme_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { login } from "$lib/stores/user_store";
    import { APIError } from "$lib/util/api_util";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { type AuthProviderInfo } from "pocketbase";
    import { _ } from "svelte-i18n";
    import { z } from "zod";
    import type { PageProps } from "./$types";

    let loading: boolean = $state(false);

    let { data: pageData }: PageProps = $props();

    const authProviders = $derived.by(
        () => pageData.authMethods.oauth2.providers,
    );

    const { form, errors, data } = createForm<User>({
        initialValues: {
            id: "",
            username: "",
            password: "",
        },
        extend: validator({
            schema: z.object({
                username: z.string().min(1, "required"),
                password: z
                    .string()
                    .min(1, "required")
                    .max(
                        72,
                        $_("must-be-at-most-n-characters-long", {
                            values: { n: 72 },
                        }),
                    ),
            }),
        }),
        onSubmit: async (newUser) => {
            loading = true;

            try {
                await login(newUser);
                window.location.href = page.url.searchParams.get("r") ?? "/";
            } catch (e) {
                if (
                    e instanceof APIError &&
                    e.message == "Failed to authenticate."
                ) {
                    show_toast({
                        icon: "close",
                        type: "error",
                        text: $_("wrong-username-or-password"),
                    });
                } else {
                    show_toast({
                        icon: "close",
                        type: "error",
                        text: $_("error-during-login"),
                    });
                }
            } finally {
                loading = false;
            }
        },
    });

    function setProvider(provider: AuthProviderInfo) {
        localStorage.setItem("provider", JSON.stringify(provider));
    }
</script>

<svelte:head>
    <title>{$_("login")} | wanderer</title>
</svelte:head>
<main class="flex justify-center">
    <form
        class="login-panel max-w-md border border-input-border rounded-xl p-8 flex flex-col justify-center items-center gap-4 w-[28rem] mt-8"
        use:form
    >
        {#if $theme == "light"}
            <LogoTextTwoLineDark></LogoTextTwoLineDark>
        {:else}
            <LogoTextTwoLineLight></LogoTextTwoLineLight>
        {/if}
        <h4 class="text-xl font-semibold">{$_("slogan")}</h4>
        {#if page.data.authMethods.password.enabled}
            <div class="space-y-6 w-80">
                <TextField
                    name="username"
                    label="{$_('username')}/{$_('email')}"
                    error={$errors.username}
                ></TextField>
                <div class="flex flex-col">
                    <TextField
                        name="password"
                        label={$_("password")}
                        type="password"
                        error={$errors.password}
                    ></TextField>
                    <a
                        class="text-xs underline decoration-dashed float-end mt-1 self-end"
                        href="/auth/reset">{$_("forgot-your-password")}</a
                    >
                </div>
                <Button
                    primary={true}
                    extraClasses={"min-w-full"}
                    type="submit"
                    {loading}>Login</Button
                >
            </div>
            {#if env.PUBLIC_DISABLE_SIGNUP !== "true"}
                <span
                    >{$_("no-account")}
                    <a class="text-blue-500 underline" href="/register"
                        >{$_("make-one")}</a
                    ></span
                >
            {/if}
        {/if}

        {#if authProviders.length}
            {#if page.data.authMethods.usernamePassword || page.data.authMethods.emailPassword}
                <div class="flex gap-4 items-center w-full">
                    <hr class="basis-full border-input-border" />
                    <span class="text-gray-500 uppercase">{$_("or")}</span>
                    <hr class="basis-full border-input-border" />
                </div>
            {/if}
            <div class="w-80 space-y-4">
                {#each authProviders as provider}
                    <a
                        href={(provider as any).url}
                        class="btn-secondary inline-flex min-w-full justify-center"
                        onclick={() => setProvider(provider)}
                    >
                        <img
                            class="w-5 aspect-square mr-4"
                            src={(provider as any).img}
                            alt="Provider logo"
                        />
                        Login with {provider.displayName}
                    </a>
                {/each}
            </div>
        {/if}
    </form>
</main>
