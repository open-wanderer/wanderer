import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/regions:
 *   get:
 *     summary: Get this instance's offline region catalog
 *     description: >
 *       Returns the full seeded group+leaf region catalog (sourced from the
 *       `regions` table, not the old admin-supplied region_config.json)
 *       merged with each leaf's build state. Every entry carries the
 *       hierarchy fields kind/parent/path/depth so a client can reconstruct
 *       the region tree; group rows are hierarchy-only and omit
 *       bbox/enabled/build-state, while leaf rows additionally carry
 *       bbox/enabled plus the build-state fields below. Proxies to the
 *       internal Go backend's /regions route (not publicly reachable on its
 *       own).
 *     tags:
 *       - Regions
 *     parameters:
 *       - in: query
 *         name: enabled
 *         schema:
 *           type: string
 *           enum: ["true"]
 *         required: false
 *         description: >
 *           When "true", the catalog is pruned to only admin-enabled leaf
 *           regions plus the ancestor groups needed to reach them. Any other
 *           value or omission returns the full catalog.
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
 *                   kind:
 *                     type: string
 *                     enum: [group, leaf]
 *                     description: >
 *                       Whether this row is a hierarchy group (no
 *                       bbox/build-state) or a downloadable leaf region.
 *                   parent:
 *                     type: string
 *                     description: >
 *                       The parent row's id (matches another entry's `id`),
 *                       or empty string for a root row.
 *                   path:
 *                     type: string
 *                     description: >
 *                       The materialized path uniquely identifying this row
 *                       in the seeded catalog; leaf rows use this value in
 *                       their download URLs.
 *                   depth:
 *                     type: integer
 *                     description: Hierarchy depth, 0 for a root row.
 *                   bbox:
 *                     type: array
 *                     items:
 *                       type: number
 *                     description: Leaf rows only.
 *                   enabled:
 *                     type: boolean
 *                     description: Leaf rows only — whether an admin has enabled this region for building.
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
    // Dumb passthrough: forward the incoming query string (e.g. `?enabled=true`)
    // to the Go backend unchanged. The backend owns any filtering semantics.
    const data = await event.locals.pb.send('/regions?' + event.url.searchParams, {
      method: 'GET',
      fetch: event.fetch,
    });

    return json(data);
  } catch (e) {
    return handleError(e);
  }
}
