import type { RequestEvent } from "@sveltejs/kit";
import { proxyJsonResponse } from "#lib/server/http.ts";
import { fetchOverpass } from "#lib/server/overpass.ts";

export async function GET(event: RequestEvent) {
    const data = event.url.searchParams.get("data");
    if (!data) {
        return Response.json({ message: "Missing query parameter: data" }, { status: 400 });
    }

    const params = new URLSearchParams({
        data,
    });

    try {
        const response = await fetchOverpass(event, params);
        const retryAfter = response.headers.get("Retry-After");
        const proxied = await proxyJsonResponse(response);
        if (retryAfter && !proxied.ok) {
            proxied.headers.set("Retry-After", retryAfter);
        }
        return proxied;
    } catch (error) {
        return Response.json({ message: "Overpass request failed" }, { status: 502 });
    }
}
