<script lang="ts">
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import Modal from "$lib/components/base/modal.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import { StravaSchema } from "$lib/models/api/integration_schema";
    import type {
        Integration,
        StravaIntegration,
    } from "$lib/models/integration";
    import { validator } from "@felte/validator-zod";
    import { createForm } from "felte";
    import { _ } from "svelte-i18n";

    interface Props {
        integration?: Integration;
        onsave?: (stravaIntegration: StravaIntegration) => void;
    }

    let { integration, onsave }: Props = $props();

    let modal: Modal;

    export function openModal() {
        errors.set({});
        modal.openModal();
    }

    const {
        form,
        errors,
        data: formData,
    } = createForm({
        initialValues: {
            clientId: integration?.strava?.clientId ?? "",
            clientSecret: integration?.strava?.clientSecret ?? "",
            routes: integration?.strava?.routes ?? true,
            activities: integration?.strava?.activities ?? true,
            active: integration?.strava?.active ?? false,
            after: integration?.strava?.after,
        },
        extend: validator({
            schema: StravaSchema,
        }),
        onSubmit: async (form) => {
            form.active = integration?.strava?.active ?? form.active;
            onsave?.(form);
            modal.closeModal();
        },
    });

    function clearAfterDate() {
        ($formData as any).after = undefined;
    }
</script>

<Modal
    id="strava-settings-modal"
    size="md:min-w-lg"
    title={"Strava " + $_("settings")}
    bind:this={modal}
>
    {#snippet content()}
        <form id="strava-settings-form" class="space-y-2" use:form>
            <TextField
                label="Client-ID"
                placeholder="000000"
                name="clientId"
                error={$errors.clientId}
            ></TextField>
            <TextField
                label="Client Secret"
                placeholder={integration?.strava
                    ? `(${$_("unchanged")})`
                    : "de8b3789bd7116d..."}
                name="clientSecret"
                type="password"
                error={$errors.clientSecret}
            ></TextField>
            <div class="flex gap-x-4">
                <Toggle name="routes" label={$_("route", { values: { n: 2 } })}
                ></Toggle>
                <Toggle
                    name="activities"
                    label={$_("activity", { values: { n: 2 } })}
                ></Toggle>
            </div>
            <p
                class="text-xs text-gray-500 max-w-lg pt-4 pb-1 border-t border-input-border"
            >
                {$_("strava-integration-after-date-hint")}
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
                form="strava-settings-form"
                type="submit"
                name="save">{$_("save")}</button
            >
        </div>
    {/snippet}</Modal
>
