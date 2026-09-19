import { json, type RequestEvent } from "@sveltejs/kit";
import { handleError } from "$lib/util/api_util";

export async function GET(event: RequestEvent) {
    const lat = event.url.searchParams.get("lat");
    const lon = event.url.searchParams.get("lon");
    if (!lat || !lon) {
        return json({ message: "Missing query parameter: lat or lon" }, { status: 400 });
    }

    if (Number.isNaN(Number(lat)) || Number.isNaN(Number(lon))) {
        return json({ message: "Invalid query parameter: lat or lon" }, { status: 400 });
    }

    const params = new URLSearchParams({ lat, lon });

    try {
        const response = await event.locals.pb.send(`/geocoding/reverse?${params}`, {
            method: "GET",
            fetch: event.fetch,
        });
        return json(response);
    } catch (error) {
        return handleError(error);
    }
}
