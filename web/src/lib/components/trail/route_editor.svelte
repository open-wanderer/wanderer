<script lang="ts">
    import type {
        RoutingEffectiveControls,
        RoutingEngine,
        RoutingOptions,
    } from "$lib/models/routing";
    import { routingStore } from "$lib/stores/routing_store.svelte";
    import { formatSpeed } from "$lib/util/format_util";
    import {
        cloneRoutingControlValues,
        formatRoutingControlUnit,
        routingControlBucketTranslationKey,
        routingControlTranslationKey,
    } from "$lib/util/routing_control_util";
    import { _, locale } from "svelte-i18n";
    import { slide } from "svelte/transition";
    import Button from "../base/button.svelte";
    import DoubleSlider from "../base/double_slider.svelte";
    import FiniteNumberField from "../base/finite_number_field.svelte";
    import SingleSelect from "../base/single_select.svelte";
    import TextField from "../base/text_field.svelte";
    import Toggle from "../base/toggle.svelte";
    import { tick } from "svelte";
    interface Props {
        options: RoutingOptions;
        onReverse: () => void;
        onReset: () => void;
        onCropToggle: (active: boolean) => void;
        onUpdateCropRange: (data: [number, number]) => void;
        onCrop: () => void;
        onRecalculateElevationData: () => void;
        onUndo: () => void;
        onRedo: () => void;
        provenanceMismatch?: boolean;
        onRerouteExisting?: () => void | Promise<void>;
        onKeepExisting?: () => void;
        routingEngines?: RoutingEngine[];
        onRoutingEngineChange?: (pluginId: string) => void | Promise<void>;
        viaAvailable?: boolean;
        routingEnabled?: boolean;
        showWaypoints?: boolean;
        effectiveControls?: RoutingEffectiveControls;
        resetLabel?: string;
        resetAriaLabel?: string;
    }

    let {
        options = $bindable(),
        onReverse,
        onReset,
        onCropToggle,
        onUpdateCropRange,
        onCrop,
        onRecalculateElevationData,
        onUndo,
        onRedo,
        provenanceMismatch = false,
        onRerouteExisting,
        onKeepExisting,
        routingEngines = [],
        onRoutingEngineChange,
        viaAvailable = false,
        routingEnabled = true,
        showWaypoints = $bindable(true),
        effectiveControls,
        resetLabel = "reset",
        resetAriaLabel = "reset-route",
    }: Props = $props();

    $effect(() => {
        if (!routingEnabled) {
            options.autoRouting = false;
            recalculateElevationData = false;
        }
        if (!routingEnabled || !options.autoRouting) {
            showSettings = false;
        }
    });

    let showSettings = $state(false);
    let editRoute = $state(false);
    let crop = $state(false);
    let recalculateElevationData = $state(false);

    const routeControls = $derived([
        ...(effectiveControls?.controls ?? []),
        ...(effectiveControls?.nativeControlGroups ?? []).flatMap((group) => group.controls),
    ]);
    const hasRouteSettings = $derived(routeControls.length > 0);

    $effect(() => {
        if (!hasRouteSettings) {
            showSettings = false;
        }
    });

    function controlLabel(control: RoutingEffectiveControls["controls"][number]) {
        const language = $locale ?? "en";
        const providerLabel =
            control.labels?.[language] ?? control.labels?.en ?? control.label ?? control.key;
        const translationKey = routingControlTranslationKey(control);
        return translationKey
            ? $_({ id: translationKey, default: providerLabel })
            : providerLabel;
    }

    function controlTarget(control: RoutingEffectiveControls["controls"][number]) {
        return control.target === "native_config" ? "nativeConfig" : "preferences";
    }

    function controlPath(control: RoutingEffectiveControls["controls"][number]) {
        return control.path?.length ? control.path : [control.key];
    }

    function nestedControlValue(values: Record<string, unknown> | undefined, path: string[]) {
        let current: unknown = values;
        for (const key of path) {
            if (!current || typeof current !== "object" || Array.isArray(current)) return undefined;
            current = (current as Record<string, unknown>)[key];
        }
        return current;
    }

    function controlValue(control: RoutingEffectiveControls["controls"][number]) {
        const values = options[controlTarget(control)] as Record<string, unknown> | undefined;
        return nestedControlValue(values, controlPath(control)) ?? control.current ?? control.default;
    }

    function setControlValue(control: RoutingEffectiveControls["controls"][number], value: unknown) {
        const target = controlTarget(control);
        const values = cloneRoutingControlValues(
            options[target] as Record<string, unknown> | undefined,
        );
        const path = controlPath(control);
        let current = values;
        for (const key of path.slice(0, -1)) {
            const nested = current[key];
            if (!nested || typeof nested !== "object" || Array.isArray(nested)) {
                current[key] = {};
            }
            current = current[key] as Record<string, unknown>;
        }
        current[path[path.length - 1]] = value;
        options[target] = values;
    }

    function controlNumber(control: RoutingEffectiveControls["controls"][number]) {
        const value = Number(controlValue(control));
        return Number.isFinite(value) ? value : Number(control.default ?? control.min ?? 0);
    }

    function controlNumberLabel(control: RoutingEffectiveControls["controls"][number]) {
        const value = controlNumber(control);
        const bucketTranslationKey = routingControlBucketTranslationKey(control, value);
        if (bucketTranslationKey) return $_(bucketTranslationKey);
        return formatRoutingControlUnit(control, value, formatSpeed) ?? String(value);
    }

    function controlOptions(control: RoutingEffectiveControls["controls"][number]) {
        const language = $locale ?? "en";
        return (control.options ?? []).map((option) => ({
            value: option.value,
            text: option.labels?.[language] ?? option.labels?.en ?? option.label ?? option.value,
        }));
    }

    function routingModeItems() {
        const items = [{ text: $_("routing-mode-segment"), value: "segment" }];
        if (viaAvailable) {
            items.push({ text: $_("routing-mode-via"), value: "via" });
        } else if (options.routingMode === "via") {
            items.push({
                text: `${$_("routing-mode-via")} (${$_("unavailable")})`,
                value: "via",
            });
        }
        return items;
    }

    function selectRoutingMode(value: unknown) {
        if (value !== "segment" && value !== "via") return;
        options.routingMode = value;
        options.routingModeExplicit = true;
    }

    async function togglePanels(_edit: boolean, _crop: boolean, _recalc: boolean) {        
        recalculateElevationData = _recalc;
        crop = _crop;
        editRoute = _edit
        await tick()
        onCropToggle(_crop);
    }

</script>

<div class="flex gap-x-2 items-start">
    <div class="flex flex-col gap-y-1 p-1 bg-background rounded-md my-2">
        <button
            class="btn-icon"
            class:bg-secondary-hover={editRoute}
            aria-label="edit route"
            onclick={async () => await togglePanels(!editRoute, false, false)}><i class="fa fa-route text-sm"></i></button
        >
        <button
            class="btn-icon"
            class:bg-secondary-hover={crop}
            aria-label="crop route"
            onclick={async () => await togglePanels(false, !crop, false)}><i class="fa fa-scissors text-sm"></i></button
        >
        {#if routingEnabled}
            <button
                class="btn-icon"
                class:bg-secondary-hover={recalculateElevationData}
                aria-label="recalculate elevation data"
                onclick={async () => await togglePanels(false, false, !recalculateElevationData)}><i class="fa fa-mountain text-sm"></i></button
            >
        {/if}
        <button
            class="btn-icon tooltip"
            class:bg-secondary-hover={showWaypoints}
            type="button"
            aria-label={$_("waypoints", { values: { n: 2 } })}
            data-title={$_("waypoints", { values: { n: 2 } })}
            onclick={() => (showWaypoints = !showWaypoints)}
            ><i class="fa fa-location-dot text-sm"></i></button
        >
        <button
            class="btn-icon tooltip hover:text-red-500"
            type="button"
            onclick={() => onReset()}
            aria-label={$_(resetAriaLabel)}
            data-title={$_(resetLabel)}><i class="fa fa-trash text-sm"></i></button
        >
        <button
            class="btn-icon"
            class:text-gray-500={routingStore.undoStack.length == 0}
            disabled={routingStore.undoStack.length == 0}
            aria-label="undo route action"
            onclick={onUndo}><i class="fa fa-undo text-sm"></i></button
        >
        <button
            class="btn-icon"
            class:text-gray-500={routingStore.redoStack.length == 0}
            disabled={routingStore.redoStack.length == 0}
            aria-label="redo route action"
            onclick={onRedo}><i class="fa fa-redo text-sm"></i></button
        >
    </div>

    {#if editRoute}
        <div class=" pt-2 pb-3 px-4 my-2 rounded-xl bg-background shadow-xl">
            {#if routingEnabled}
                <Toggle
                    bind:value={options.autoRouting}
                    label={$_("enable-auto-routing")}
                ></Toggle>
            {/if}
            <div class="flex items-center gap-4 mt-4">
                <button
                    class="btn-icon tooltip"
                    type="button"
                    onclick={() => onReverse()}
                    aria-label="Reverse trail direction"
                    data-title={$_("reverse-direction")}
                    ><i class="fa fa-arrow-right-arrow-left"></i></button
                >
                {#if routingEnabled}
                    <button
                        class="btn-icon tooltip"
                        type="button"
                        disabled={!options.autoRouting || !hasRouteSettings}
                        onclick={() => (showSettings = !showSettings)}
                        data-title={hasRouteSettings
                            ? $_("more-route-settings")
                            : $_("no-route-settings")}
                        aria-label="Toggle routing settings"
                        ><i
                            class="fa fa-cogs"
                            class:text-gray-500={!options.autoRouting ||
                                !hasRouteSettings}
                        ></i></button
                    >
                {/if}
            </div>
            {#if showSettings}
                <div class="pt-4 space-y-4" in:slide out:slide>
                    {#each routeControls as control (control.target + ":" + control.key)}
                        <div>
                            {#if control.type === "boolean" || control.valueType === "boolean"}
                                <Toggle
                                    name={`route-control-${control.key}`}
                                    label={controlLabel(control)}
                                    value={Boolean(controlValue(control))}
                                    onchange={(value) => setControlValue(control, value)}
                                ></Toggle>
                            {:else if control.options?.length}
                                <SingleSelect
                                    name={`route-control-${control.key}`}
                                    label={controlLabel(control)}
                                    items={controlOptions(control)}
                                    value={controlValue(control)}
                                    onchange={(value) => setControlValue(control, value)}
                                ></SingleSelect>
                            {:else if (control.type === "number" || control.valueType === "number") && control.min === undefined && control.max === undefined}
                                <FiniteNumberField
                                    label={controlLabel(control)}
                                    extraClasses="!h-10 !px-3 !py-0"
                                    value={controlValue(control)}
                                    onchange={(value) => setControlValue(control, value)}
                                ></FiniteNumberField>
                            {:else if control.type === "number" || control.valueType === "number"}
                                <label class="block text-sm font-medium" for={`route-control-${control.key}`}>
                                    {controlLabel(control)}
                                    <span class="ml-2 font-normal text-gray-500">{controlNumberLabel(control)}</span>
                                </label>
                                <input
                                    id={`route-control-${control.key}`}
                                    class="mt-2 w-full accent-toggle-active"
                                    type="range"
                                    min={control.min ?? 0}
                                    max={control.max ?? 1}
                                    step={control.step ?? 0.05}
                                    value={controlNumber(control)}
                                    oninput={(event) =>
                                        setControlValue(control, Number(event.currentTarget.value))}
                                />
                            {:else}
                                <TextField
                                    label={controlLabel(control)}
                                    extraClasses="!h-10 !px-3 !py-0"
                                    value={String(controlValue(control) ?? "")}
                                    oninput={(event) =>
                                        setControlValue(control, event.currentTarget.value)}
                                ></TextField>
                            {/if}
                        </div>
                    {/each}
                </div>
            {/if}
            {#if routingEnabled && options.autoRouting}
                <div class="mt-4 border-t border-input-border pt-4 space-y-3 min-w-72 max-w-md">
                    {#if provenanceMismatch}
                        <div class="rounded-md border border-orange-400 p-3 text-sm space-y-2">
                            <p>{$_("routing-provenance-mismatch")}</p>
                            <div class="flex flex-wrap gap-2">
                                <button class="btn-primary !py-1 !px-3" type="button" onclick={onRerouteExisting}>
                                    {$_("routing-reroute-existing")}
                                </button>
                                <button class="btn-secondary !py-1 !px-3" type="button" onclick={onKeepExisting}>
                                    {$_("routing-keep-existing")}
                                </button>
                            </div>
                        </div>
                    {/if}
                    <div class="grid grid-cols-1 gap-3">
                        {#if routingEngines.length > 1}
                            <div class="text-sm">
                                <SingleSelect
                                    name="route-editor-routing-engine"
                                    label={$_("routing-engine-settings")}
                                    items={routingEngines.map((engine) => ({
                                        text: engine.name,
                                        value: engine.pluginId,
                                    }))}
                                    value={options.routingPluginId ??
                                        routingEngines[0]?.pluginId ??
                                        ""}
                                    onchange={(pluginId) =>
                                        onRoutingEngineChange?.(String(pluginId))}
                                ></SingleSelect>
                            </div>
                        {/if}
                        <div class="text-sm">
                            <SingleSelect
                                name="route-editor-routing-mode"
                                label={$_("routing-mode")}
                                items={routingModeItems()}
                                value={options.routingMode ?? "segment"}
                                onchange={selectRoutingMode}
                            ></SingleSelect>
                        </div>
                    </div>
                </div>
            {/if}
        </div>
    {/if}

    {#if crop}
        <div
            class="p-4 my-2 rounded-xl bg-background shadow-xl min-w-72 flex flex-col"
        >
            <DoubleSlider onupdate={onUpdateCropRange}></DoubleSlider>
            <button
                class="btn-secondary mb-2"
                onclick={() => {
                    crop = false;
                    onCrop();
                    onCropToggle(false);
                }}>{$_("crop")}</button
            >
        </div>
    {/if}

    {#if routingEnabled && recalculateElevationData}
        <div
            class="p-4 my-2 rounded-xl bg-background shadow-xl flex flex-col max-w-70"
        >
            <Button secondary onclick={onRecalculateElevationData}
                >{$_("recalculate-elevation-data")}</Button
            >
            <p class="bg-background/50 rounded-xl text-sm text-gray-500 mt-3">
                {$_("recalculating-elevation-data-hint")}
            </p>
        </div>
    {/if}
</div>
