import { handleError } from "$lib/util/api_util";
import { fromFile } from "$lib/util/gpx_util";
import type { RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

/**
 * @swagger
 * /api/v1/trail/convert:
 *   post:
 *     summary: Transcode an uploaded file to GPX.
 *     description: >
 *       Accepts a GPX/KML/KMZ/TCX/FIT file (multipart), or a GPX string (JSON body with
 *       `gpx`/`gpxData`, or a raw text body), and returns the equivalent GPX document. It does
 *       NOT compute or persist a trail - clients compute trail metrics themselves.
 *       Breaking change: prior to this version the endpoint returned a JSON `Trail`.
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             type: object
 *             properties:
 *               file:
 *                 type: string
 *                 format: binary
 *               name:
 *                 type: string
 *     responses:
 *       200:
 *         description: The transcoded GPX document.
 *         content:
 *           application/gpx+xml:
 *             schema:
 *               type: string
 *       400:
 *         description: Bad Request - Invalid or empty file
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        let gpxData: string = "";

        const contentType = event.request.headers.get("content-type") || "";

        if (contentType.includes("multipart/form-data")) {
            // 1. Handle File Upload
            const data = await event.request.formData();
            const fileBlob = data.get("file") as Blob | null;

            if (!fileBlob) {
                throw new ClientResponseError({ status: 400, response: { message: "Missing file field" } });
            }

            // Extract the text content from the file blob
            const parsed = await fromFile(fileBlob);
            gpxData = parsed.gpxData;
            // The `name` field is accepted-and-ignored rather than rejected as a 400: the
            // endpoint no longer names anything, but rejecting it would break older app
            // builds harder than necessary for no benefit.
        } else if (contentType.includes("application/json")) {
            // 2. Handle JSON / Direct String Input
            const body = await event.request.json();
            gpxData = body.gpx || body.gpxData || "";
        } else {
            // 3. Fallback: Treat raw body text as the GPX string directly
            gpxData = await event.request.text();
        }

        // Validate we actually got something
        if (!gpxData || !gpxData.trim().length) {
            throw new ClientResponseError({ status: 400, response: { message: "Empty GPX data" } });
        }

        return new Response(gpxData, {
            headers: { "Content-Type": "application/gpx+xml" },
        });
    } catch (e) {
        return handleError(e);
    }
}
