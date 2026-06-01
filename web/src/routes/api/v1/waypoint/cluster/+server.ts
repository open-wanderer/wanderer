import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const WaypointClusterPointSchema = z.object({
    id: z.string().min(1),
    lat: z.number().min(-90).max(90),
    lon: z.number().min(-180).max(180),
});

const WaypointClusterSchema = z.object({
    category: z.string().length(15).or(z.literal("")).optional(),
    photos: z.array(WaypointClusterPointSchema),
    waypoints: z.array(WaypointClusterPointSchema),
});

export async function POST(event: RequestEvent) {
    try {
        const data = WaypointClusterSchema.parse(await event.request.json());
        const response = await event.locals.pb.send("/waypoint/cluster", {
            method: "POST",
            headers: {
                "content-type": "application/json",
            },
            body: JSON.stringify(data),
            fetch: event.fetch,
        });

        return json(response);
    } catch (e) {
        return handleError(e);
    }
}
