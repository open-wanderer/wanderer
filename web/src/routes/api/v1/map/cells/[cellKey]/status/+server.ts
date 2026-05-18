import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/maps/cells/{cellKey}/status:
 *   get:
 *     summary: Poll the generation status of a grid cell
 *     description: >
 *       Lightweight endpoint to check whether a grid cell has finished generating.
 *       Poll this every few seconds after receiving a 202 from the cell request endpoint.
 *       When status becomes "ready", proceed to download the cell via the download endpoint.
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
 *         description: Current status of the cell
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 status:
 *                   type: string
 *                   enum: [pending, generating, ready, error]
 *                 size_bytes:
 *                   type: integer
 *                   description: Present only when status is "ready"
 *                 download_url:
 *                   type: string
 *                   description: Present only when status is "ready"
 *                 error:
 *                   type: string
 *                   description: Present only when status is "error"
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
    const result = await event.locals.pb.send(`/map/cells/${cellKey}/status`, {
      method: "GET",
      fetch: event.fetch,
    });
 
    return json(result, { status: 200 });
  } catch (e) {
    return handleError(e);
  }
}
