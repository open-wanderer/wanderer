import { RegionIdSchema } from '$lib/models/api/region_schema';
import { proxyRegionArchive } from '$lib/server/region_archive';
import { handleError } from '$lib/util/api_util';
import { type RequestEvent } from '@sveltejs/kit';

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
 *         description: Region id (the region's materialized path)
 *       - in: header
 *         name: Range
 *         required: false
 *         schema:
 *           type: string
 *         description: Byte range to resume a download, e.g. `bytes=1048576-`
 *       - in: header
 *         name: If-Range
 *         required: false
 *         schema:
 *           type: string
 *         description: >
 *           ETag from the interrupted download. If the archive was rebuilt in the
 *           meantime, the whole new file is returned with 200 instead of a 206.
 *       - in: header
 *         name: If-None-Match
 *         required: false
 *         schema:
 *           type: string
 *         description: ETag of a copy the client already has
 *     responses:
 *       200:
 *         description: The whole vector .pmtiles file as a binary stream, with ETag and Last-Modified
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       206:
 *         description: The requested byte range, with Content-Range
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       304:
 *         description: The archive matches the ETag sent in If-None-Match
 *       400:
 *         description: Invalid region id
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Region archive not ready yet
 *       416:
 *         description: The requested range is outside the archive
 *       500:
 *         description: Internal Server Error
 *       504:
 *         description: The backend did not respond
 *   head:
 *     summary: Check a region's vector archive without downloading it
 *     tags:
 *       - Regions
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: The archive exists; Content-Length, ETag and Last-Modified describe it
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Region archive not ready yet
 */
export async function GET(event: RequestEvent) {
  return proxy(event, 'GET');
}

export async function HEAD(event: RequestEvent) {
  return proxy(event, 'HEAD');
}

async function proxy(event: RequestEvent, method: 'GET' | 'HEAD') {
  try {
    const { id } = RegionIdSchema.parse(event.params);
    return await proxyRegionArchive(event, `/regions/${id}/download`, `${id}-vector.pmtiles`, method);
  } catch (e) {
    return handleError(e);
  }
}
