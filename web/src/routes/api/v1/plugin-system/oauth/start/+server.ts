import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const r = await event.locals.pb.send("/plugins/oauth/start", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e) {
        return handleError(e);
    }
}
