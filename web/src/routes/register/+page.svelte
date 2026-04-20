<script lang="ts">
    import { invalidateAll } from "$app/navigation";
    import { page } from "$app/state";
    import Button from "$lib/components/base/button.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import LogoTextTwoLineDark from "$lib/components/logo/logo_text_two_line_dark.svelte";
    import LogoTextTwoLineLight from "$lib/components/logo/logo_text_two_line_light.svelte";
    import { Language } from "$lib/models/settings";
    import type { User } from "$lib/models/user";
    import { settings_update } from "$lib/stores/settings_store";
    import { theme } from "$lib/stores/theme_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { login, users_create } from "$lib/stores/user_store";
    import { APIError } from "$lib/util/api_util";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { ClientResponseError } from "pocketbase";
    import { _ } from "svelte-i18n";
    import { z } from "zod";

    let loading: boolean = $state(false);
    const { form, errors } = createForm<User>({
        initialValues: {
            id: "",
            username: "",
            email: "",
            password: "",
        },
        extend: validator({
            schema: z.object({
                username: z
                    .string()
                    .min(
                        3,
                        $_("must-be-at-least-n-characters-long", {
                            values: { n: 3 },
                        }),
                    )
                    .regex(/^[\w][\w\.]*$/, $_("invalid-username")),
                email: z
                    .string()
                    .min(1, "required")
                    .email("not-a-valid-email-address"),
                password: z
                    .string()
                    .min(
                        8,
                        $_("must-be-at-least-n-characters-long", {
                            values: { n: 8 },
                        }),
                    )
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
                await users_create(newUser);
            } catch (e) {
                if (
                    e instanceof APIError &&
                    e.detail?.data?.email?.code === "validation_not_unique"
                ) {
                    $errors.email = [$_("email-not-unique")];
                } else if (
                    e instanceof APIError &&
                    e.detail?.data?.username?.code === "validation_not_unique"
                ) {
                    $errors.username = [$_("username-not-unique")];
                } else {
                    show_toast({
                        icon: "close",
                        type: "error",
                        text: $_("error-creating-user"),
                    });
                }

                return;
            } finally {
                loading = false;
            }
            try {
                await login(newUser);
            } catch (e) {
                show_toast({
                    icon: "close",
                    type: "error",
                    text: $_("error-during-login"),
                });
            }
            try {
                await invalidateAll();
                const browserLang: Language = window.navigator.language
                    .toLocaleLowerCase()
                    .slice(0, 2) as Language;
                const language = Object.values(Language).includes(browserLang)
                    ? browserLang
                    : Language.en;
                await settings_update({ id: page.data.settings.id, language });
            } catch (e) {
                console.error(e);
            } finally {
                window.location.href = "/";
                loading = false;
            }
        },
    });
</script>

<svelte:head>
    <title>{$_("register")} | {page.data.appTitle}</title>
</svelte:head>
<main class="flex justify-center">
    <form
        class="login-panel max-w-md border border-input-border rounded-xl p-8 flex flex-col justify-center items-center gap-8 w-[28rem] mt-8"
        use:form
    >
        {#if $theme == "light"}
            <LogoTextTwoLineDark></LogoTextTwoLineDark>
        {:else}
            <LogoTextTwoLineLight></LogoTextTwoLineLight>
        {/if}
        <h4 class="text-xl font-semibold">{$_("slogan")}</h4>
        <div class="space-y-6 w-80">
            <TextField
                name="username"
                label={$_("username")}
                error={$errors.username}
            ></TextField>
            <TextField name="email" label={$_("email")} error={$errors.email}
            ></TextField>
            <TextField
                name="password"
                label={$_("password")}
                type="password"
                error={$errors.password}
            ></TextField>
            <Button
                id="submit"
                primary={true}
                extraClasses={"min-w-full"}
                type="submit"
                {loading}>{$_("register")}</Button
            >
        </div>
        <span
            >{$_("already-account")}
            <a class="text-blue-500 underline" href="/login">{$_("login")}!</a
            ></span
        >
    </form>
</main>
