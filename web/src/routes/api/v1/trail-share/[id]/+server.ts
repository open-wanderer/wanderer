import { TrailShareUpdateSchema } from "$lib/models/api/trail_share_schema";
import type { TrailShare } from "$lib/models/trail_share";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail-share/{id}:
 *   get:
 *     summary: Get trail share
 *     description: Retrieves a trail share by ID
 *     tags:
 *       - Trail Shares
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
 *         description: TrailShare
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<TrailShare>(event, Collection.trail_share)
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-share/{id}:
 *   post:
 *     summary: Update trail share
 *     tags:
 *       - Trail Shares
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
 *             $ref: '#/components/schemas/TrailShareUpdateInput'
 *     responses:
 *       200:
 *         description: Trail share updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/TrailShare'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<TrailShare>(event, TrailShareUpdateSchema, Collection.trail_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-share/{id}:
 *   delete:
 *     summary: Delete trail share
 *     tags:
 *       - Trail Shares
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Trail share deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.trail_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
