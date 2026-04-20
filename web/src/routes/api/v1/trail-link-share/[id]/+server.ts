import { TrailLinkShareUpdateSchema } from "$lib/models/api/trail_link_share_schema";
import type { TrailLinkShare } from "$lib/models/trail_link_share";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail-link-share/{id}:
 *   get:
 *     summary: Get trail link share
 *     description: Retrieves a trail link share by ID
 *     tags:
 *       - Trail Link Shares
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
 *         description: TrailLinkShare
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<TrailLinkShare>(event, Collection.trail_link_share)
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-link-share/{id}:
 *   post:
 *     summary: Update trail link share
 *     tags:
 *       - Trail Link Shares
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
 *             $ref: '#/components/schemas/TrailLinkShareUpdateInput'
 *     responses:
 *       200:
 *         description: Trail link share updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/TrailLinkShare'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<TrailLinkShare>(event, TrailLinkShareUpdateSchema, Collection.trail_link_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-link-share/{id}:
 *   delete:
 *     summary: Delete trail link share
 *     tags:
 *       - Trail Link Shares
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Trail link share deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.trail_link_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
