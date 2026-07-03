import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const paramsSchema = z.object({
    id: z.string().min(1).max(64).regex(/^[a-fA-F0-9]+$/),
});

export async function GET(event: RequestEvent) {
    try {
        const params = paramsSchema.parse(event.params);
        const r = await event.locals.pb.send(
            `/plugins/assets/jobs/materialize/${encodeURIComponent(params.id)}`,
            { method: "GET" },
        );
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
