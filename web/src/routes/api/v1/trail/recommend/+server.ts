import { TrailRecommendSchema } from '$lib/models/api/trail_schema';
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail/recommend:
 *   get:
 *     summary: Get trail recommendations
 *     description: Retrieves random trail recommendations from Meilisearch
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: query
 *         name: size
 *         schema:
 *           type: integer
 *           default: 10
 *     responses:
 *       200:
 *         description: Array of recommended trails
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const searchParams = Object.fromEntries(event.url.searchParams);
        const safeSearchParams = TrailRecommendSchema.parse(searchParams);

        const numberOfTrails = (await event.locals.ms.index("trails").search("", {limit: 1})).estimatedTotalHits
        const randomOffset = (safeSearchParams.size ?? 0) > numberOfTrails ? 0 : Math.floor(Math.random() * (numberOfTrails - 1) + 1)
        const response = await event.locals.ms.index("trails").search("", {limit: safeSearchParams.size, offset: randomOffset})

        return json(response.hits)
    } catch (e: any) {
        return handleError(e);
    }
}
