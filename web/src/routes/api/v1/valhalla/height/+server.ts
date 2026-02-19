import { getValhallaBaseUrl } from '$lib/server/valhalla';
import { proxyJsonResponse } from '$lib/server/http';
import { json, type RequestEvent } from "@sveltejs/kit";


export async function POST(event: RequestEvent) {
    const baseUrl = getValhallaBaseUrl();
    const data = await event.request.json()
    if (!baseUrl) {
        return json({ message: "VALHALLA_URL not set" }, { status: 400 })
    }
    try {
        const response = await event.fetch(baseUrl + '/height', { method: "POST", body: JSON.stringify(data) });
        return await proxyJsonResponse(response);
    } catch (e: any) {
        return json({ message: "Valhalla request failed" }, { status: 502 })
    }
}
