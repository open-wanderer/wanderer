<script lang="ts">
    import PluginCard from "$lib/components/settings/plugins/plugin_card.svelte";
    import PluginInstanceSettingsModal from "$lib/components/settings/plugins/plugin_instance_settings_modal.svelte";
    import type { Category } from "$lib/models/category.js";
    import type { PluginInstance } from "$lib/models/plugin_instance.js";
    import type { PluginProvider } from "$lib/models/plugin_provider.js";
    import {
        plugin_instances_create,
        plugin_instances_update,
    } from "$lib/stores/plugin_instance_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import {
        pluginDescription as localizedPluginDescription,
        pluginTitle as localizedPluginTitle,
    } from "$lib/util/plugin_i18n";
    import { onMount, tick, untrack } from "svelte";
    import { _, locale } from "svelte-i18n";
    import { theme } from "$lib/stores/theme_store";

    let { data } = $props();

    let instances: PluginInstance[] = $state(
        untrack(() => data.pluginInstances ?? []),
    );
    let plugins: PluginProvider[] = $state(
        untrack(() => data.pluginProviders ?? []),
    );
    let categories: Category[] = $state(untrack(() => data.categories ?? []));

    let pluginSettingsModal: PluginInstanceSettingsModal | undefined = $state();
    let selectedPlugin: PluginProvider | undefined = $state();
    let currentTheme: "dark" | "light" = $state("light");
    let pluginGroups = $derived.by(() => {
        const groups: { type: PluginProvider["type"]; plugins: PluginProvider[] }[] = [];
        for (const plugin of plugins) {
            let group = groups.find((candidate) => candidate.type === plugin.type);
            if (!group) {
                group = { type: plugin.type, plugins: [] };
                groups.push(group);
            }
            group.plugins.push(plugin);
        }
        return groups;
    });

    onMount(() => {
        currentTheme = document.documentElement.classList.contains("dark") ? "dark" : "light";
        return theme.subscribe((value) => {
            currentTheme = value;
        });
    });

    async function savePluginInstance(instance: Partial<PluginInstance>) {
        try {
            let saved: PluginInstance;
            if (instance.id) {
                saved = await plugin_instances_update(
                    instance as PluginInstance,
                );
            } else {
                saved = await plugin_instances_create(instance);
            }

            instances = [
                ...instances.filter((existing) => existing.id != saved.id),
                saved,
            ];

            show_toast({
                text: $_("settings-saved"),
                icon: "check",
                type: "success",
            });
            return saved;
        } catch (e) {
            show_toast({
                text: $_("error-setting-up-plugin", {
                    values: { provider: instance.plugin_id },
                }),
                icon: "close",
                type: "error",
            });
            throw e;
        }
    }

    async function onPluginToggle(
        plugin: PluginProvider,
        instance: PluginInstance | undefined,
        value: boolean,
    ) {
        if (!instance) {
            return;
        }
        if (value && pluginRequiresConnection(plugin, instance)) {
            show_toast({
                text: $_("plugin-connect-before-enabling"),
                icon: "close",
                type: "error",
            });
            return;
        }

        try {
            const saved = await plugin_instances_update({
                ...instance,
                enabled: value,
                status: plugin.auth.type === "oauth2" || value ? "configured" : "disabled",
            });
            instances = [
                ...instances.filter((existing) => existing.id != saved.id),
                saved,
            ];
        } catch (e) {
            show_toast({
                text: $_("error-setting-up-plugin", { values: { provider: pluginTitle(plugin) } }),
                icon: "close",
                type: "error",
            });
            return;
        }

        show_toast({
            text: pluginTitle(plugin) + " " + $_(`plugin-${value ? "enabled" : "disabled"}`),
            icon: "check",
            type: "success",
        });
    }

    function instanceError(instance: PluginInstance | undefined) {
        return instance?.last_error?.message ?? "";
    }

    function instanceForPlugin(plugin: PluginProvider) {
        return instances.find((instance) => instance.plugin_id == plugin.id);
    }

    async function openPluginSettings(plugin: PluginProvider) {
        selectedPlugin = plugin;
        await tick();
        pluginSettingsModal?.openModal();
    }

    function pluginLogo(plugin: PluginProvider) {
        if (currentTheme === "dark" && plugin.iconDark) {
            return plugin.iconDark;
        }
        return plugin.icon || undefined;
    }

    function pluginTitle(plugin: PluginProvider) {
        return localizedPluginTitle(plugin, $locale);
    }

    function pluginDescription(plugin: PluginProvider) {
        return localizedPluginDescription(plugin, $locale);
    }

    function pluginTypeTitle(type: PluginProvider["type"]) {
        return $_(`plugin-type-${type}`);
    }

    function pluginTypeDescription(type: PluginProvider["type"]) {
        return $_(`plugin-type-${type}-description`);
    }

    function pluginCardError(
        plugin: PluginProvider,
        instance: PluginInstance | undefined,
    ) {
        if (plugin.status != "available") {
            return plugin.error ?? "";
        }
        return instanceError(instance);
    }

    function pluginRequiresConnection(
        plugin: PluginProvider,
        instance: PluginInstance | undefined,
    ) {
        return plugin.auth.type === "oauth2" && instance?.status !== "configured";
    }

</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>

<h3 class="text-2xl font-semibold">{$_("plugins")}</h3>
<hr class="mt-4 mb-6 border-input-border" />

<div class="space-y-8">
    {#each pluginGroups as group (group.type)}
        <section>
            <div class="mb-4 space-y-2">
                <h4 class="text-xl font-medium">{pluginTypeTitle(group.type)}</h4>
                <p class="text-sm text-gray-500 max-w-3xl">
                    {pluginTypeDescription(group.type)}
                </p>
            </div>
            <div class="space-y-3">
                {#each group.plugins as plugin (plugin.id)}
                    {@const instance = instanceForPlugin(plugin)}
                    <PluginCard
                        img={pluginLogo(plugin)}
                        title={pluginTitle(plugin)}
                        description={pluginDescription(plugin)}
                        disabled={!instance || plugin.status != "available" || pluginRequiresConnection(plugin, instance)}
                        active={instance?.enabled ?? false}
                        lastSyncAt={instance?.last_sync_at}
                        error={pluginCardError(plugin, instance)}
                        onclick={() => openPluginSettings(plugin)}
                        ontoggle={(value) => onPluginToggle(plugin, instance, value)}
                    ></PluginCard>
                {/each}
            </div>
        </section>
    {/each}
</div>

{#if selectedPlugin}
    {#key selectedPlugin.id}
        <PluginInstanceSettingsModal
            bind:this={pluginSettingsModal}
            plugin={selectedPlugin}
            instance={instanceForPlugin(selectedPlugin)}
            categories={categories}
            onsave={savePluginInstance}
        ></PluginInstanceSettingsModal>
    {/key}
{/if}
