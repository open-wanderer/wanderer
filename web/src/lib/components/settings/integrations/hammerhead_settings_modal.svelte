<script lang="ts">
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import Modal from "$lib/components/base/modal.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import { HammerheadSchema } from "$lib/models/api/integration_schema";
    import type {
        Integration,
        HammerheadIntegration,
    } from "$lib/models/integration";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";

    interface Props {
        integration?: Integration;
        onsave?: (hammerheadIntegration: HammerheadIntegration) => void;
    }

    let { integration, onsave }: Props = $props();

    let modal: Modal;

    export function openModal() {
        errors.set({});
        modal.openModal();
    }

    const getInitialFormValues = () => ({
        email: integration?.hammerhead?.email ?? "",
        password: integration?.hammerhead?.password ?? "",
        completed: integration?.hammerhead?.completed ?? true,
        planned: integration?.hammerhead?.planned ?? true,
        active: integration?.hammerhead?.active ?? false,
        after: integration?.hammerhead?.after,
    });

    const {
        form,
        errors,
        data: formData,
    } = createForm({
        initialValues: getInitialFormValues(),
        extend: validator({
            schema: HammerheadSchema,
        }),
        onSubmit: async (form) => {
            form.active = integration?.hammerhead?.active ?? form.active;
            onsave?.(form);
            modal.closeModal();
        },
    });

    function clearAfterDate() {
        ($formData as any).after = undefined;
    }
</script>

<Modal
    id="hammerhead-settings-modal"
    size="md:max-w-lg"
    title={"Hammerhead " + $_("settings")}
    bind:this={modal}
>
    {#snippet content()}
        <form id="hammerhead-settings-form" class="space-y-2" use:form>
            <TextField
                label={$_("email")}
                placeholder="user@example.com"
                name="email"
                error={$errors.email}
            ></TextField>
            <TextField
                label={$_("password")}
                placeholder={integration?.hammerhead ? `(${$_("unchanged")})` : ""}
                name="password"
                type="password"
                error={$errors.password}
            ></TextField>
            <div class="flex flex-wrap gap-x-4">
                <Toggle name="planned" label={$_("planned-tours", { values: { n: 2 } })}
                ></Toggle>
                <Toggle
                    name="completed"
                    label={$_("completed-tours", { values: { n: 2 } })}
                ></Toggle>
            </div>
            <p
                class="text-xs text-gray-500 max-w-lg pt-4 pb-1 border-t border-input-border"
            >
                {$_("hammerhead-integration-after-date-hint")}
            </p>
            <div class="flex items-end relative gap-x-2">
                <Datepicker
                    error={$errors.after}
                    label={$_("ignore-trails-before-date")}
                    bind:value={$formData.after}
                ></Datepicker>
                <button
                    class="btn-icon mb-[10px]"
                    type="button"
                    onclick={clearAfterDate}
                    aria-label="Clear 'after' date"
                    ><i class="fa fa-close"></i></button
                >
            </div>
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
            <button
                class="btn-primary"
                form="hammerhead-settings-form"
                type="submit"
                name="save">{$_("save")}</button
            >
        </div>
    {/snippet}</Modal
>
