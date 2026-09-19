import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function POST(event: RequestEvent) {
    try {
        const data = await event.request.json();
        const r = await event.locals.pb.send("/plugins/assets/auto-attach", {
            method: "POST",
            body: JSON.stringify(data),
            headers: { "Content-Type": "application/json" },
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
