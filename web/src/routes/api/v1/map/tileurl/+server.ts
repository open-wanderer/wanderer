import { env } from '$env/dynamic/private';
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/map/tileurl:
 *   get:
 *     summary: Get the configured tile server URL
 *     description: >
 *       Returns the vector tile URL template for this instance.
 *       Requires authentication. Operators can override the default by setting
 *       TILE_SERVER_URL; otherwise falls back to the built-in Protomaps endpoint.
 *     tags:
 *       - Maps
 *     responses:
 *       200:
 *         description: Tile URL template
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 url:
 *                   type: string
 *                   example: "https://tiles.example.com/{z}/{x}/{y}.mvt"
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.locals.pb.authStore.record) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }

        const tileUrl = env.TILE_SERVER_URL
            ?? `https://api.protomaps.com/tiles/v4/{z}/{x}/{y}.mvt?key=${env.PROTOMAPS_API_KEY ?? ''}`;

        return json({ url: tileUrl });
    } catch (e) {
        return handleError(e);
    }
}
