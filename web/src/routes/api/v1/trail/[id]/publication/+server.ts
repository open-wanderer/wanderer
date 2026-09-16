import { RecordIdSchema } from "$lib/models/api/base_schema";
import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

async function publication(event: RequestEvent, method: "GET" | "POST") {
    try {
        const { id } = RecordIdSchema.parse(event.params);
        const job = await event.locals.pb.send(`/trails/${id}/publication`, {
            method,
            fetch: event.fetch,
            requestKey: null,
        });
        return json(job, { status: method === "POST" ? 202 : 200, headers: { "Cache-Control": "private, no-store" } });
    } catch (error) {
        return handleError(error);
    }
}

export const GET = (event: RequestEvent) => publication(event, "GET");
export const POST = (event: RequestEvent) => publication(event, "POST");
