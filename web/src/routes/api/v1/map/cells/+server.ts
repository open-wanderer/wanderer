import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import { z } from 'zod';

const BboxSchema = z.object({
  bbox: z.string().refine(
    (s) => {
      const parts = s.split(',');
      return parts.length === 4 && parts.every((p) => Number.isFinite(Number(p)));
    },
    { message: 'Malformed bbox: expected 4 comma-separated numbers' }
  ),
});

/**
 * @swagger
 * /api/v1/maps/cells:
 *   get:
 *     summary: List grid cells for a bounding box
 *     description: >
 *       Resolves a bounding box to the fixed grid cells that cover it.
 *       Returns the current status of each cell (new / pending / ready / error)
 *       and URLs for requesting, polling, and downloading each one.
 *       The client should call this first to determine what needs to be downloaded.
 *     tags:
 *       - Maps
 *     parameters:
 *       - in: query
 *         name: bbox
 *         required: true
 *         schema:
 *           type: string
 *           example: "6.1,51.2,6.8,51.7"
 *         description: Bounding box as "minLon,minLat,maxLon,maxLat"
 *     responses:
 *       200:
 *         description: List of grid cells covering the bounding box
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 cells:
 *                   type: array
 *                   items:
 *                     type: object
 *                     properties:
 *                       key:
 *                         type: string
 *                         example: "6.00_51.00_6.50_51.50"
 *                       status:
 *                         type: string
 *                         enum: [new, pending, ready, error]
 *                       request_url:
 *                         type: string
 *                         description: URL to trigger generation of this cell
 *                       status_url:
 *                         type: string
 *                         description: URL to poll for generation progress
 *                       download_url:
 *                         type: string
 *                         description: URL to download the .pmtiles file once ready
 *                       size_bytes:
 *                         type: integer
 *                         description: File size in bytes (only present when status is ready)
 *       400:
 *         description: Missing or malformed bbox parameter
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
  try {
    const { bbox } = BboxSchema.parse(Object.fromEntries(event.url.searchParams));

    const result = await event.locals.pb.send("/map/cells", {
      method: "GET",
      query: { bbox },
      fetch: event.fetch,
    });
 
    return json(result, { status: 200 });
  } catch (e) {
    return handleError(e);
  }
}
