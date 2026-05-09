import { resolveBaseUrl } from "$lib/server/url";

export function getValhallaBaseUrl(): string {
    return resolveBaseUrl("VALHALLA_URL");
}
