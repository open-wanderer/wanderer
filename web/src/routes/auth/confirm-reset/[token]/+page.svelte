<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import Button from "$lib/components/base/button.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import LogoTextTwoLineDark from "$lib/components/logo/logo_text_two_line_dark.svelte";
    import LogoTextTwoLineLight from "$lib/components/logo/logo_text_two_line_light.svelte";
    import { theme } from "$lib/stores/theme_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { users_confirm_reset } from "$lib/stores/user_store";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";
    import { z } from "zod";

    let loading: boolean = $state(false);

    const { form, errors } = createForm({
        initialValues: {
            password: "",
            passwordConfirm: "",
        },
        extend: validator({
            schema: z
                .object({
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
                    passwordConfirm: z
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
                })
                .refine((d) => d.password === d.passwordConfirm, {
                    message: "passwords-must-match",
                    path: ["passwordConfirm"],
                }),
        }),
        onSubmit: async (form) => {
            loading = true;
            try {
                await users_confirm_reset({
                    ...form,
                    token: page.params.token!,
                });
                goto("/login");
                show_toast({
                    icon: "check",
                    type: "success",
                    text: $_("new-password-success"),
                });
            } catch (e) {
                show_toast({
                    icon: "close",
                    type: "error",
                    text: $_("new-password-error"),
                });
            } finally {
                loading = false;
            }
        },
    });
</script>

<svelte:head>
    <title>{$_("login")} | {page.data.appTitle}</title>
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
        <h4 class="text-xl font-semibold">{$_("new-password-text")}</h4>
        <div class="space-y-6 w-80">
            <TextField
                name="password"
                label={$_("password")}
                type="password"
                error={$errors.password}
            ></TextField>
            <TextField
                name="passwordConfirm"
                label={$_("password-confirm")}
                type="password"
                error={$errors.passwordConfirm}
            ></TextField>

            <Button
                primary={true}
                extraClasses={"min-w-full"}
                type="submit"
                {loading}>{$_("reset-password")}</Button
            >
        </div>
    </form>
</main>
