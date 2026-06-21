<script lang="ts">
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import PluginCard from "$lib/components/settings/plugins/plugin_card.svelte";
    import PluginInstanceSettingsModal from "$lib/components/settings/plugins/plugin_instance_settings_modal.svelte";
    import type { Category } from "$lib/models/category.js";
    import type { PluginInstance } from "$lib/models/plugin_instance.js";
    import type { PluginProvider } from "$lib/models/plugin_provider.js";
    import {
        plugin_category_remap_apply,
        plugin_category_remap_preview,
        plugin_instances_create,
        plugin_instances_update,
    } from "$lib/stores/plugin_instance_store.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import {
        pluginDescription as localizedPluginDescription,
        pluginTitle as localizedPluginTitle,
    } from "$lib/util/plugin_i18n";
    import { translatePluginError } from "$lib/util/plugin_error_i18n";
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
    let categoryRemapConfirmModal: ConfirmModal | undefined = $state();
    let selectedPlugin: PluginProvider | undefined = $state();
    let pendingCategoryRemap:
        | {
              instanceId: string;
              pluginTitle: string;
              count: number;
              backfilledSinceMapping: number;
              backfilledOnly: boolean;
              beforeSave?: boolean;
              onresolve?: (confirmed: boolean) => void;
          }
        | undefined = $state();
    let currentTheme: "dark" | "light" = $state("light");
    const applyRemapAfterSave = new Set<string>();
    const suppressRemapPromptAfterSave = new Set<string>();
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
        queueMicrotask(() => {
            void maybePromptBackfilledCategoryRemap();
        });
        return theme.subscribe((value) => {
            currentTheme = value;
        });
    });

    async function savePluginInstance(instance: Partial<PluginInstance>) {
        let saved: PluginInstance;
        try {
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

        if (saved.id && applyRemapAfterSave.has(saved.id)) {
            applyRemapAfterSave.delete(saved.id);
            try {
                await applyCategoryRemap(saved.id);
            } catch (e) {
                show_toast({
                    text: $_("plugin-category-remap-error"),
                    icon: "close",
                    type: "error",
                });
            }
        } else if (saved.id && suppressRemapPromptAfterSave.has(saved.id)) {
            suppressRemapPromptAfterSave.delete(saved.id);
        }
        return saved;
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
            if (value) {
                await maybePromptCategoryRemap(saved, plugin);
            }
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

    async function maybePromptCategoryRemap(
        instance: PluginInstance,
        plugin: PluginProvider | undefined,
        options: { backfilledOnly?: boolean } = {},
    ) {
        if (!instance.id || !plugin) {
            return;
        }
        try {
            const preview = await plugin_category_remap_preview(instance.id);
            const backfilledSinceMapping = preview.backfilledSinceMapping ?? 0;
            if (preview.count <= 0 || (options.backfilledOnly && backfilledSinceMapping <= 0)) {
                return;
            }
            pendingCategoryRemap = {
                instanceId: instance.id,
                pluginTitle: pluginTitle(plugin),
                count: preview.count,
                backfilledSinceMapping,
                backfilledOnly: options.backfilledOnly ?? false,
            };
            await tick();
            categoryRemapConfirmModal?.openModal();
        } catch (e) {
            show_toast({
                text: $_("plugin-category-remap-preview-error"),
                icon: "close",
                type: "error",
            });
        }
    }

    async function applyCategoryRemap(instanceId: string) {
        const result = await plugin_category_remap_apply(instanceId);
        show_toast({
            text: $_("plugin-category-remap-success", {
                values: { count: result.remapped ?? 0 },
            }),
            icon: "check",
            type: "success",
        });
    }

    async function confirmCategoryRemap() {
        if (!pendingCategoryRemap) {
            return;
        }
        if (pendingCategoryRemap.beforeSave) {
            if (pendingCategoryRemap.count > 0) {
                applyRemapAfterSave.add(pendingCategoryRemap.instanceId);
            } else {
                suppressRemapPromptAfterSave.add(pendingCategoryRemap.instanceId);
            }
            pendingCategoryRemap.onresolve?.(true);
            pendingCategoryRemap = undefined;
            return;
        }
        try {
            await applyCategoryRemap(pendingCategoryRemap.instanceId);
        } catch (e) {
            show_toast({
                text: $_("plugin-category-remap-error"),
                icon: "close",
                type: "error",
            });
        } finally {
            pendingCategoryRemap = undefined;
        }
    }

    function continueWithoutCategoryRemap() {
        if (!pendingCategoryRemap?.beforeSave) {
            return;
        }
        suppressRemapPromptAfterSave.add(pendingCategoryRemap.instanceId);
        pendingCategoryRemap.onresolve?.(true);
        pendingCategoryRemap = undefined;
    }

    async function dismissCategoryRemap() {
        if (!pendingCategoryRemap) {
            return;
        }
        if (pendingCategoryRemap.beforeSave) {
            pendingCategoryRemap.onresolve?.(false);
            pendingCategoryRemap = undefined;
            return;
        }
        const instance = instances.find(
            (candidate) => candidate.id === pendingCategoryRemap?.instanceId,
        );
        if (!instance || !pendingCategoryRemap.backfilledOnly) {
            pendingCategoryRemap = undefined;
            return;
        }

        try {
            const saved = await plugin_instances_update({
                ...instance,
                config: {
                    ...(instance.config ?? {}),
                    host: {
                        ...((instance.config?.host as Record<string, unknown> | undefined) ?? {}),
                        categoryRemapDismissedAt: new Date().toISOString(),
                    },
                },
            });
            instances = [
                ...instances.filter((existing) => existing.id != saved.id),
                saved,
            ];
        } catch (e) {
            show_toast({
                text: $_("plugin-category-remap-dismiss-error"),
                icon: "close",
                type: "error",
            });
        } finally {
            pendingCategoryRemap = undefined;
        }
    }

    function instanceError(instance: PluginInstance | undefined) {
        if (!instance?.last_error) {
            return "";
        }
        const code = instance.last_error.code?.trim();
        const message = instance.last_error.message?.trim();
        if (!code && !message) {
            return "";
        }
        return translatePluginError(code, message);
    }

    function instanceForPlugin(plugin: PluginProvider) {
        return instances.find((instance) => instance.plugin_id == plugin.id);
    }

    function pluginForInstance(instance: PluginInstance) {
        return plugins.find((plugin) => plugin.id === instance.plugin_id) ?? selectedPlugin;
    }

    async function maybePromptBackfilledCategoryRemap() {
        for (const instance of instances) {
            if (pendingCategoryRemap) {
                return;
            }
            const hostConfig = (instance.config?.host as Record<string, unknown> | undefined) ?? {};
            const mappingUpdatedAt = typeof hostConfig.categoryMappingUpdatedAt === "string"
                ? Date.parse(hostConfig.categoryMappingUpdatedAt)
                : 0;
            const dismissedAt = typeof hostConfig.categoryRemapDismissedAt === "string"
                ? Date.parse(hostConfig.categoryRemapDismissedAt)
                : 0;
            if (dismissedAt > mappingUpdatedAt) {
                continue;
            }
            const plugin = pluginForInstance(instance);
            if (!plugin || plugin.status !== "available") {
                continue;
            }
            await maybePromptCategoryRemap(instance, plugin, { backfilledOnly: true });
        }
    }

    async function confirmCategoryMappingSave(candidate: Partial<PluginInstance>) {
        if (!candidate.id) {
            return true;
        }
        const plugin = plugins.find((item) => item.id === candidate.plugin_id) ?? selectedPlugin;
        if (!plugin) {
            return true;
        }

        try {
            const preview = await plugin_category_remap_preview(candidate.id, candidate.config);

            let resolvePrompt: (confirmed: boolean) => void = () => {};
            const result = new Promise<boolean>((resolve) => {
                resolvePrompt = resolve;
            });
            pendingCategoryRemap = {
                instanceId: candidate.id,
                pluginTitle: pluginTitle(plugin),
                count: preview.count,
                backfilledSinceMapping: 0,
                backfilledOnly: false,
                beforeSave: true,
                onresolve: resolvePrompt,
            };
            await tick();
            categoryRemapConfirmModal?.openModal();
            return await result;
        } catch (e) {
            show_toast({
                text: $_("plugin-category-remap-preview-error"),
                icon: "close",
                type: "error",
            });
            return false;
        }
    }

    function categoryRemapConfirmText() {
        if (!pendingCategoryRemap) {
            return "";
        }

        if (pendingCategoryRemap.backfilledOnly) {
            return $_("plugin-category-remap-backfilled-confirm", {
                values: {
                    count: pendingCategoryRemap.count,
                },
            });
        }

        if (pendingCategoryRemap.beforeSave && pendingCategoryRemap.count <= 0) {
            return $_("plugin-category-remap-confirm-unspecified");
        }

        const confirmText = $_("plugin-category-remap-confirm", {
            values: {
                count: pendingCategoryRemap.count,
            },
        });
        if (pendingCategoryRemap.backfilledSinceMapping <= 0) {
            return confirmText;
        }
        return `${confirmText} ${$_("plugin-category-remap-sync-hint")}`;
    }

    function categoryRemapAction() {
        if (pendingCategoryRemap?.beforeSave && pendingCategoryRemap.count <= 0) {
            return "save";
        }
        return "plugin-category-remap-action";
    }

    function categoryRemapAlternative() {
        if (!pendingCategoryRemap?.beforeSave || pendingCategoryRemap.count <= 0) {
            return undefined;
        }
        return "plugin-category-remap-continue-without-remap";
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

    function pluginSettingsModalKey(plugin: PluginProvider) {
        const instance = instanceForPlugin(plugin);
        return `${plugin.id}:${instance?.id ?? "new"}:${JSON.stringify(instance?.config ?? {})}`;
    }

</script>

<svelte:head>
    <title>{$_("settings")} | wanderer</title>
</svelte:head>

<h3 class="text-2xl font-semibold">{$_("plugins")}</h3>
<hr class="mt-4 mb-6 border-input-border" />

{#if pluginGroups.length === 0}
    <div
        class="rounded-xl border border-dashed border-input-border bg-input-background px-6 py-10 text-center"
    >
        <div
            class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-input-border text-xl text-gray-500"
            aria-hidden="true"
        >
            <i class="fa fa-plug"></i>
        </div>
        <h4 class="text-lg font-semibold">{$_("plugins-empty-title")}</h4>
        <p class="mx-auto mt-2 max-w-xl text-sm text-gray-500">
            {$_("plugins-empty-description")}
        </p>
        <a
            class="btn-secondary mt-6 inline-flex items-center justify-center gap-2"
            href="https://wanderer.to/run/installation/plugins"
            target="_blank"
            rel="noreferrer"
        >
            <i class="fa fa-book" aria-hidden="true"></i>
            {$_("plugins-empty-docs-link")}
        </a>
    </div>
{:else}
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
                    {@const settingsDisabled = plugin.status != "available"}
                    <PluginCard
                        img={pluginLogo(plugin)}
                        title={pluginTitle(plugin)}
                        description={pluginDescription(plugin)}
                        {settingsDisabled}
                        toggleDisabled={!instance || settingsDisabled || pluginRequiresConnection(plugin, instance)}
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
{/if}

{#if pendingCategoryRemap}
    <ConfirmModal
        bind:this={categoryRemapConfirmModal}
        id="plugin-category-remap-confirm"
        title={$_("plugin-category-remap-title")}
        text={categoryRemapConfirmText()}
        action={categoryRemapAction()}
        deny={pendingCategoryRemap.backfilledOnly
            ? "plugin-category-remap-ignore"
            : pendingCategoryRemap.beforeSave
              ? "plugin-category-remap-back-to-settings"
              : "cancel"}
        alternative={categoryRemapAlternative()}
        onconfirm={confirmCategoryRemap}
        oncancel={dismissCategoryRemap}
        onalternative={continueWithoutCategoryRemap}
    ></ConfirmModal>
{/if}

{#if selectedPlugin}
    {#key pluginSettingsModalKey(selectedPlugin)}
        <PluginInstanceSettingsModal
            bind:this={pluginSettingsModal}
            plugin={selectedPlugin}
            instance={instanceForPlugin(selectedPlugin)}
            categories={categories}
            onbeforecategorymappingsave={confirmCategoryMappingSave}
            onsave={savePluginInstance}
        ></PluginInstanceSettingsModal>
    {/key}
{/if}
