import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const paramsSchema = z.object({
    plugin: z.string().min(1).max(128).regex(/^[a-zA-Z0-9._-]+$/),
});

export async function POST(event: RequestEvent) {
    try {
        const params = paramsSchema.parse(event.params);
        const r = await event.locals.pb.send(
            `/plugins/assets/${encodeURIComponent(params.plugin)}/repair-remote-assets`,
            { method: "POST" },
        );
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
