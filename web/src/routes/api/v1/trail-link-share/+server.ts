import { TrailLinkShareCreateSchema } from '$lib/models/api/trail_link_share_schema';
import type { TrailLinkShare } from '$lib/models/trail_link_share';
import { Collection, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent) {
    try {
        const r = await list<TrailLinkShare>(event, Collection.trail_link_share);
        return json(r)
    } catch (e: any) {
        handleError(e)
    }
}

export async function PUT(event: RequestEvent) {
    try {
        const data = await event.request.json();
        const safeData = TrailLinkShareCreateSchema.parse(data);

        const r = await event.locals.pb.collection(Collection.trail_link_share).create<TrailLinkShare>(safeData)

        return json(r);
    } catch (e) {
        return handleError(e)
    }
}