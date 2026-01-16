import { TrailCreateSchema } from '$lib/models/api/trail_schema';
import type { Trail } from '$lib/models/trail';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent) {
    try {
        const r = await list<Trail>(event, Collection.trails);

        for (const t of r.items) {
            if (!t.author || !event.locals.pb.authStore.record) {
                continue;
            }
            if (!t.expand) {
                t.expand = {} as any
            }

            t.expand?.waypoints_via_trail?.sort((a, b) => (a.distance_from_start ?? 0) - (b.distance_from_start ?? 0))
        }
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

export async function PUT(event: RequestEvent) {
    try {        
        const r = await create<Trail>(event, TrailCreateSchema, Collection.trails)
        enrichRecord(r);
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}

function enrichRecord(r: Trail) {
    r.date = r.date?.substring(0, 10) ?? "";
    for (const log of r.expand?.summit_logs_via_trail ?? []) {
        log.date = log.date.substring(0, 10);
    }
}