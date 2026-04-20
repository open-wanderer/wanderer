import { TrailLikeCreateSchema } from '$lib/models/api/trail_like_schema';
import { Collection, handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';


/**
 * @swagger
 * /api/v1/trail-like/delete:
 *   post:
 *     summary: Delete trail like by actor and trail
 *     description: Deletes a trail like without needing to know its ID. Uses actor and trail IDs to find and delete
 *     tags:
 *       - Trail Likes
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailLikeInput'
 *     responses:
 *       200:
 *         description: Trail like deleted
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {

        const data = await event.request.json();
        const safeData = TrailLikeCreateSchema.parse(data);

        const like = await event.locals.pb.collection(Collection.trail_like).getFirstListItem(`actor='${safeData.actor}'&&trail='${safeData.trail}'`)

        const r = await event.locals.pb.collection(Collection.trail_like).delete(like.id)

        return json({ 'acknowledged': r })
    } catch (e: any) {
        handleError(e)
    }
}