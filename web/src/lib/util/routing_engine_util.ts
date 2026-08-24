import type {
    RoutingEffectiveControls,
    RoutingEngine,
    RoutingOptions,
} from "$lib/models/routing";

function supportsRoute(engine: RoutingEngine) {
    return engine.enabled && engine.roles?.includes("route");
}

export function routingEngineSupportsVia(engine: RoutingEngine) {
    const routing = engine.metadata?.routing;
    return (
        typeof routing === "object" &&
        routing !== null &&
        (routing as Record<string, unknown>).supportsViaRouting === true
    );
}

export function selectEnabledRoutingPlugin(
    engines: RoutingEngine[],
    preferredPluginId: string | undefined,
    role: string,
) {
    const supportsRole = (engine: RoutingEngine) =>
        engine.enabled && engine.roles?.includes(role);
    if (
        preferredPluginId &&
        engines.some(
            (engine) => engine.pluginId === preferredPluginId && supportsRole(engine),
        )
    ) {
        return preferredPluginId;
    }
    return engines.find(supportsRole)?.pluginId ?? "";
}

export function routingModeToTransport(
    mode: string | undefined,
): RoutingOptions["modeOfTransport"] | undefined {
    switch (mode) {
        case "foot":
            return "pedestrian";
        case "bike":
            return "bicycle";
        case "motor":
            return "auto";
        default:
            return undefined;
    }
}

export function applyRoutingEffectiveControlDefaults(
    options: RoutingOptions,
    _controls: RoutingEffectiveControls,
) {
    // Effective-control defaults are display fallbacks. Copying them into the
    // request would turn an untouched slider into an explicit profile override.
    options.preferences ??= {};
    options.nativeConfig ??= {};
}

export function applyRouteEditorRoutingEngineSelection(
    options: RoutingOptions,
    engines: RoutingEngine[],
    pluginId: string,
) {
    if (!pluginId || pluginId === options.routingPluginId) return undefined;
    const engine = engines.find(
        (candidate) => candidate.pluginId === pluginId && supportsRoute(candidate),
    );
    if (!engine) return undefined;

    options.routingPluginId = engine.pluginId;
    options.routingInstanceId = engine.instanceId;
    options.nativeProfileKey = undefined;
    options.profileRevisions = {};
    options.preferences = {};
    options.nativeConfig = {};
    if (options.routingMode === "via" && !routingEngineSupportsVia(engine)) {
        options.routingMode = "segment";
        options.routingModeExplicit = true;
    }
    return engine;
}
