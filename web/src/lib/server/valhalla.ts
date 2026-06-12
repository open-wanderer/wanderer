import { resolveBaseUrl } from "$lib/server/url";

export function getValhallaBaseUrl(): string | null {
    const url = resolveBaseUrl("VALHALLA_URL");
    return url || null;
}
