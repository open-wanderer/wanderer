import { json, type RequestEvent } from "@sveltejs/kit";
import { proxyJsonResponse } from "$lib/server/http";
import { fetchOverpass } from "$lib/server/overpass";

export async function GET(event: RequestEvent) {
    const data = event.url.searchParams.get("data");
    if (!data) {
        return json({ message: "Missing query parameter: data" }, { status: 400 });
    }

    const params = new URLSearchParams({
        data,
    });

    try {
        const response = await fetchOverpass(event, params);
        return await proxyJsonResponse(response);
    } catch (error) {
        return json({ message: "Overpass request failed" }, { status: 502 });
    }
}
