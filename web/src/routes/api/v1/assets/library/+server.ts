import { RecordIdValueSchema } from "$lib/models/api/base_schema";
import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const LibraryRequestSchema = z.object({
    trailId: RecordIdValueSchema.optional(),
    waypointId: RecordIdValueSchema.optional(),
    summitLogId: RecordIdValueSchema.optional(),
    lat: z.number().optional(),
    lon: z.number().optional(),
    trailData: z.string().optional(),
    takenAfter: z.string().datetime().optional(),
    takenBefore: z.string().datetime().optional(),
    doubleRadius: z.boolean().optional(),
    page: z.number().int().min(1).optional(),
    perPage: z.number().int().min(1).max(250).optional(),
});

export async function POST(event: RequestEvent) {
    try {
        const data = LibraryRequestSchema.parse(await event.request.json().catch(() => ({})));
        const response = await event.locals.pb.send("/assets/library", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(data),
            fetch: event.fetch,
        });

        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}
