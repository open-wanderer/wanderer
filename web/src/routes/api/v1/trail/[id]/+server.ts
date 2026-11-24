import { TrailUpdateSchema } from '#lib/models/api/trail_schema.ts';
import type { Trail } from "#lib/models/trail.ts";
import { Collection, handleError, remove, update } from "#lib/util/api_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import type PocketBase from "pocketbase";
import { acceptsActivityPub, activityPubObject } from "#lib/server/activitypub_objects.ts";

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

    if (acceptsActivityPub(event.request)) {
        return activityPubObject(event, `/api/v1/activitypub/trail/${params.id}`);
    }

    try {
        let trail: Trail = await event.locals.pb.send(`/remote/trail/${params.id}?` + url.searchParams, {
            method: "GET",
            fetch: event.fetch,
        })

        await enrichRecord(event.locals.pb, trail);
        trail.expand?.waypoints_via_trail?.sort((a, b) => (a.distance_from_start ?? 0) - (b.distance_from_start ?? 0))
        return Response.json(trail, { headers: { vary: "Accept" } })
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/trail/{id}:
 *   post:
 *     summary: Update a trail
 *     tags: [Trails]
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             allOf:
 *               - $ref: '#/components/schemas/TrailUpdateInput'
 *               - type: object
 *                 required: [name]
 *     responses:
 *       200:
 *         description: Updated trail
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Invalid trail update
 *       401:
 *         description: Authentication required
 *       403:
 *         description: Trail update is not allowed
 *       404:
 *         description: Trail not found
 *       500:
 *         description: Trail update failed
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<Trail>(event, TrailUpdateSchema, Collection.trails)
        await enrichRecord(event.locals.pb, r)
        return Response.json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail/{id}:
 *   delete:
 *     summary: Delete a trail
 *     tags: [Trails]
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Deleted trail
 *       401:
 *         description: Authentication required
 *       403:
 *         description: Trail deletion is not allowed
 *       404:
 *         description: Trail not found
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.trails)
        return Response.json(r);
    } catch (e: any) {
        return handleError(e)
    }
}



async function enrichRecord(pb: PocketBase, r: Trail) {
    r.date = r.date?.substring(0, 10) ?? "";
    for (const log of r.expand?.summit_logs_via_trail ?? []) {
        log.date = log.date.substring(0, 10);
    }
}