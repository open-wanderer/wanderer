import { env } from '$env/dynamic/public';
import { error, json, type NumericRange, type RequestEvent } from "@sveltejs/kit";

type RouteRequestBody = Record<string, unknown> & {
    include_elevation_profile?: boolean;
};

async function fetchRoute(event: RequestEvent, body: Record<string, unknown>) {
    const response = await event.fetch(env.PUBLIC_VALHALLA_URL + '/route', {
        method: "POST",
        body: JSON.stringify(body)
    });
    const payload = await response.json();
    if (!response.ok) {
        throw error(response.status as NumericRange<400, 500>, payload);
    }
    return payload;
}

export async function POST(event: RequestEvent) {
    const data: RouteRequestBody = await event.request.json();
    if (!env.PUBLIC_VALHALLA_URL) {
        return json({ message: "PUBLIC_VALHALLA_URL not set" }, { status: 400 })
    }

    try {
        const route = await fetchRoute(event, data);
        return json(route);
    } catch (e: any) {
        const status = typeof e?.status === "number" ? e.status : 500;
        const message = e?.body ?? e?.message ?? e;
        return json({ message }, { status })
    }
}
