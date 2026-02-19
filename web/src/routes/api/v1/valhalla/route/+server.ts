import { getValhallaBaseUrl } from '$lib/server/valhalla';
import { proxyJsonResponse } from '$lib/server/http';
import { json, type RequestEvent } from "@sveltejs/kit";

type RouteRequestBody = Record<string, unknown> & {
    include_elevation_profile?: boolean;
};

export async function POST(event: RequestEvent) {
    const baseUrl = getValhallaBaseUrl();
    const data: RouteRequestBody = await event.request.json();
    if (!baseUrl) {
        return json({ message: "VALHALLA_URL not set" }, { status: 400 })
    }

    try {
        const response = await event.fetch(baseUrl + '/route', {
            method: "POST",
            body: JSON.stringify(data)
        });
        return await proxyJsonResponse(response);
    } catch (e: any) {
        return json({ message: "Valhalla request failed" }, { status: 502 })
    }
}
