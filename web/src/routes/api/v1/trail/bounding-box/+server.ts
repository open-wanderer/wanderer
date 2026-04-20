import { type TrailBoundingBox, type TrailFilterValues } from '$lib/models/trail';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail/bounding-box:
 *   get:
 *     summary: Get trail bounding box
 *     description: Retrieves geographic bounding box (lat/lon bounds) for user's trails
 *     tags:
 *       - Trails
 *     responses:
 *       200:
 *         description: Bounding box coordinates
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 max_lat:
 *                   type: number
 *                 min_lat:
 *                   type: number
 *                 max_lon:
 *                   type: number
 *                 min_lon:
 *                   type: number
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    if (!event.locals.pb.authStore.record) {
        return json({
            max_lat: 0,
            min_lat: 0,
            max_lon: 0,
            min_lon: 0
        });
    }
    try {
        const r = await event.locals.pb.collection('trails_bounding_box').getOne<TrailBoundingBox>(event.locals.user.actor!)
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}
