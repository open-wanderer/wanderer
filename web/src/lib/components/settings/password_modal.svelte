<script lang="ts">
    import Modal from "#lib/components/base/modal.svelte";
    import { _ } from "svelte-i18n";
    import TextField from "../base/text_field.svelte";
    import { validator } from "#lib/vendor/felte-validator-zod/index.ts";
    import { createForm } from "felte";
    import { z } from "zod";

    interface Props {
        onsave?: (data: {
            oldPassword: string;
            password: string;
            passwordConfirm: string;
        }) => void;
    }

    let { onsave }: Props = $props();

    let modal: Modal;
    export function openModal() {
        setFields("password", "");
        setFields("oldPassword", "");
        setErrors("password", []);
        setErrors("oldPassword", []);
        modal.openModal!();
    }

    const { form, errors, setFields, setErrors } = createForm<{
        oldPassword: string;
        password: string;
        passwordConfirm: string;
    }>({
        initialValues: { oldPassword: "", password: "" },
        extend: validator({
            schema: z
                .object({
                    oldPassword: z
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
            onsave?.({
                oldPassword: form.oldPassword,
                password: form.password,
                passwordConfirm: form.password,
            });
            modal.closeModal();
        },
    });
</script>

<Modal
    id="password-modal"
    size="md:min-w-sm"
    title={$_("change-password")}
    bind:this={modal}
>
    {#snippet content()}
        <form id="password-form" use:form>
            <TextField
                name="oldPassword"
                label={$_("current-password")}
                type="password"
                error={$errors.oldPassword}
            ></TextField>
            <TextField
                name="password"
                label={$_("new-password")}
                type="password"
                error={$errors.password}
            ></TextField>
            <TextField
                name="passwordConfirm"
                label={$_("password-confirm")}
                type="password"
                error={$errors.passwordConfirm}
            ></TextField>
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
            <button
                class="btn-primary"
                type="submit"
                form="password-form"
                name="save">{$_("save")}</button
            >
        </div>
    {/snippet}</Modal
>
