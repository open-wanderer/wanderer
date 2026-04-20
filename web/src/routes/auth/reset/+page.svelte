<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import Button from "$lib/components/base/button.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import LogoTextTwoLineDark from "$lib/components/logo/logo_text_two_line_dark.svelte";
    import LogoTextTwoLineLight from "$lib/components/logo/logo_text_two_line_light.svelte";
    import { theme } from "$lib/stores/theme_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { users_reset_password } from "$lib/stores/user_store";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";
    import { z } from "zod";

    let loading: boolean = $state(false);

    const { form, errors } = createForm({
        initialValues: {
            email: "",
        },
        extend: validator({
            schema: z.object({
                email: z
                    .string()
                    .min(1, "required")
                    .email("not-a-valid-email-address"),
            }),
        }),
        onSubmit: async (email) => {
            loading = true;
            try {
                await users_reset_password(email);
                goto("/");
                show_toast({
                    icon: "check",
                    type: "success",
                    text: $_("password-reset-sent"),
                });
            } catch (e) {
                show_toast({
                    icon: "close",
                    type: "error",
                    text: $_("error-during-password-reset"),
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
        <h4 class="text-xl font-semibold">{$_("forgot-your-password")}</h4>
        <p class="text-gray-500 w-80">{$_('password-reset-text')}</p>
        <div class="space-y-6 w-80">
            <TextField
                name="email"
                label={$_("email")}
                error={$errors.email}
            ></TextField>
            <div class="grid grid-cols-2 gap-x-4">
                <a class="btn-secondary text-center" href="/login"
                    >{$_("back-to-login")}</a
                >
                <Button
                    primary={true}
                    extraClasses={"min-w-full"}
                    type="submit"
                    {loading}>{$_("reset-password")}</Button
                >
            </div>
        </div>
    </form>
</main>
