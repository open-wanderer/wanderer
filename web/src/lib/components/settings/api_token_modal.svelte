<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import { APITokenCreateSchema } from "$lib/models/api/api_token_schema";
    import type { APIToken } from "$lib/models/api_token";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";
    import { z } from "zod";
    import Datepicker from "../base/datepicker.svelte";
    import TextField from "../base/text_field.svelte";
    import Toggle from "../base/toggle.svelte";

    interface Props {
        onsave?: (token: APIToken) => void;
    }

    let { onsave }: Props = $props();

    let modal: Modal;

    let expires: boolean = $state(false);

    const today = new Date();
    today.setDate(today.getDate() + 1);
    const tomorrow = today.toISOString().split("T")[0];

    export function openModal() {
        modal.openModal();
        setFields("name", "");
        setFields("expiration", undefined);
        setErrors("name", []);
        setErrors("expiration", []);
    }

    const { form, errors, setFields, setErrors } = createForm<
        z.infer<typeof APITokenCreateSchema>
    >({
        initialValues: {
            name: "",
            expiration: "",
        },
        extend: validator({
            schema: APITokenCreateSchema.extend({
                user: z.string().optional(),
            }),
        }),
        onSubmit: async (form) => {
            onsave?.(form);
            modal.closeModal!();
        },
    });
</script>

<Modal
    id="api-token-modal"
    size="md:min-w-md"
    title={$_("generate-new-token")}
    bind:this={modal}
>
    {#snippet content()}
        <form class="space-y-4" id="api-token-form" use:form>
            <TextField label={$_("name")} name="name" error={$errors.name}
            ></TextField>
            <Toggle label={$_("expires")} bind:value={expires}></Toggle>
            {#if expires}
                <Datepicker
                    label={$_("expiration")}
                    name="expiration"
                    error={$errors.expiration}
                    min={tomorrow}
                ></Datepicker>
            {/if}
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
                form="api-token-form"
                name="save">{$_("save")}</button
            >
        </div>
    {/snippet}</Modal
>
