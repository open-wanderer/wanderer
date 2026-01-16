<script lang="ts">
    import { page } from "$app/state";
    import IntegrationCard from "$lib/components/settings/integrations/integration_card.svelte";
    import KomootSettingsModal from "$lib/components/settings/integrations/komoot_settings_modal.svelte";
    import StravaSettingsModal from "$lib/components/settings/integrations/strava_settings_modal.svelte";
    import {
        Integration,
        type KomootIntegration,
        type StravaIntegration
    } from "$lib/models/integration.js";
    import {
        integrations_create,
        integrations_update,
    } from "$lib/stores/integration_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import { _ } from "svelte-i18n";

    let { data } = $props();

    let integration = $state(data.integration);

    const scope = "read_all,activity:read_all";
    const redirectUri = page.url.href + "/callback/strava";

    let stravaSettingsModal: StravaSettingsModal;
    let stravaToggleValue: boolean = $derived(
        integration?.strava?.active || false,
    );

    let komootSettingsModal: KomootSettingsModal;
    let komootToggleValue: boolean = $state(
        data.integration?.komoot?.active ?? false,
    );

    async function onSettingsSave(
        form: StravaIntegration | KomootIntegration,
        key: "strava" | "komoot",
    ) {
        try {
            if (integration) {
                integration[key] = form as any;
                integration = await integrations_update(integration);
            } else {
                const newIntegration: Integration = {
                    user: "",
                    [key]: form,
                };
                integration = await integrations_create(newIntegration);
            }

            show_toast({
                text: $_("settings-saved"),
                icon: "check",
                type: "success",
            });
        } catch (e) {
            show_toast({
                text: $_("error-setting-up-integration", {
                    values: { provider: key },
                }),
                icon: "close",
                type: "error",
            });
        }
    }

    async function onStravaToggle(value: boolean) {
        if (!integration?.strava) {
            return;
        }
        if (value) {
            const authUrl = `https://www.strava.com/oauth/authorize?client_id=${integration.strava.clientId}&response_type=code&redirect_uri=${redirectUri}&scope=${scope}&approval_prompt=auto`;
            window.location.href = authUrl;
        } else {
            const deauthUrl = `https://www.strava.com/oauth/deauthorize`;

            await fetch(deauthUrl, {
                method: "POST",
                headers: {
                    Authorization: `Bearer ${integration.strava.accessToken}`,
                },
            });

            integration.strava = {
                clientId: integration.strava.clientId,
                clientSecret: integration.strava.clientSecret,
                routes: integration.strava.routes,
                activities: integration.strava.activities,
                privacy: integration.strava.privacy,
                accessToken: undefined,
                refreshToken: undefined,
                expiresAt: undefined,
                active: false,
            };
            try {
                integration = await integrations_update(integration);
            } catch (e) {
                show_toast({
                    text: $_("error-disabling-strava-integration"),
                    icon: "close",
                    type: "error",
                });
            }

            show_toast({
                text: "strava " + $_("integration-disabled"),
                icon: "check",
                type: "success",
            });
        }
    }

    async function onKomootToggle(value: boolean) {
        if (!integration?.komoot) {
            return;
        }
        if (value) {
            try {
                const r = await fetch("/api/v1/integration/komoot/login", {
                    method: "GET",
                });

                if (!r.ok) {
                    throw Error();
                }
            } catch (e) {
                komootToggleValue = false;
                show_toast({
                    text: $_("error-logging-in-to-komoot"),
                    icon: "close",
                    type: "error",
                });
                return;
            }
            integration.komoot.active = true;
        } else {
            integration.komoot.active = false;
        }
        try {
            integration = await integrations_update(integration);
        } catch (e) {
            show_toast({
                text: $_("error-updating-strava-integration"),
                icon: "close",
                type: "error",
            });
            return;
        }

        show_toast({
            text:
                "komoot " + $_(`integration-${value ? "enabled" : "disabled"}`),
            icon: "check",
            type: "success",
        });
    }
</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>

<h3 class="text-2xl font-semibold">{$_("integrations")}</h3>
<hr class="mt-4 mb-6 border-input-border" />

<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
    <IntegrationCard
        img="https://upload.wikimedia.org/wikipedia/commons/c/cb/Strava_Logo.svg"
        title="strava"
        description={$_("integration-description-strava")}
        disabled={!integration?.strava}
        active={stravaToggleValue}
        onclick={() => stravaSettingsModal.openModal()}
        ontoggle={onStravaToggle}
    ></IntegrationCard>
    <IntegrationCard
        img="https://upload.wikimedia.org/wikipedia/commons/8/82/Komoot-logo-type.svg"
        title="komoot"
        description={$_("integration-description-komoot")}
        disabled={!integration?.komoot}
        bind:active={komootToggleValue}
        onclick={() => komootSettingsModal.openModal()}
        ontoggle={onKomootToggle}
    ></IntegrationCard>
</div>

<StravaSettingsModal
    bind:this={stravaSettingsModal}
    {integration}
    onsave={(form) => onSettingsSave(form, "strava")}
></StravaSettingsModal>

<KomootSettingsModal
    bind:this={komootSettingsModal}
    {integration}
    onsave={(form) => onSettingsSave(form, "komoot")}
></KomootSettingsModal>
