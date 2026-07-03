import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function GET(event: RequestEvent) {
    try {
        const r = await event.locals.pb.send("/plugins/assets/maintenance/trails", {
            method: "GET",
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
