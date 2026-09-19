import type { RoutingControl } from "$lib/models/routing";

const STANDARD_ROUTING_CONTROLS = new Set([
    "speedPreference",
    "hillPreference",
    "maxHikingDifficulty",
    "roadPreference",
    "avoidBadSurfaces",
    "vehicleWidth",
    "vehicleHeight",
]);

// Routing control state is a JSON contract, but Svelte wraps nested state in
// proxies that the browser's structuredClone rejects. Serializing through the
// same JSON boundary used by the HTTP request unwraps those proxies and keeps
// the returned object detached from editor state.
export function cloneRoutingControlValues(
    values: Record<string, unknown> | undefined,
): Record<string, unknown> {
    return JSON.parse(JSON.stringify(values ?? {})) as Record<string, unknown>;
}

export function parseFiniteRoutingControlNumber(value: unknown): number | undefined {
    if (typeof value === "number") return Number.isFinite(value) ? value : undefined;
    if (typeof value !== "string" || value.trim() === "") return undefined;

    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
}

export function isStandardRoutingControl(control: RoutingControl) {
    return control.target !== "native_config" && STANDARD_ROUTING_CONTROLS.has(control.key);
}

export function routingControlTranslationKey(control: RoutingControl) {
    if (!isStandardRoutingControl(control)) return undefined;
    return `routing-control-${control.key}`;
}

function routingPreferenceBucket(value: number) {
    if (value <= 0.05) return "none";
    if (value < 0.35) return "low";
    if (value < 0.7) return "medium";
    return "high";
}

export function routingControlBucketTranslationKey(
    control: RoutingControl,
    value: number,
) {
    if (!isStandardRoutingControl(control)) return undefined;

    switch (control.key) {
        case "hillPreference":
            return `routing-preference-hills-${routingPreferenceBucket(value)}`;
        case "roadPreference":
            return `routing-preference-roads-${routingPreferenceBucket(value)}`;
        case "avoidBadSurfaces":
            return `routing-preference-surfaces-${
                value <= 0.000001 ? "none" : routingPreferenceBucket(value)
            }`;
        default:
            return undefined;
    }
}

export function formatRoutingControlUnit(
    control: RoutingControl,
    value: number,
    formatUserSpeed: (metersPerSecond: number) => string,
) {
    const unit = control.unit?.trim();
    if (!unit) return undefined;

    switch (unit.toLowerCase()) {
        case "km/h":
        case "kph":
            return formatUserSpeed(value / 3.6);
        case "m/s":
            return formatUserSpeed(value);
        default:
            return `${value} ${unit}`;
    }
}
