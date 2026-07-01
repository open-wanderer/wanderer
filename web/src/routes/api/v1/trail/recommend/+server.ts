import { TrailRecommendSchema } from '$lib/models/api/trail_schema';
import { withTrailPreferenceMeiliFilter } from '$lib/server/category_preference_filter';
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

        const size = safeSearchParams.size ?? 10;
        const response = await recommend(event, size);

        return json(response.hits)
    } catch (e: any) {
        return handleError(e);
    }
}

async function recommend(event: RequestEvent, size: number) {
    const filter = await withTrailPreferenceMeiliFilter(event, undefined);
    const countResponse = await event.locals.ms.index("trails").search("", { limit: 1, filter });
    const numberOfTrails = countResponse.estimatedTotalHits ?? countResponse.totalHits ?? 0;

    if (numberOfTrails === 0 || size === 0) {
        return { hits: [] };
    }

    const maxOffset = Math.max(0, numberOfTrails - size);
    const randomOffset = Math.floor(Math.random() * (maxOffset + 1));
    return event.locals.ms.index("trails").search("", { limit: size, offset: randomOffset, filter });
}
