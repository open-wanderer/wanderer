import type { RoutingOptions } from "$lib/models/routing";
import { APIError } from "$lib/util/api_util";
import { cloneRoutingControlValues } from "$lib/util/routing_control_util";
import {
    ROUTING_MAX_ENGINES,
    ROUTING_MAX_VARIANT_ANCHORS,
    ROUTING_MAX_VARIANTS,
} from "$lib/util/routing_variant_util";

type Translate = (key: string, options?: any) => string;

export function routeCalculationErrorText(error: unknown, translate: Translate) {
    if (error instanceof APIError) {
        const code = nestedRoutingErrorValue(error.detail, "code");
        if (code === "routing_disabled") return translate("routing-error-disabled");
        if (code === "anchor_limit_exceeded") {
            return translate("routing-error-anchor-limit", {
                values: {
                    max:
                        nestedRoutingErrorNumber(error.detail, "maxAnchors") ??
                        ROUTING_MAX_VARIANT_ANCHORS,
                },
            });
        }
        if (code === "fanout_limit_exceeded") {
            const max = nestedRoutingErrorNumber(error.detail, "maxWork") ?? 0;
            return translate("routing-error-fanout-limit", {
                values: {
                    work: nestedRoutingErrorNumber(error.detail, "work") ?? `>${max}`,
                    max,
                },
            });
        }
        if (code === "profile_preparation_fanout_limit_exceeded") {
            const max = nestedRoutingErrorNumber(error.detail, "maxWork") ?? 0;
            return translate("routing-error-profile-preparation-fanout-limit", {
                values: {
                    work: nestedRoutingErrorNumber(error.detail, "work") ?? `>${max}`,
                    max,
                },
            });
        }
        if (code === "variant_limit_exceeded") {
            return translate("routing-error-variant-limit", {
                values: { max: ROUTING_MAX_VARIANTS },
            });
        }
        if (code === "engine_limit_exceeded") {
            return translate("routing-error-engine-limit", {
                values: { max: ROUTING_MAX_ENGINES },
            });
        }
        if (code === "provider_timeout") return translate("routing-error-provider-timeout");
        if (
            code === "provider_error" ||
            code === "provider_unavailable" ||
            code === "connector_error"
        ) {
            return translate("routing-error-provider-unavailable");
        }
    }
    if (error instanceof Error && error.message) return error.message;
    return "Error calculating route";
}

export function routingPlanningKey(options: RoutingOptions) {
    return JSON.stringify({
        category: options.category,
        subcategory: options.subcategory,
        pluginId: options.routingPluginId,
        instanceId: options.routingInstanceId,
        routingMode: options.routingMode,
        profileRevisions: options.profileRevisions,
        preferences: options.preferences,
        nativeConfig: options.nativeConfig,
    });
}

export function routingPreparationOptions(options: RoutingOptions): RoutingOptions {
    return {
        autoRouting: options.autoRouting,
        category: options.category,
        subcategory: options.subcategory,
        modeOfTransport: options.modeOfTransport,
        routingPluginId: options.routingPluginId,
        routingInstanceId: options.routingInstanceId,
        routingMode: options.routingMode,
        routingModeExplicit: options.routingModeExplicit,
        engineMode: options.engineMode,
        desiredVariants: options.desiredVariants,
        nativeProfileKey: options.nativeProfileKey,
        profileRevisions: { ...(options.profileRevisions ?? {}) },
        preferences: cloneRoutingControlValues(options.preferences),
        nativeConfig: cloneRoutingControlValues(options.nativeConfig),
    };
}

function nestedRoutingErrorValue(value: unknown, key: string): string | undefined {
    if (!value || typeof value !== "object") return undefined;
    const record = value as Record<string, unknown>;
    if (typeof record[key] === "string") return record[key];
    for (const nested of Object.values(record)) {
        const result = nestedRoutingErrorValue(nested, key);
        if (result !== undefined) return result;
    }
    return undefined;
}

function nestedRoutingErrorNumber(value: unknown, key: string): number | undefined {
    if (!value || typeof value !== "object") return undefined;
    const record = value as Record<string, unknown>;
    if (typeof record[key] === "number") return record[key];
    for (const nested of Object.values(record)) {
        const result = nestedRoutingErrorNumber(nested, key);
        if (result !== undefined) return result;
    }
    return undefined;
}
