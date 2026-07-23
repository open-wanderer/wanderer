import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/regions:
 *   get:
 *     summary: Get this instance's offline region catalog
 *     description: >
 *       Returns the merged config-plus-build-state catalog for every region
 *       configured in this instance's admin-supplied region catalog file.
 *       Proxies to the internal Go backend's /regions route (not
 *       publicly reachable on its own — see the routing decision in
 *       21.5-03-PLAN.md).
 *     tags:
 *       - Regions
 *     responses:
 *       200:
 *         description: The instance's region catalog as a JSON array
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 type: object
 *                 properties:
 *                   id:
 *                     type: string
 *                   name:
 *                     type: string
 *                   bbox:
 *                     type: array
 *                     items:
 *                       type: number
 *                   status:
 *                     type: string
 *                     enum: [ready, building, error]
 *                   version:
 *                     type: string
 *                   vector_url:
 *                     type: string
 *                   vector_size:
 *                     type: integer
 *                   dem_status:
 *                     type: string
 *                   dem_url:
 *                     type: string
 *                   dem_size:
 *                     type: integer
 *                   error:
 *                     type: string
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
  try {
    const data = await event.locals.pb.send('/regions', {
      method: 'GET',
      fetch: event.fetch,
    });

    return json(data);
  } catch (e) {
    return handleError(e);
  }
}
