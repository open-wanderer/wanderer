import { env as privateEnv } from '$env/dynamic/private';
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
    return privateEnv.PRIVATE_VALHALLA_URL ?? "";
}
