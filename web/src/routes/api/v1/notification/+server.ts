import type { Notification } from '$lib/models/notification';
import { Collection, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/notification:
 *   get:
 *     summary: List notifications
 *     tags:
 *       - Notifications
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *       - in: query
 *         name: sort
 *         schema:
 *           type: string
 *       - in: query
 *         name: filter
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: List of notifications
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<Notification>(event, Collection.notifications);

        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}
