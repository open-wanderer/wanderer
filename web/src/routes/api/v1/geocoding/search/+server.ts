import { json, type RequestEvent } from "@sveltejs/kit";
import { proxyJsonResponse } from "$lib/server/http";
import { fetchNominatim } from "$lib/server/nominatim";

export async function GET(event: RequestEvent) {
    const q = event.url.searchParams.get("q");
    if (!q) {
        return json({ message: "Missing query parameter: q" }, { status: 400 });
    }

    const limit = event.url.searchParams.get("limit");
    if (limit !== null && Number.isNaN(Number(limit))) {
        return json({ message: "Invalid query parameter: limit" }, { status: 400 });
    }

    const params = new URLSearchParams({
        q,
        format: "geojson",
        addressdetails: "1",
    });
    if (limit) {
        params.set("limit", limit);
    }

    try {
        const response = await fetchNominatim(event, "/search", params);
        return await proxyJsonResponse(response);
    } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        const detail = {
            name: err.name,
            message: err.message,
            cause: err.cause instanceof Error ? err.cause.message : err.cause,
        };
        console.error("Nominatim search request failed", detail);
        return json({ message: "Nominatim request failed", detail }, { status: 502 });
    }
}
