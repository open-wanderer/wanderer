import { env } from '$env/dynamic/private';
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/map/style-sources:
 *   get:
 *     summary: Get map style engine sources
 *     description: >
 *       Returns the vector tile URL template, glyph (font stack) URL, and sprite asset URL
 *       required to render the map base layers. Requires authentication. Operators can
 *       override defaults using environment variables.
 *     tags:
 *       - Maps
 *     responses:
 *       200:
 *         description: Map style sources successfully retrieved
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 tileUrl:
 *                   type: string
 *                   example: "https://tiles.example.com/{z}/{x}/{y}.mvt"
 *                 glyphUrl:
 *                   type: string
 *                   example: "https://protomaps.github.io/basemaps-assets/fonts/{fontstack}/{range}.pbf"
 *                 spriteUrl:
 *                   type: string
 *                   example: "https://protomaps.github.io/basemaps-assets/sprites/v4/dark"
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

        const glyphUrl = env.GLYPH_URL
            ?? "https://protomaps.github.io/basemaps-assets/fonts/{fontstack}/{range}.pbf";

        const spriteUrl = "https://protomaps.github.io/basemaps-assets/sprites/v4/dark";

        return json({ tileUrl, glyphUrl, spriteUrl });
    } catch (e) {
        return handleError(e);
    }
}