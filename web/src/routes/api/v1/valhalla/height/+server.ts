import { env as privateEnv } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';
import { error, json, type NumericRange, type RequestEvent } from "@sveltejs/kit";


export async function POST(event: RequestEvent) {
    const data = await event.request.json()
    if (!getValhallaBaseUrl()) {
        return error(400, "VALHALLA_URL not set")
    }
    try {
        const r = await event.fetch(getValhallaBaseUrl() + '/height', { method: "POST", body: JSON.stringify(data) });        
        const response = await r.json();
        if (!r.ok) {
            throw error(r.status as NumericRange<400,500>, response);
        }
        return json(response);
    } catch (e: any) {
        throw error(e.status || 500, e)
    }
}

function getValhallaBaseUrl(): string {
    const rawUrl =
        privateEnv.PRIVATE_VALHALLA_URL ??
        publicEnv.PUBLIC_VALHALLA_URL ??
        "";
    return normalizeBaseUrl(rawUrl);
}

function normalizeBaseUrl(url: string): string {
    if (!/^https?:\/\//i.test(url)) {
        return `https://${url}`;
    }
    return url;
}
