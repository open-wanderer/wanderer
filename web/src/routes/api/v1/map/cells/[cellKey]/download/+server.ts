import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

const CELL_KEY_RE = /^-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+$/;

/**
 * @swagger
 * /api/v1/maps/cells/{cellKey}/download:
 *   get:
 *     summary: Download a generated grid cell
 *     description: >
 *       Streams the .pmtiles file for the given grid cell.
 *       Only call this once the cell status is "ready" — returns 404 if generation
 *       has not completed yet. The client should save this file to local storage
 *       and reuse it for all trails that overlap this grid cell.
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
 *         description: The .pmtiles file as a binary stream
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       400:
 *         description: Invalid cell key format
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Cell not ready yet or does not exist
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const cellKey = event.params.cellKey;

    if (!cellKey || !CELL_KEY_RE.test(cellKey)) {
        return json({ message: "Invalid cell key format" }, { status: 400 });
    }

    try {
        // Fetch the binary file from PocketBase and stream it through
        const response = await event.fetch(
            `${event.locals.pb.baseURL}/map/cells/${cellKey}/download`,
            {
                headers: {
                    Authorization: event.locals.pb.authStore.token
                        ? `Bearer ${event.locals.pb.authStore.token}`
                        : "",
                },
            }
        );

        if (!response.ok) {
            return new Response(response.body, { status: response.status });
        }

        return new Response(response.body, {
            status: 200,
            headers: {
                "Content-Type": "application/octet-stream",
                "Content-Disposition": `attachment; filename="${cellKey}.pmtiles"`,
                // Forward Content-Length if present so the client can show download progress
                ...(response.headers.get("Content-Length")
                    ? { "Content-Length": response.headers.get("Content-Length")! }
                    : {}),
            },
        });
    } catch (e) {
        return handleError(e);
    }
}
