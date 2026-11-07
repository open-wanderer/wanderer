import { resolveBaseUrl, type ExternalServiceUrlKey } from "#lib/server/url.ts";

function getServiceUrl(key: ExternalServiceUrlKey): string | null {
    return resolveBaseUrl(key) || null;
}

export function getValhallaUrl(): string | null {
    return getServiceUrl("VALHALLA_URL");
}