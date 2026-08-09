import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import { z } from 'zod';

const RegionIdSchema = z.object({
  id: z
    .string()
    .regex(/^[a-z0-9][a-z0-9_.'-]*$/, 'Invalid region id')
    .refine((v) => !v.includes('..'), 'Invalid region id'),
});

/**
 * @swagger
 * /api/v1/regions/{id}/geometry:
 *   get:
 *     summary: Read a region's cached boundary geometry
 *     description: >
 *       Reads the already-cached `region_geometry` row for the given region
 *       path. This route never triggers an upstream fetch — a missing cached
 *       row is a 404, not a build trigger. The Go backend's
 *       `/regions/{id}/geometry` route is superuser-gated because it performs
 *       an outbound CoMaps fetch on cache miss; this route deliberately stays
 *       a pure cached-row read so it can be exposed to any authenticated
 *       user.
 *     tags:
 *       - Regions
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *         description: The region's materialized path (e.g. canada.alberta.south)
 *     responses:
 *       200:
 *         description: The cached region boundary
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 path:
 *                   type: string
 *                 polygon:
 *                   type: object
 *                 bbox:
 *                   type: array
 *                   items:
 *                     type: number
 *       400:
 *         description: Invalid region id
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: No cached geometry for this region
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
  try {
    const { id } = RegionIdSchema.parse(event.params);

    // Cached-row read ONLY — never issue the SvelteKit request-scoped fetch
    // helper or the PocketBase client's low-level send call here. The Go
    // /regions/{id}/geometry route (db/routes/regions_geometry_get.go) is
    // deliberately gated behind apis.RequireSuperuserAuth() because it
    // performs an outbound CoMaps fetch on cache miss (D-13 — open proxy /
    // upstream rate-limit abuse). This route must stay a pure PocketBase
    // read: no row means 404, never an upstream fetch. Do not "fix" this
    // into a proxy.
    const record = await event.locals.pb
      .collection('region_geometry')
      .getFirstListItem(event.locals.pb.filter('path = {:path}', { path: id }));

    return json({
      path: record.path,
      polygon: record.polygon,
      bbox: record.bbox,
    });
  } catch (e) {
    return handleError(e);
  }
}
