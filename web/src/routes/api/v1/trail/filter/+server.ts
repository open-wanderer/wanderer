import { type TrailFilterValues } from '$lib/models/trail';
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail/filter:
 *   get:
 *     summary: Get trail filter values
 *     description: Retrieves min/max values for trail filtering (distance, elevation gain/loss)
 *     tags:
 *       - Trails
 *     responses:
 *       200:
 *         description: Trail filter values (min/max for distance, elevation)
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 min_distance:
 *                   type: number
 *                 max_distance:
 *                   type: number
 *                 min_elevation_gain:
 *                   type: number
 *                 max_elevation_gain:
 *                   type: number
 *                 min_elevation_loss:
 *                   type: number
 *                 max_elevation_loss:
 *                   type: number
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    if (!event.locals.pb.authStore.record) {
        return json({
            min_distance: 0,
            max_distance: 20000,
            min_elevation_gain: 0,
            max_elevation_gain: 4000,
            min_elevation_loss: 0,
            max_elevation_loss: 4000
        });
    }
    try {
        const r = await event.locals.pb.collection('trails_filter').getOne<TrailFilterValues>(event.locals.user.actor)
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}
