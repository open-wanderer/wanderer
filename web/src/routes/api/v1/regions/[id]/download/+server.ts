import { proxyRegionArchive } from '$lib/server/region_archive';
import { handleError } from '$lib/util/api_util';
import { type RequestEvent } from '@sveltejs/kit';
import { z } from 'zod';

const RegionIdSchema = z.object({
  id: z
    .string()
    .regex(/^[a-z0-9][a-z0-9_.'-]*$/, 'Invalid region id')
    .refine((v) => !v.includes('..'), 'Invalid region id'),
});

/**
 * @swagger
 * /api/v1/regions/{id}/download:
 *   get:
 *     summary: Download a region's pre-built vector archive
 *     description: >
 *       Streams the .pmtiles vector archive for the given region.
 *       Only call this once the region's status is "ready" — returns 404 if
 *       the archive hasn't been built yet. Proxies to the internal Go
 *       backend's /regions/{id}/download route, forwarding the
 *       caller's own auth token.
 *     tags:
 *       - Regions
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *         description: Region id
 *     responses:
 *       200:
 *         description: The vector .pmtiles file as a binary stream
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       400:
 *         description: Invalid region id
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Region archive not ready yet
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
  try {
    const { id } = RegionIdSchema.parse(event.params);
    return await proxyRegionArchive(event, `/regions/${id}/download`, `${id}-vector.pmtiles`);
  } catch (e) {
    return handleError(e);
  }
}
