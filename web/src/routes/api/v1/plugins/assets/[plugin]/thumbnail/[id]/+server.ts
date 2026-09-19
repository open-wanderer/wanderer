import { error, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const paramsSchema = z.object({
    plugin: z.string().min(1).max(128).regex(/^[a-zA-Z0-9._-]+$/),
    id: z.string().min(1).max(256),
});

export async function GET(event: RequestEvent) {
    const safeParams = paramsSchema.safeParse(event.params);

    if (!safeParams.success) {
        throw error(400, "Invalid asset plugin thumbnail request");
    }

    const headers: Record<string, string> = {};
    const token = event.locals.pb.authStore.token;
    if (token) {
        headers.Authorization = `Bearer ${token}`;
    }
    const ifNoneMatch = event.request.headers.get("If-None-Match");
    if (ifNoneMatch) {
        headers["If-None-Match"] = ifNoneMatch;
    }

    const thumbnailURL = event.locals.pb.buildURL(
        `plugins/assets/${encodeURIComponent(safeParams.data.plugin)}/thumbnail/${encodeURIComponent(safeParams.data.id)}`,
    );
    const response = await event.fetch(thumbnailURL, { headers });

    if (!response.ok && response.status !== 304) {
        throw error(response.status, "Thumbnail not available");
    }

    return new Response(response.body, {
        headers: response.headers,
        status: response.status,
    });
}
