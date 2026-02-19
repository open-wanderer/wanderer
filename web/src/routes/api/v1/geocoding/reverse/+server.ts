import { json, type RequestEvent } from "@sveltejs/kit";
import { proxyJsonResponse } from "$lib/server/http";
import { fetchNominatim } from "$lib/server/nominatim";

export async function GET(event: RequestEvent) {
    const lat = event.url.searchParams.get("lat");
    const lon = event.url.searchParams.get("lon");
    if (!lat || !lon) {
        return json({ message: "Missing query parameter: lat or lon" }, { status: 400 });
    }

    if (Number.isNaN(Number(lat)) || Number.isNaN(Number(lon))) {
        return json({ message: "Invalid query parameter: lat or lon" }, { status: 400 });
    }

    const params = new URLSearchParams({
        lat,
        lon,
        format: "geojson",
        addressdetails: "1",
    });

    try {
        const response = await fetchNominatim(event, "/reverse", params);
        return await proxyJsonResponse(response);
    } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        const detail = {
            name: err.name,
            message: err.message,
            cause: err.cause instanceof Error ? err.cause.message : err.cause,
        };
        console.error("Nominatim reverse request failed", detail);
        return json({ message: "Nominatim request failed", detail }, { status: 502 });
    }
}
