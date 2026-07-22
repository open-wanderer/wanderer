import { handleError } from '$lib/util/api_util';
import { type RequestEvent } from '@sveltejs/kit';
import { z } from 'zod';

const RegionIdSchema = z.object({
  id: z.string().regex(/^[a-z0-9][a-z0-9_-]*$/, 'Invalid region id'),
});

/**
 * @swagger
 * /api/v1/regions/{id}/download-dem:
 *   get:
 *     summary: Download a region's pre-built DEM (hillshade) archive
 *     description: >
 *       Streams the companion DEM .pmtiles archive for the given region.
 *       Only call this once the region's dem_status is "ready" — returns 404
 *       if the archive hasn't been built yet. Proxies to the internal Go
 *       backend's /regions/{id}/download-dem route, forwarding the
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
 *         description: The DEM .pmtiles file as a binary stream
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
 *         description: Region DEM archive not ready yet
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
  try {
    const { id } = RegionIdSchema.parse(event.params);
    const range = event.request.headers.get('Range');

    const response = await event.fetch(
      `${event.locals.pb.baseURL}/regions/${id}/download-dem`,
      {
        headers: {
          Authorization: event.locals.pb.authStore.token
            ? `Bearer ${event.locals.pb.authStore.token}`
            : '',
          ...(range ? { Range: range } : {}),
        },
      }
    );

    if (!response.ok) {
      return new Response(response.body, { status: response.status });
    }

    return new Response(response.body, {
      status: response.status,
      headers: {
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': `attachment; filename="${id}-dem.pmtiles"`,
        // Forward Content-Length if present so the client can show download progress
        ...(response.headers.get('Content-Length')
          ? { 'Content-Length': response.headers.get('Content-Length')! }
          : {}),
        // Forward Content-Range/Accept-Ranges so a resumed (Range) request's
        // 206 Partial Content response carries the byte-offset info the
        // client needs to append correctly
        ...(response.headers.get('Content-Range')
          ? { 'Content-Range': response.headers.get('Content-Range')! }
          : {}),
        ...(response.headers.get('Accept-Ranges')
          ? { 'Accept-Ranges': response.headers.get('Accept-Ranges')! }
          : {}),
      },
    });
  } catch (e) {
    return handleError(e);
  }
}
