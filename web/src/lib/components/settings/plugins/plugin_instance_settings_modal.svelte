<script lang="ts">
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import Modal from "$lib/components/base/modal.svelte";
    import Select, { type SelectItem } from "$lib/components/base/select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import PluginMergeSettings from "$lib/components/settings/plugins/plugin_merge_settings.svelte";
    import type { PluginInstance } from "$lib/models/plugin_instance";
    import type { ConfigField, PluginProvider } from "$lib/models/plugin_provider";
    import { plugin_oauth_start } from "$lib/stores/plugin_instance_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        plugin: PluginProvider;
        instance?: PluginInstance;
        onsave?: (instance: Partial<PluginInstance>) => Promise<PluginInstance | void> | PluginInstance | void;
    }

    let { plugin, instance, onsave }: Props = $props();

    let modal: Modal;
    let auth: Record<string, string> = $state(initialAuth());
    let planned = $state(true);
    let completed = $state(true);
    let mergeEnabled = $state(false);
    let privacy = $state("original");
    let authFields = $derived(plugin.auth.fields ?? []);
    let secretFields = $derived(new Set(plugin.auth.secretFields ?? plugin.auth.fields ?? []));
    let isOAuthPlugin = $derived(plugin.auth.type === "oauth2");
    let isConnected = $derived(instance?.status === "configured");
    let needsOAuthConnect = $derived(isOAuthPlugin && (!isConnected || authChanged()));
    let plugin_id = $derived(plugin.id);
    let configSchema = $derived(plugin.configSchema ?? []);
    let extraConfig: Record<string, any> = $state(initialExtraConfig());
    let supportsPlanned = $derived(
        plugin.capabilities?.includes("list_routes.v1") ?? false,
    );
    let supportsCompleted = $derived(
        plugin.capabilities?.includes("list_activities.v1") ?? false,
    );
    let supportsSourcePrivacy = $derived(
        plugin.capabilities?.includes("source_privacy") ?? false,
    );

    const configLabels: Record<string, string> = {
        after: "ignore-trails-before-date",
    };
    const configHints: Record<string, string> = {
        after: "plugin-after-date-hint",
    };
    const privacySelectItems: SelectItem[] = [
        { text: $_("keep-original"), value: "original" },
        { text: $_("apply-user-settings"), value: "settings" },
    ];
    const authLabels: Record<string, string> = {
        email: $_("email"),
        password: $_("password"),
        clientId: "Client ID",
        clientSecret: "Client Secret",
    };

    function initialAuth() {
        return Object.fromEntries(
            (plugin.auth.fields ?? []).map((field) => [
                field,
                (instance?.auth?.[field] as string | undefined) ?? "",
            ]),
        );
    }

    function initialExtraConfig(): Record<string, any> {
        const config = pluginConfig();
        return Object.fromEntries(
            configSchema.map((field) => {
                const saved = config[field.key];
                if (saved !== undefined) return [field.key, saved];
                if (field.default !== undefined) return [field.key, String(field.default)];
                if (field.type === "boolean") {
                    return [field.key, false];
                }
                if (field.type === "select" && field.options?.length) {
                    return [field.key, field.options[0].value];
                }
                if (field.type === "select") {
                    return [field.key, ""];
                }
                return [field.key, undefined];
            }),
        );
    }

    function selectItems(field: ConfigField): SelectItem[] {
        return (field.options ?? []).map((o) => ({
            text: $_(o.value),
            value: o.value,
        }));
    }

    function fieldLabel(field: ConfigField): string {
        if (field.label) {
            return field.label;
        }
        return $_(configLabels[field.key] ?? field.key);
    }

    function fieldHint(field: ConfigField): string | undefined {
        if (field.description) {
            return field.description;
        }
        const hint = configHints[field.key];
        return hint ? $_(hint) : undefined;
    }

    function authLabel(field: string): string {
        return authLabels[field] ?? field;
    }

    function authFieldType(field: string): "password" | "text" {
        return secretFields.has(field) ? "password" : "text";
    }

    function configSection(key: string): Record<string, any> {
        const section = instance?.config?.[key];
        if (section && typeof section === "object" && !Array.isArray(section)) {
            return section as Record<string, any>;
        }
        return {};
    }

    function pluginConfig(): Record<string, any> {
        return configSection("plugin");
    }

    function hostConfig(): Record<string, any> {
        return configSection("host");
    }

    function authChanged() {
        for (const field of authFields) {
            const value = auth[field] ?? "";
            if (secretFields.has(field)) {
                if (value !== "") {
                    return true;
                }
                continue;
            }
            if (value !== ((instance?.auth?.[field] as string | undefined) ?? "")) {
                return true;
            }
        }
        return false;
    }

    export function openModal() {
        auth = initialAuth();
        const config = hostConfig();
        planned = (config.planned as boolean | undefined) ?? true;
        completed = (config.completed as boolean | undefined) ?? true;
        mergeEnabled = Boolean((config.merge as any)?.enabled);
        privacy = (config.privacy as string | undefined) ?? "original";
        extraConfig = initialExtraConfig();
        modal.openModal();
    }

    function pluginInstanceFromForm(): Partial<PluginInstance> {
        const submittedAuth: Record<string, string> = {};
        for (const field of authFields) {
            submittedAuth[field] = auth[field] ?? "";
        }

        const pluginRuntimeConfig: Record<string, unknown> = { ...pluginConfig() };
        const pluginHostConfig: Record<string, unknown> = { ...hostConfig() };
        if (supportsPlanned) {
            pluginHostConfig.planned = planned;
        }
        if (supportsCompleted) {
            pluginHostConfig.completed = completed;
        }
        if (supportsSourcePrivacy) {
            pluginHostConfig.privacy = privacy;
        }
        pluginHostConfig.merge = { enabled: mergeEnabled };
        for (const field of configSchema) {
            const val = extraConfig[field.key];
            if (val !== undefined && val !== "") {
                pluginRuntimeConfig[field.key] = val;
            } else {
                delete pluginRuntimeConfig[field.key];
            }
        }
        const config: Record<string, unknown> = {
            plugin: pluginRuntimeConfig,
            host: pluginHostConfig,
        };

        const status = isOAuthPlugin
            ? !instance?.id
                ? "needs_auth"
                : authChanged()
                  ? isConnected
                      ? "needs_reauth"
                      : "needs_auth"
                  : instance.status
            : instance?.enabled
              ? "configured"
              : "disabled";

        return {
            ...instance,
            plugin_id,
            enabled: isOAuthPlugin && status !== "configured" ? false : (instance?.enabled ?? false),
            auth: submittedAuth,
            config,
            status,
        };
    }

    function submit() {
        onsave?.(pluginInstanceFromForm());
        modal.closeModal();
    }

    async function startOAuth() {
        try {
            const saved = await onsave?.(pluginInstanceFromForm());
            const instanceId = saved?.id ?? instance?.id;
            if (!instanceId) {
                return;
            }
            const redirectUri = `${window.location.origin}/settings/plugins/oauth/callback`;
            const result = await plugin_oauth_start({
                pluginId: plugin.id,
                instanceId,
                redirectUri,
            });
            sessionStorage.setItem(`wanderer_plugin_oauth_${result.state}`, result.instanceId);
            window.location.href = result.url;
        } catch (e) {
            show_toast({
                text: e instanceof Error ? e.message : $_("error-starting-oauth"),
                icon: "close",
                type: "error",
            });
        }
    }
</script>

<Modal
    id="{plugin_id}-plugin-settings-modal"
    size="md:max-w-lg"
    title={plugin.name + " " + $_("settings")}
    bind:this={modal}
>
    {#snippet content()}
        <form
            id="{plugin_id}-plugin-settings-form"
            class="space-y-3"
            onsubmit={(event) => {
                event.preventDefault();
                submit();
            }}
        >
            {#each authFields as field}
                <TextField
                    label={authLabel(field)}
                    placeholder={secretFields.has(field)
                        ? instance?.id
                            ? `(${$_("unchanged")})`
                            : ""
                        : field == "email"
                          ? "user@example.com"
                          : ""}
                    bind:value={auth[field]}
                    name={field}
                    type={authFieldType(field)}
                ></TextField>
            {/each}

            {#if isOAuthPlugin}
                <p class="text-xs text-gray-500 max-w-lg">
                    {#if isConnected}
                        {$_("plugin-oauth-connected-hint")}
                    {:else}
                        {$_("plugin-oauth-needs-connect-hint")}
                    {/if}
                </p>
            {/if}

            {#if supportsPlanned || supportsCompleted}
                <div class="flex flex-wrap gap-x-4">
                    {#if supportsPlanned}
                        <Toggle
                            bind:value={planned}
                            label={$_("planned-tours", { values: { n: 2 } })}
                        ></Toggle>
                    {/if}
                    {#if supportsCompleted}
                        <Toggle
                            bind:value={completed}
                            label={$_("completed-tours", { values: { n: 2 } })}
                        ></Toggle>
                    {/if}
                </div>
            {/if}

            {#if supportsSourcePrivacy}
                <Select
                    label={$_("privacy")}
                    items={privacySelectItems}
                    bind:value={privacy}
                ></Select>
                <p class="text-xs text-gray-500 max-w-lg">
                    {#if privacy == "original"}
                        {$_("plugin-privacy-hint-original")}
                    {:else}
                        {$_("plugin-privacy-hint-user")}
                    {/if}
                </p>
            {/if}

            {#each configSchema as field}
                {#if field.type === "select"}
                    <Select
                        label={fieldLabel(field)}
                        items={selectItems(field)}
                        bind:value={extraConfig[field.key] as string}
                    ></Select>
                {:else if field.type === "boolean"}
                    <Toggle
                        bind:value={extraConfig[field.key]}
                        label={fieldLabel(field)}
                    ></Toggle>
                    {@const hint = fieldHint(field)}
                    {#if hint}
                        <p class="text-xs text-gray-500 max-w-lg">{hint}</p>
                    {/if}
                {:else if field.type === "date"}
                    {@const hint = fieldHint(field)}
                    {#if hint}
                        <p
                            class="text-xs text-gray-500 max-w-lg pt-4 pb-1 border-t border-input-border"
                        >
                            {hint}
                        </p>
                    {/if}
                    <div class="flex items-end relative gap-x-2">
                        <Datepicker
                            label={fieldLabel(field)}
                            bind:value={extraConfig[field.key]}
                        ></Datepicker>
                        <button
                            class="btn-icon mb-[10px]"
                            type="button"
                            onclick={() => {
                                extraConfig[field.key] = undefined;
                            }}
                            aria-label={$_("clear")}
                        ><i class="fa fa-close"></i></button>
                    </div>
                {:else if field.type === "text" || field.type === "url"}
                    <TextField
                        label={fieldLabel(field)}
                        bind:value={extraConfig[field.key]}
                        name={field.key}
                        type={field.type === "url" ? "url" : "text"}
                    ></TextField>
                {/if}
            {/each}

            <PluginMergeSettings prefix="merge" bind:value={mergeEnabled} />
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
            {#if isOAuthPlugin}
                {#if needsOAuthConnect}
                    <button class="btn-primary" type="button" onclick={startOAuth}
                        >{isConnected ? $_("save-and-reconnect") : $_("save-and-connect")}</button
                    >
                {:else}
                    <button
                        class="btn-primary"
                        form="{plugin_id}-plugin-settings-form"
                        type="submit"
                        name="save">{$_("save")}</button
                    >
                {/if}
            {:else}
                <button
                    class="btn-primary"
                    form="{plugin_id}-plugin-settings-form"
                    type="submit"
                    name="save">{$_("save")}</button
                >
            {/if}
        </div>
    {/snippet}
</Modal>
