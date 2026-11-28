import type { RequestEvent } from "@sveltejs/kit";
import { proxyGeocoding } from "#lib/server/geocoding_proxy.ts";

export async function GET(event: RequestEvent) {
    const lat = event.url.searchParams.get("lat");
    const lon = event.url.searchParams.get("lon");
    if (!lat || !lon) {
        return Response.json({ message: "Missing query parameter: lat or lon" }, { status: 400 });
    }

    if (Number.isNaN(Number(lat)) || Number.isNaN(Number(lon))) {
        return Response.json({ message: "Invalid query parameter: lat or lon" }, { status: 400 });
    }

    return proxyGeocoding(event, "reverse", new URLSearchParams({ lat, lon }));
}
