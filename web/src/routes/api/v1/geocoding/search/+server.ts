import { json, type RequestEvent } from "@sveltejs/kit";
import { handleError } from "$lib/util/api_util";

export async function GET(event: RequestEvent) {
    const q = event.url.searchParams.get("q");
    if (!q) {
        return json({ message: "Missing query parameter: q" }, { status: 400 });
    }

    const limit = event.url.searchParams.get("limit");
    if (limit !== null && Number.isNaN(Number(limit))) {
        return json({ message: "Invalid query parameter: limit" }, { status: 400 });
    }

    const params = new URLSearchParams({ q });
    if (limit) {
        params.set("limit", limit);
    }

    try {
        const response = await event.locals.pb.send(`/geocoding/search?${params}`, {
            method: "GET",
            fetch: event.fetch,
        });
        return json(response);
    } catch (error) {
        return handleError(error);
    }
}
