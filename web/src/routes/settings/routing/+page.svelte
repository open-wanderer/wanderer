<script lang="ts">
    import { page } from "$app/state";
    import Select from "$lib/components/base/select.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import type { RoutingEngine, RoutingSettings } from "$lib/models/routing";
    import { settings_update } from "$lib/stores/settings_store";
    import {
        routingEngines,
        routingSettings,
        updateRoutingSettings,
    } from "$lib/stores/routing_store.svelte";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { currentUser } from "$lib/stores/user_store";
    import { APIError } from "$lib/util/api_util";
    import { ROUTING_MAX_VARIANTS } from "$lib/util/routing_variant_util";
    import { onMount, untrack } from "svelte";
    import { _ } from "svelte-i18n";

    let settings = $derived(page.data.settings);
    let allowAutoGeolocate = $state(
        page.data.settings.behavior?.allowAutoGeolocate ?? false,
    );
    let engines: RoutingEngine[] = $state([]);
    let routingConfig: RoutingSettings = $state({});
    let primaryRoutePluginId = $state("");
    let elevationPluginId = $state("");
    let maneuverPluginId = $state("");
    let defaultRoutingMode: "segment" | "via" = $state("segment");
    let defaultVariantCount = $state(3);
    let routingEnabled = $state(true);
    let variantsEnabled = $state(true);
    let parallelRoutingEnabled = $state(true);
    let loading = $state(true);
    let loadError = $state("");
    let savingBehavior = $state(false);
    let savingRouting = $state(false);
    let enabledRoutePluginAvailable = $derived(
        engines.some(
            (engine) =>
                engine.enabled && (engine.roles ?? []).includes("route"),
        ),
    );
    $effect(() => {
        const behavior = page.data.settings?.behavior;
        if (behavior) {
            untrack(() => {
                allowAutoGeolocate = behavior.allowAutoGeolocate ?? false;
            });
        }
    });

    onMount(() => {
        void loadRoutingSettings();
    });

    async function loadRoutingSettings() {
        loading = true;
        loadError = "";
        try {
            const [loadedSettings, loadedEngines] = await Promise.all([
                routingSettings(),
                routingEngines(),
            ]);
            routingConfig = loadedSettings;
            engines = loadedEngines;
            primaryRoutePluginId = loadedSettings.primaryRoutePluginId ?? "";
            elevationPluginId = loadedSettings.elevationPluginId ?? "";
            maneuverPluginId = loadedSettings.maneuverPluginId ?? "";
            defaultRoutingMode = loadedSettings.defaultRoutingMode ?? "segment";
            defaultVariantCount = Math.min(
                ROUTING_MAX_VARIANTS,
                Math.max(1, loadedSettings.defaultVariantCount ?? 3),
            );
            routingEnabled = loadedSettings.exposedFeatures?.routing !== false;
            variantsEnabled = loadedSettings.exposedFeatures?.variants !== false;
            parallelRoutingEnabled =
                loadedSettings.exposedFeatures?.parallelRouting !== false;
        } catch (error) {
            loadError = errorMessage(error);
        } finally {
            loading = false;
        }
    }

    async function selectPrimary(value: string) {
        const previous = primaryRoutePluginId;
        primaryRoutePluginId = value;
        savingRouting = true;
        try {
            routingConfig = await updateRoutingSettings({
                primaryRoutePluginId: value,
            });
        } catch (error) {
            primaryRoutePluginId = previous;
            showError(error);
        } finally {
            savingRouting = false;
        }
    }

    async function selectElevation(value: string) {
        const previous = elevationPluginId;
        elevationPluginId = value;
        savingRouting = true;
        try {
            routingConfig = await updateRoutingSettings({ elevationPluginId: value });
        } catch (error) {
            elevationPluginId = previous;
            showError(error);
        } finally {
            savingRouting = false;
        }
    }

    async function selectManeuver(value: string) {
        const previous = maneuverPluginId;
        maneuverPluginId = value;
        savingRouting = true;
        try {
            routingConfig = await updateRoutingSettings({ maneuverPluginId: value });
        } catch (error) {
            maneuverPluginId = previous;
            showError(error);
        } finally {
            savingRouting = false;
        }
    }

    async function updatePhaseFiveSettings(patch: Partial<RoutingSettings>) {
        savingRouting = true;
        try {
            routingConfig = await updateRoutingSettings(patch);
        } catch (error) {
            showError(error);
            await loadRoutingSettings();
        } finally {
            savingRouting = false;
        }
    }

    async function selectDefaultRoutingMode(value: unknown) {
        if (value !== "segment" && value !== "via") return;
        defaultRoutingMode = value;
        await updatePhaseFiveSettings({ defaultRoutingMode });
    }

    async function selectDefaultVariantCount(value: unknown) {
        const count = Number(value);
        if (!Number.isInteger(count) || count < 1 || count > ROUTING_MAX_VARIANTS) return;
        defaultVariantCount = count;
        await updatePhaseFiveSettings({ defaultVariantCount });
    }

    async function toggleRoutingVariants(enabled: boolean) {
        const previous = !enabled;
        variantsEnabled = enabled;
        savingRouting = true;
        try {
            routingConfig = await updateRoutingSettings({
                exposedFeatures: { variants: enabled },
            });
            variantsEnabled = routingConfig.exposedFeatures?.variants !== false;
        } catch (error) {
            variantsEnabled = previous;
            showError(error);
        } finally {
            savingRouting = false;
        }
    }

    async function toggleRouting(enabled: boolean) {
        const previous = !enabled;
        if (enabled && !enabledRoutePluginAvailable) {
            routingEnabled = false;
            showError(new Error($_("routing-enable-requires-plugin")));
            return;
        }
        routingEnabled = enabled;
        savingRouting = true;
        try {
            routingConfig = await updateRoutingSettings({
                exposedFeatures: { routing: enabled },
            });
            routingEnabled = routingConfig.exposedFeatures?.routing !== false;
        } catch (error) {
            routingEnabled = previous;
            showError(error);
        } finally {
            savingRouting = false;
        }
    }

    async function toggleParallelRouting(enabled: boolean) {
        const previous = !enabled;
        parallelRoutingEnabled = enabled;
        savingRouting = true;
        try {
            routingConfig = await updateRoutingSettings({
                exposedFeatures: { parallelRouting: enabled },
            });
            parallelRoutingEnabled =
                routingConfig.exposedFeatures?.parallelRouting !== false;
        } catch (error) {
            parallelRoutingEnabled = previous;
            showError(error);
        } finally {
            savingRouting = false;
        }
    }

    function engineSupportsVia(engine: RoutingEngine) {
        const routing = engine.metadata?.routing;
        return typeof routing === "object" && routing !== null &&
            (routing as Record<string, unknown>).supportsViaRouting === true;
    }

    async function handleAutoGeolocateChange() {
        if (!settings) return;
        const previous = settings.behavior?.allowAutoGeolocate ?? false;
        savingBehavior = true;
        try {
            await settings_update({
                ...settings,
                behavior: {
                    ...settings.behavior,
                    allowAutoGeolocate,
                },
            });
        } catch (error) {
            allowAutoGeolocate = previous;
            showError(error);
        } finally {
            savingBehavior = false;
        }
    }

    function routeEngineItems() {
        return engineItems("route", primaryRoutePluginId);
    }

    function elevationEngineItems() {
        return engineItems("elevation", elevationPluginId);
    }

    function maneuverEngineItems() {
        return engineItems("maneuvers", maneuverPluginId);
    }

    function showRouteEngineSelect() {
        return routeEngineItems().length > 1;
    }

    function showElevationEngineSelect() {
        return elevationEngineItems().length > 1;
    }

    function showManeuverEngineSelect() {
        return maneuverEngineItems().length > 1;
    }

    function routingModeItems() {
        const items = [
            { text: $_("routing-mode-segment"), value: "segment" },
        ];
        if (engines.some((engine) => engine.enabled && engineSupportsVia(engine))) {
            items.push({ text: $_("routing-mode-via"), value: "via" });
        }
        return items;
    }

    function variantCountItems() {
        return Array.from({ length: ROUTING_MAX_VARIANTS }, (_, index) => ({
            text: String(index + 1),
            value: index + 1,
        }));
    }

    function comparisonEngines() {
        return engines.filter(
            (engine) =>
                engine.enabled &&
                engine.pluginId !== primaryRoutePluginId &&
                (engine.roles ?? []).includes("route"),
        );
    }

    function engineItems(role: string, selected: string) {
        const items = engines
            .filter((engine) => engine.enabled && (engine.roles ?? []).includes(role))
            .map((engine) => ({ text: engine.name, value: engine.pluginId }));
        if (selected && !items.some((item) => item.value === selected)) {
            items.push({
                text: `${selected} (${$_("plugin-disabled")})`,
                value: selected,
            });
        }
        return items;
    }

    function showError(error: unknown) {
        show_toast({
            type: "error",
            icon: "close",
            text: errorMessage(error),
        });
    }

    function errorMessage(error: unknown) {
        if (
            error instanceof APIError &&
            (error.message === "routing_plugin_required" ||
                routingSettingsErrorCode(error.detail) === "routing_plugin_required")
        ) {
            return $_("routing-enable-requires-plugin");
        }
        return error instanceof Error && error.message ? error.message : $_("error-generic");
    }

    function routingSettingsErrorCode(value: unknown): string | undefined {
        if (!value || typeof value !== "object") return undefined;
        const record = value as Record<string, unknown>;
        if (typeof record.code === "string") return record.code;
        for (const nested of Object.values(record)) {
            const code = routingSettingsErrorCode(nested);
            if (code) return code;
        }
        return undefined;
    }
</script>

<svelte:head>
    <title>{$_("routing")} | {$_("settings")} | wanderer</title>
</svelte:head>

<h2 class="text-2xl font-semibold">{$_("routing")}</h2>
<hr class="mt-4 mb-6 border-input-border" />

{#if $currentUser}
    <div class="space-y-12">
        {#if loading}
            <div class="flex items-center gap-3 py-4 text-sm text-gray-500">
                <span class="spinner inline-block h-5 w-5"></span>
                <span>{$_("routing-settings-loading")}</span>
            </div>
        {:else if loadError}
            <p class="rounded-md border border-red-400 p-3 text-sm text-red-400">
                {loadError}
            </p>
        {:else}
            <section class="space-y-4">
                <div class="grid grid-cols-[1fr_min-content] items-center gap-4">
                    <div>
                        <h4 class="mb-2 text-xl font-medium">{$_("routing-enabled")}</h4>
                        <p class="max-w-3xl text-sm text-gray-500">
                            {$_("routing-enabled-help")}
                        </p>
                        {#if !routingEnabled && !enabledRoutePluginAvailable}
                            <p class="mt-2 text-xs text-orange-500">
                                {$_("routing-enable-requires-plugin")}
                            </p>
                        {/if}
                    </div>
                    <Toggle
                        name="routing-enabled"
                        bind:value={routingEnabled}
                        disabled={savingRouting || (!routingEnabled && !enabledRoutePluginAvailable)}
                        ariaLabel={$_("routing-enabled")}
                        onchange={toggleRouting}
                    ></Toggle>
                </div>
            </section>

            {#if showManeuverEngineSelect()}
                <section class="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <Select
                        name="routing-maneuver-engine"
                        label={$_("routing-maneuver-engine")}
                        items={maneuverEngineItems()}
                        value={maneuverPluginId}
                        disabled={savingRouting}
                        onchange={selectManeuver}
                    ></Select>
                </section>
            {/if}

            {#if routingEnabled && (showRouteEngineSelect() || engines.length === 0)}
                <section class="space-y-4">
                    <div>
                        <h4 class="mb-2 text-xl font-medium">{$_("routing-engine-settings")}</h4>
                        {#if showRouteEngineSelect()}
                            <p class="max-w-3xl text-sm text-gray-500">
                                {$_("routing-engine-settings-help-route-only")}
                            </p>
                        {/if}
                    </div>

                    {#if showRouteEngineSelect()}
                        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <Select
                                name="routing-primary-engine"
                                items={routeEngineItems()}
                                value={primaryRoutePluginId}
                                disabled={savingRouting}
                                onchange={selectPrimary}
                            ></Select>
                        </div>
                    {/if}
                    {#if engines.length === 0}
                        <p class="text-sm text-gray-500">
                            {$_("no-routing-plugin-available")}
                            <a class="ml-1 underline" href="/settings/plugins">{$_("plugins")}</a>
                        </p>
                    {/if}
                </section>
            {/if}

            {#if routingEnabled && showElevationEngineSelect()}
                <section class="space-y-4">
                    <div>
                        <h4 class="mb-2 text-xl font-medium">{$_("routing-elevation-engine")}</h4>
                        <p class="max-w-3xl text-sm text-gray-500">
                            {$_("routing-engine-settings-help-elevation-only")}
                        </p>
                    </div>
                    <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                        <Select
                            name="routing-elevation-engine"
                            items={elevationEngineItems()}
                            value={elevationPluginId}
                            disabled={savingRouting}
                            onchange={selectElevation}
                        ></Select>
                    </div>
                </section>
            {/if}

            {#if routingEnabled}
                <section class="space-y-4">
                    <div class="grid grid-cols-[1fr_min-content] items-center gap-4">
                        <div>
                            <h4 class="mb-2 text-xl font-medium">{$_("routing-variants")}</h4>
                            <p class="max-w-3xl text-sm text-gray-500">
                                {$_("routing-variants-enabled-help")}
                            </p>
                        </div>
                        <Toggle
                            name="routing-variants-enabled"
                            bind:value={variantsEnabled}
                            disabled={savingRouting}
                            ariaLabel={$_("routing-variants-enabled")}
                            onchange={toggleRoutingVariants}
                        ></Toggle>
                    </div>

                    {#if variantsEnabled}
                        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <Select
                                name="routing-default-variant-count"
                                label={$_("routing-default-variant-count")}
                                items={variantCountItems()}
                                value={defaultVariantCount}
                                disabled={savingRouting}
                                onchange={selectDefaultVariantCount}
                            ></Select>
                        </div>

                        {#if comparisonEngines().length > 0}
                            <div class="grid grid-cols-[1fr_min-content] items-center gap-4 pt-2">
                                <div>
                                    <h5 class="mb-2 text-lg font-medium">
                                        {$_("routing-comparison-engines")}
                                    </h5>
                                    <p class="max-w-3xl text-sm text-gray-500">
                                        {$_("routing-comparison-engines-help")}
                                    </p>
                                </div>
                                <Toggle
                                    name="routing-comparison-engines"
                                    bind:value={parallelRoutingEnabled}
                                    ariaLabel={$_("routing-comparison-engines")}
                                    disabled={savingRouting}
                                    onchange={toggleParallelRouting}
                                ></Toggle>
                            </div>
                        {/if}
                    {/if}
                </section>
            {/if}
        {/if}

        {#if !loading && !loadError && routingEnabled}
            <section class="space-y-4">
                <div>
                    <h4 class="mb-2 text-xl font-medium">{$_("routing-editor-behavior")}</h4>
                    <p class="max-w-3xl text-sm text-gray-500">
                        {$_("routing-editor-behavior-help")}
                    </p>
                </div>
                <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <Select
                        name="routing-default-mode"
                        label={$_("routing-default-mode")}
                        items={routingModeItems()}
                        value={defaultRoutingMode}
                        disabled={savingRouting}
                        onchange={selectDefaultRoutingMode}
                    ></Select>
                </div>
                <div class="grid grid-cols-[1fr_min-content] items-center gap-4">
                    <p>{$_("allow-auto-geolocate")}</p>
                    <Toggle
                        bind:value={allowAutoGeolocate}
                        disabled={savingBehavior}
                        onchange={handleAutoGeolocateChange}
                    ></Toggle>
                </div>
            </section>
        {/if}
    </div>
{/if}
