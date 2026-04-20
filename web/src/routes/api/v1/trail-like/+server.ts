import { TrailLikeCreateSchema } from '$lib/models/api/trail_like_schema';
import type { TrailLike } from '$lib/models/trail_like';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail-like:
 *   get:
 *     summary: List trail likes
 *     tags:
 *       - Trail Likes
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
 *         description: List of trail likes
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
        const r = await list<TrailLike>(event, Collection.trail_share);
        return json(r)
    } catch (e: any) {
        handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-like:
 *   put:
 *     summary: Create trail like
 *     tags:
 *       - Trail Likes
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailLikeInput'
 *     responses:
 *       201:
 *         description: Trail like created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/TrailLike'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<TrailLike>(event, TrailLikeCreateSchema, Collection.trail_like)

        return json(r);
    } catch (e) {
        return handleError(e)
    }
}