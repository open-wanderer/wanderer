<script lang="ts">
    import ConfirmModal from "$lib/components/confirm_modal.svelte";
    import PluginCard from "$lib/components/settings/plugins/plugin_card.svelte";
    import Modal from "$lib/components/base/modal.svelte";
    import PluginInstanceSettingsModal from "$lib/components/settings/plugins/plugin_instance_settings_modal.svelte";
    import type { Category } from "$lib/models/category.js";
    import type { PluginInstance } from "$lib/models/plugin_instance.js";
    import type { PluginProvider } from "$lib/models/plugin_provider.js";
    import type { RoutingSettings } from "$lib/models/routing.js";
    import type { Subcategory } from "$lib/models/subcategory.js";
    import {
        plugin_category_remap_apply,
        plugin_category_remap_preview,
        plugin_instances_create,
        plugin_instances_update,
    } from "$lib/stores/plugin_instance_store.js";
    import { checkRoutingPlugin } from "$lib/stores/routing_store.svelte.js";
    import { show_toast } from "$lib/stores/toast_store.svelte.js";
    import {
        pluginDescription as localizedPluginDescription,
        pluginTitle as localizedPluginTitle,
    } from "$lib/util/plugin_i18n";
    import {
        translatePluginAPIError,
        translatePluginError,
    } from "$lib/util/plugin_error_i18n";
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
    let subcategories: Subcategory[] = $state(untrack(() => data.subcategories ?? []));
    let currentRoutingSettings: RoutingSettings = $state(
        untrack(() => data.currentRoutingSettings ?? {}),
    );

    let pluginSettingsModal: PluginInstanceSettingsModal | undefined = $state();
    let categoryRemapConfirmModal: ConfirmModal | undefined = $state();
    let disableRemoteAssetsModal: Modal | undefined = $state();
    let selectedPlugin: PluginProvider | undefined = $state();
    let currentTheme: "dark" | "light" = $state("light");
    let remoteAssetSummaries: Record<string, RemoteAssetsSummary> = $state({});
    let materializeJob: MaterializeJob | null = $state(null);
    let materializingPluginId: string | null = $state(null);
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
    let pendingDisable:
        | { plugin: PluginProvider; instance: PluginInstance; summary: RemoteAssetsSummary }
        | undefined = $state();
    type PluginGroup = { type: PluginProvider["type"]; plugins: PluginProvider[] };
    const applyRemapAfterSave = new Set<string>();
    const suppressRemapPromptAfterSave = new Set<string>();

    let pluginGroups = $derived.by(() => {
        const collator = new Intl.Collator($locale ?? undefined, {
            sensitivity: "base",
            numeric: true,
        });
        const groups: PluginGroup[] = [];
        const sortedPlugins = [...plugins].sort((a, b) => {
            return (
                comparePluginTypes(a.type, b.type, collator) ||
                comparePluginsByName(a, b, collator)
            );
        });

        for (const plugin of sortedPlugins) {
            let group = groups.find((candidate) => candidate.type === plugin.type);
            if (!group) {
                group = { type: plugin.type, plugins: [] };
                groups.push(group);
            }
            group.plugins.push(plugin);
        }
        return groups;
    });

    interface RemoteAssetsSummary {
        count: number;
        publicCount: number;
        missingCount: number;
        inaccessibleCount: number;
    }

    interface MaterializeJob {
        id: string;
        pluginId: string;
        kind: "materialize" | "repair" | "delete";
        status: "running" | "completed" | "failed";
        total: number;
        processed: number;
        failed: number;
        error?: string;
    }

    onMount(() => {
        currentTheme = document.documentElement.classList.contains("dark") ? "dark" : "light";
        void loadRemoteAssetSummaries();
        queueMicrotask(() => {
            void maybePromptBackfilledCategoryRemap();
        });
        return theme.subscribe((value) => {
            currentTheme = value;
        });
    });

    async function savePluginInstance(instance: Partial<PluginInstance>) {
        try {
            const plugin = plugins.find((candidate) => candidate.id === instance.plugin_id);
            if (plugin?.type === "routing" && instance.enabled && !(await validateRoutingPlugin(plugin, instance))) {
                return;
            }
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
            await loadRemoteAssetSummaries();

            show_toast({
                text: $_("settings-saved"),
                icon: "check",
                type: "success",
            });

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
        } catch (e) {
            show_toast({
                text: translatePluginAPIError(
                    e,
                    $_("error-setting-up-plugin", {
                        values: { provider: instance.plugin_id },
                    }),
                ),
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
            if (!value || !canCreateInstanceFromToggle(plugin)) {
                return;
            }
            await createEnabledPluginInstance(plugin);
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
        if (value && plugin.type === "routing" && !(await validateRoutingPlugin(plugin, instance))) {
            return;
        }
        if (!value && plugin.type === "assets") {
            const summary = await fetchRemoteAssetSummary(plugin);
            if (summary.count > 0) {
                pendingDisable = { plugin, instance, summary };
                materializeJob = null;
                materializingPluginId = null;
                instances = [...instances];
                await tick();
                disableRemoteAssetsModal?.openModal();
                return;
            }
        }

        await setPluginEnabled(plugin, instance, value);
    }

    async function createEnabledPluginInstance(plugin: PluginProvider) {
        try {
            const instance = {
                plugin_id: plugin.id,
                enabled: true,
                status: "configured" as const,
            };
            if (plugin.type === "routing" && !(await validateRoutingPlugin(plugin, instance))) {
                return;
            }
            const saved = await plugin_instances_create({
                ...instance,
            });
            instances = [
                ...instances.filter((existing) => existing.id != saved.id),
                saved,
            ];
            await loadRemoteAssetSummaries();
        } catch (e) {
            show_toast({
                text: translatePluginAPIError(
                    e,
                    $_("error-setting-up-plugin", { values: { provider: pluginTitle(plugin) } }),
                ),
                icon: "close",
                type: "error",
            });
            return;
        }

        show_toast({
            text: pluginTitle(plugin) + " " + $_("plugin-enabled"),
            icon: "check",
            type: "success",
        });
    }

    async function setPluginEnabled(
        plugin: PluginProvider,
        instance: PluginInstance,
        value: boolean,
    ) {
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
            await loadRemoteAssetSummaries();
            if (value) {
                await maybePromptCategoryRemap(saved, plugin);
            }
        } catch (e) {
            show_toast({
                text: translatePluginAPIError(
                    e,
                    $_("error-setting-up-plugin", { values: { provider: pluginTitle(plugin) } }),
                ),
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

    async function fetchRemoteAssetSummary(plugin: PluginProvider): Promise<RemoteAssetsSummary> {
        if (plugin.type !== "assets") {
            return emptyRemoteAssetsSummary();
        }
        const response = await fetch(`/api/v1/plugins/assets/${encodeURIComponent(plugin.id)}/remote-assets-summary`);
        if (!response.ok) {
            return emptyRemoteAssetsSummary();
        }
        const summary = (await response.json()) as RemoteAssetsSummary;
        remoteAssetSummaries = {
            ...remoteAssetSummaries,
            [plugin.id]: summary,
        };
        return summary;
    }

    async function loadRemoteAssetSummaries() {
        const assetPlugins = plugins.filter((plugin) => plugin.type === "assets" && instanceForPlugin(plugin));
        await Promise.all(assetPlugins.map((plugin) => fetchRemoteAssetSummary(plugin)));
    }

    async function startMaterialize(plugin: PluginProvider, disableAfter = false) {
        await startRemoteAssetJob(plugin, "materialize", disableAfter);
    }

    async function startRepair(plugin: PluginProvider) {
        await startRemoteAssetJob(plugin, "repair", false);
    }

    async function startDelete(plugin: PluginProvider, disableAfter = false) {
        await startRemoteAssetJob(plugin, "delete", disableAfter);
    }

    async function startRemoteAssetJob(
        plugin: PluginProvider,
        action: "materialize" | "repair" | "delete",
        disableAfter = false,
    ) {
        materializingPluginId = plugin.id;
        materializeJob = null;
        try {
            const endpoint =
                action === "materialize"
                    ? "materialize-all"
                    : action === "repair"
                      ? "repair-remote-assets"
                      : "delete-remote-assets";
            const response = await fetch(`/api/v1/plugins/assets/${encodeURIComponent(plugin.id)}/${endpoint}`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({}),
            });
            if (!response.ok) {
                const error = await response.json().catch(() => ({}));
                throw new Error(error.message ?? $_("error-generic"));
            }
            materializeJob = (await response.json()) as MaterializeJob;
            const finished = await pollMaterializeJob(materializeJob.id);
            await loadRemoteAssetSummaries();
            if (finished.failed > 0 || finished.status === "failed") {
                show_toast({
                    text: $_(`plugin-linked-photos-${action}-failed`),
                    icon: "close",
                    type: "error",
                });
                return;
            }
            show_toast({
                text: $_(`plugin-linked-photos-${action}-complete`),
                icon: "check",
                type: "success",
            });
            if (disableAfter && pendingDisable) {
                await setPluginEnabled(pendingDisable.plugin, pendingDisable.instance, false);
                pendingDisable = undefined;
                disableRemoteAssetsModal?.closeModal();
            }
        } catch (e) {
            show_toast({
                text: e instanceof Error ? e.message : $_("error-generic"),
                icon: "close",
                type: "error",
            });
        } finally {
            materializingPluginId = null;
        }
    }

    async function validateRoutingPlugin(plugin: PluginProvider, instance: Partial<PluginInstance>) {
        try {
            await checkRoutingPlugin(plugin.id, instance.config);
            return true;
        } catch (e) {
            show_toast({
                text: $_("routing-plugin-unavailable"),
                icon: "close",
                type: "error",
            });
            return false;
        }
    }

    async function pollMaterializeJob(id: string): Promise<MaterializeJob> {
        while (true) {
            await new Promise((resolve) => setTimeout(resolve, 750));
            const response = await fetch(`/api/v1/plugins/assets/jobs/materialize/${encodeURIComponent(id)}`);
            if (!response.ok) {
                const error = await response.json().catch(() => ({}));
                throw new Error(error.message ?? $_("error-generic"));
            }
            materializeJob = (await response.json()) as MaterializeJob;
            if (materializeJob.status !== "running") {
                return materializeJob;
            }
        }
    }

    function remoteSummary(plugin: PluginProvider): RemoteAssetsSummary {
        return remoteAssetSummaries[plugin.id] ?? emptyRemoteAssetsSummary();
    }

    function emptyRemoteAssetsSummary(): RemoteAssetsSummary {
        return { count: 0, publicCount: 0, missingCount: 0, inaccessibleCount: 0 };
    }

    function remoteProblemCount(plugin: PluginProvider) {
        const summary = remoteSummary(plugin);
        return summary.missingCount + summary.inaccessibleCount;
    }

    function pluginCardActionLabel(plugin: PluginProvider, instance: PluginInstance | undefined): string {
        if (plugin.type !== "assets" || !instance?.enabled || remoteSummary(plugin).count === 0) {
            return "";
        }
        if (remoteProblemCount(plugin) > 0) {
            return $_("plugin-linked-photos-repair");
        }
        return $_("plugin-linked-photos-download");
    }

    function pluginCardAction(plugin: PluginProvider, instance: PluginInstance | undefined) {
        if (plugin.type !== "assets" || !instance?.enabled || remoteSummary(plugin).count === 0) {
            return undefined;
        }
        if (remoteProblemCount(plugin) > 0) {
            return () => startRepair(plugin);
        }
        return () => startMaterialize(plugin, false);
    }

    function materializeProgress() {
        if (!materializeJob?.total) {
            return 0;
        }
        return Math.round((materializeJob.processed / materializeJob.total) * 100);
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

    function comparePluginTypes(
        a: PluginProvider["type"],
        b: PluginProvider["type"],
        collator: Intl.Collator,
    ) {
        return compareLocalizedText(pluginTypeTitle(a), pluginTypeTitle(b), a, b, collator);
    }

    function comparePluginsByName(
        a: PluginProvider,
        b: PluginProvider,
        collator: Intl.Collator,
    ) {
        return compareLocalizedText(pluginTitle(a), pluginTitle(b), a.id, b.id, collator);
    }

    function compareLocalizedText(
        a: string,
        b: string,
        fallbackA: string,
        fallbackB: string,
        collator: Intl.Collator,
    ) {
        const result = collator.compare(a.trim() || fallbackA, b.trim() || fallbackB);
        return result || collator.compare(fallbackA, fallbackB);
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
        return `${plugin.id}:${instance?.id ?? "new"}`;
    }

    function canCreateInstanceFromToggle(plugin: PluginProvider) {
        return plugin.auth.type !== "oauth2" && (plugin.auth.fields?.length ?? 0) === 0;
    }

    function pluginSettingsAvailable(plugin: PluginProvider) {
        return true;
    }

    function routingPluginIsSelected(plugin: PluginProvider) {
        return plugin.type === "routing" &&
            currentRoutingSettings.exposedFeatures?.routing !== false &&
            (currentRoutingSettings.primaryRoutePluginId === plugin.id ||
                currentRoutingSettings.elevationPluginId === plugin.id);
    }

    function pluginToggleDisabled(
        plugin: PluginProvider,
        instance: PluginInstance | undefined,
    ) {
        return plugin.status != "available" ||
            pluginRequiresConnection(plugin, instance) ||
            (!instance && !canCreateInstanceFromToggle(plugin)) ||
            (instance?.enabled === true && routingPluginIsSelected(plugin));
    }

    function pluginToggleTitle(
        plugin: PluginProvider,
        instance: PluginInstance | undefined,
    ) {
        if (instance?.enabled && routingPluginIsSelected(plugin)) {
            return $_("plugin-selected-in-routing-settings");
        }
        return "";
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
                        disabled={pluginToggleDisabled(plugin, instance)}
                        active={instance?.enabled ?? false}
                        lastSyncAt={instance?.last_sync_at}
                        error={pluginCardError(plugin, instance)}
                        toggleTitle={pluginToggleTitle(plugin, instance)}
                        settingsAvailable={pluginSettingsAvailable(plugin)}
                        onclick={() => openPluginSettings(plugin)}
                        ontoggle={(value) => onPluginToggle(plugin, instance, value)}
                        actionLabel={pluginCardActionLabel(plugin, instance)}
                        actionLoading={materializingPluginId === plugin.id}
                        onaction={pluginCardAction(plugin, instance)}
                    ></PluginCard>
                {/each}
            </div>
        </section>
    {/each}
</div>

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
            subcategories={subcategories}
            onbeforecategorymappingsave={confirmCategoryMappingSave}
            onsave={savePluginInstance}
        ></PluginInstanceSettingsModal>
    {/key}
{/if}

<Modal
    id="plugin-linked-photos-disable-modal"
    title={$_("plugin-disable-linked-photos-title")}
    bind:this={disableRemoteAssetsModal}
>
    {#snippet content()}
        {#if pendingDisable}
            <p>
                {$_("plugin-disable-linked-photos-body", {
                    values: {
                        count: pendingDisable.summary.count,
                        publicCount: pendingDisable.summary.publicCount,
                    },
                })}
            </p>
            {#if materializeJob}
                <div class="space-y-2">
                    <div class="h-2 overflow-hidden rounded bg-input-background">
                        <div
                            class="h-full bg-toggle-active transition-all"
                            style={`width: ${materializeProgress()}%`}
                        ></div>
                    </div>
                    <p class="text-sm text-gray-500">
                        {$_("plugin-linked-photos-progress", {
                            values: {
                                processed: materializeJob.processed,
                                total: materializeJob.total,
                                failed: materializeJob.failed,
                            },
                        })}
                    </p>
                </div>
            {/if}
            {#if pendingDisable.summary.missingCount + pendingDisable.summary.inaccessibleCount > 0}
                <p class="text-sm text-red-400">
                    {$_("plugin-linked-photos-status-warning", {
                        values: {
                            missing: pendingDisable.summary.missingCount,
                            inaccessible: pendingDisable.summary.inaccessibleCount,
                        },
                    })}
                </p>
            {/if}
        {/if}
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button
                class="btn-secondary"
                disabled={materializingPluginId !== null}
                onclick={() => {
                    pendingDisable = undefined;
                    disableRemoteAssetsModal?.closeModal();
                }}
            >{$_("cancel")}</button>
            {#if pendingDisable}
                <button
                    class="btn-primary"
                    class:btn-disabled={materializingPluginId !== null}
                    disabled={materializingPluginId !== null}
                    type="button"
                    onclick={() => pendingDisable && startMaterialize(pendingDisable.plugin, true)}
                >
                    {#if materializingPluginId === pendingDisable.plugin.id}
                        <span class="spinner mr-2 inline-block h-4 w-4"></span>
                    {/if}
                    {$_("plugin-linked-photos-download-and-disable")}
                </button>
                <button
                    class="btn-danger"
                    class:btn-disabled={materializingPluginId !== null}
                    disabled={materializingPluginId !== null}
                    type="button"
                    onclick={() => pendingDisable && startDelete(pendingDisable.plugin, true)}
                >
                    {#if materializingPluginId === pendingDisable.plugin.id && materializeJob?.kind === "delete"}
                        <span class="spinner mr-2 inline-block h-4 w-4"></span>
                    {/if}
                    {$_("plugin-linked-photos-delete-and-disable")}
                </button>
            {/if}
        </div>
    {/snippet}
</Modal>
