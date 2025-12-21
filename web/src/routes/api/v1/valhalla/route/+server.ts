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

async function fetchHeights(event: RequestEvent, shape?: string) {
    if (!shape) {
        return null;
    }
    const response = await event.fetch(env.PUBLIC_VALHALLA_URL + '/height', {
        method: "POST",
        body: JSON.stringify({ encoded_polyline: shape })
    });
    const payload = await response.json();
    if (!response.ok) {
        throw error(response.status as NumericRange<400, 500>, payload);
    }
    if (!Array.isArray(payload?.height)) {
        return null;
    }
    return payload.height as number[];
}

export async function POST(event: RequestEvent) {
    const data: RouteRequestBody = await event.request.json();
    if (!env.PUBLIC_VALHALLA_URL) {
        return json({ message: "PUBLIC_VALHALLA_URL not set" }, { status: 400 })
    }

    const { include_elevation_profile, ...forwardData } = data ?? {};

    try {
        const route = await fetchRoute(event, forwardData);

        if (include_elevation_profile) {
            const legs = route?.trip?.legs;
            if (Array.isArray(legs) && legs.length > 0) {
                const heightProfile = await fetchHeights(event, legs[0]?.shape);
                if (Array.isArray(heightProfile)) {
                    legs[0] = { ...legs[0], heights: heightProfile };
                }
            }
        }

        return json(route);
    } catch (e: any) {
        const status = typeof e?.status === "number" ? e.status : 500;
        const message = e?.body ?? e?.message ?? e;
        return json({ message }, { status })
    }
}
