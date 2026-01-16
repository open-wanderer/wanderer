import { TrailLinkShareUpdateSchema } from "$lib/models/api/trail_link_share_schema";
import type { TrailLinkShare } from "$lib/models/trail_link_share";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function GET(event: RequestEvent) {
    try {
        const r = await show<TrailLinkShare>(event, Collection.trail_link_share)
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

export async function POST(event: RequestEvent) {
    try {
        const r = await update<TrailLinkShare>(event, TrailLinkShareUpdateSchema, Collection.trail_link_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.trail_link_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
