import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';


/**
 * @swagger
 * /api/v1/maps/cells/{cellKey}:
 *   get:
 *     summary: Request generation of a grid cell
 *     description: >
 *       Triggers background generation of the .pmtiles file for the given grid cell
 *       if it is not already ready. Always returns immediately.
 *       - Returns 200 if the cell is already ready — the client can proceed to download.
 *       - Returns 202 if generation was started or is already in progress — the client
 *         should poll the status_url until the status becomes "ready".
 *     tags:
 *       - Maps
 *     parameters:
 *       - in: path
 *         name: cellKey
 *         required: true
 *         schema:
 *           type: string
 *           example: "6.00_51.00_6.50_51.50"
 *         description: Grid cell key in "minLon_minLat_maxLon_maxLat" format
 *     responses:
 *       200:
 *         description: Cell is already ready for download
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 status:
 *                   type: string
 *                   example: ready
 *                 download_url:
 *                   type: string
 *       202:
 *         description: Generation started or already in progress
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 status:
 *                   type: string
 *                   example: generating
 *                 status_url:
 *                   type: string
 *       400:
 *         description: Invalid cell key format
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
  const cellKey = event.params.cellKey;
 
  try {
    const result = await event.locals.pb.send(`/map/cells/${cellKey}`, {
      method: "GET",
      fetch: event.fetch,
    });
 
    const status = result.status === "ready" ? 200 : 202;
    return json(result, { status });
  } catch (e) {
    return handleError(e);
  }
}
