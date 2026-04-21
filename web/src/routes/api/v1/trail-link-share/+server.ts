import { TrailLinkShareCreateSchema } from '$lib/models/api/trail_link_share_schema';
import type { TrailLinkShare } from '$lib/models/trail_link_share';
import { Collection, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail-link-share:
 *   get:
 *     summary: List trail link shares
 *     tags:
 *       - Trail Link Shares
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
 *         description: List of trail link shares
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
        const r = await list<TrailLinkShare>(event, Collection.trail_link_share);
        return json(r)
    } catch (e: any) {
        handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-link-share:
 *   put:
 *     summary: Create trail link share
 *     tags:
 *       - Trail Link Shares
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailLinkShareInput'
 *     responses:
 *       201:
 *         description: Trail link share created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/TrailLinkShare'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
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