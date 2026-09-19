import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const bodySchema = z.object({
    trailId: z.string().min(1).max(64),
});

export async function POST(event: RequestEvent) {
    try {
        const data = bodySchema.parse(await event.request.json());
        const r = await event.locals.pb.send("/plugins/assets/maintenance/attach", {
            method: "POST",
            body: JSON.stringify(data),
            headers: { "Content-Type": "application/json" },
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
