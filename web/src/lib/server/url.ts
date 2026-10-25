import * as privateEnv from "$app/env/private";
import * as publicEnv from "$app/env/public";

export type ExternalServiceUrlKey = "VALHALLA_URL" | "NOMINATIM_URL" | "OVERPASS_API_URL";

export function normalizeBaseUrl(url: string): string {
    const trimmedUrl = url.trim();
    if (!trimmedUrl) {
        return "";
    }
    if (!/^https?:\/\//i.test(trimmedUrl)) {
        return `https://${trimmedUrl}`;
    }
    return trimmedUrl;
}

export function resolveBaseUrl(
    key: ExternalServiceUrlKey,
    fallback: string = "",
): string {
    const publicKey = `PUBLIC_${key}` as const;
    const rawUrl = privateEnv[key] ?? publicEnv[publicKey] ?? fallback;
    return normalizeBaseUrl(rawUrl);
}
