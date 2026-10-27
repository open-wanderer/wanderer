import { resolveBaseUrl } from "#lib/server/url.ts";

export function getValhallaBaseUrl(): string {
    return resolveBaseUrl("VALHALLA_URL");
}
