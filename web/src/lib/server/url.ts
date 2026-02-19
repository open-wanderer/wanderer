import { env as privateEnv } from "$env/dynamic/private";
import { env as publicEnv } from "$env/dynamic/public";

export type ExternalServiceUrlKey = "VALHALLA_URL" | "NOMINATIM_URL" | "OVERPASS_API_URL";

export function normalizeBaseUrl(url: string): string {
    if (!/^https?:\/\//i.test(url)) {
        return `https://${url}`;
    }
    return url;
}

export function resolveBaseUrl(
    key: ExternalServiceUrlKey,
    fallback: string = "",
): string {
    const privateKey = `PRIVATE_${key}` as `PRIVATE_${string}`;
    const publicKey = `PUBLIC_${key}` as `PUBLIC_${string}`;
    const rawUrl = privateEnv[privateKey] ?? publicEnv[publicKey] ?? fallback;
    return normalizeBaseUrl(rawUrl);
}
