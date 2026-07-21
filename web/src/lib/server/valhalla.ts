import { resolveBaseUrl, type ExternalServiceUrlKey } from "$lib/server/url";

function getServiceUrl(key: ExternalServiceUrlKey): string | null {
    return resolveBaseUrl(key) || null;
}

export function getValhallaRouteUrl(): string | null {
    return getServiceUrl("VALHALLA_ROUTE_URL");
}

export function getValhallaHeightUrl(): string | null {
    return getServiceUrl("VALHALLA_HEIGHT_URL");
}

export function getValhallaNavigateUrl(): string | null {
    return getServiceUrl("VALHALLA_NAVIGATE_URL");
}

export function getValhallaTraceRouteUrl(): string | null {
    return getServiceUrl("VALHALLA_TRACE_ROUTE_URL");
}
