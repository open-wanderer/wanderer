import type { RequestEvent } from "@sveltejs/kit";
import { proxyGeocoding } from "#lib/server/geocoding_proxy.ts";

export async function GET(event: RequestEvent) {
    const q = event.url.searchParams.get("q");
    if (!q) {
        return Response.json({ message: "Missing query parameter: q" }, { status: 400 });
    }

    const limit = event.url.searchParams.get("limit");
    if (limit !== null && Number.isNaN(Number(limit))) {
        return Response.json({ message: "Invalid query parameter: limit" }, { status: 400 });
    }

    const params = new URLSearchParams({ q });
    if (limit) {
        params.set("limit", limit);
    }

    return proxyGeocoding(event, "search", params);
}
