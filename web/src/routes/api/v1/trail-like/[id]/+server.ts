import type { TrailLike } from "$lib/models/trail_like";
import { Collection, handleError, remove, show } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail-like/{id}:
 *   get:
 *     summary: Get trail like
 *     description: Retrieves a trail like by ID
 *     tags:
 *       - Trail Likes
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
 *     responses:
 *       200:
 *         description: TrailLike
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<TrailLike>(event, Collection.trail_like)
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-like/{id}:
 *   delete:
 *     summary: Delete trail like
 *     tags:
 *       - Trail Likes
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Trail like deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.trail_like)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
