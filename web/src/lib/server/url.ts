import { env as privateEnv } from "$env/dynamic/private";
import { env as publicEnv } from "$env/dynamic/public";

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
    const publicKey = `PUBLIC_${key}` as `PUBLIC_${string}`;
    const rawUrl = privateEnv[key] ?? publicEnv[publicKey] ?? fallback;
    return normalizeBaseUrl(rawUrl);
}
