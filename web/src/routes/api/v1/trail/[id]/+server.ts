import { TrailUpdateSchema } from '$lib/models/api/trail_schema';
import { applyLegacyPhotoNamesForMissingAssetExpands } from "$lib/server/legacy_photo_compat";
import type { Trail } from "$lib/models/trail";
import { Collection, handleError, remove, update } from "$lib/util/api_util";
import { enrichTrailAssetExpands } from "$lib/util/asset_link_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail/{id}:
 *   get:
 *     summary: Get trail
 *     description: Retrieves a trail by ID
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *       - in: query
 *         name: share
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Trail
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const { url, params } = event;

    try {
        let trail: Trail = await event.locals.pb.send(`/remote/trail/${params.id}?` + url.searchParams, {
            method: "GET",
            fetch: event.fetch,
        })

        await enrichRecord(trail, url.searchParams.get("share") ?? undefined);
        await applyLegacyPhotoNamesForMissingAssetExpands(event, trail);
        trail.expand?.waypoints_via_trail?.sort((a, b) => (a.distance_from_start ?? 0) - (b.distance_from_start ?? 0))
        return json(trail)
    } catch (e: any) {
        return handleError(e);
    }
}

export async function POST(event: RequestEvent) {
    try {
        const r = await update<Trail>(event, TrailUpdateSchema, Collection.trails)
        await enrichRecord(r)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.trails)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}



async function enrichRecord(r: Trail, share?: string) {
    r.date = r.date?.substring(0, 10) ?? "";
    for (const log of r.expand?.summit_logs_via_trail ?? []) {
        log.date = log.date.substring(0, 10);
    }
    enrichTrailAssetExpands(r, { share });
}
