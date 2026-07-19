import type { Trail } from "$lib/models/trail";
import { searchLocationReverse } from "$lib/stores/search_store";
import { handleError } from "$lib/util/api_util";
import { fromFile, gpx2trail } from "$lib/util/gpx_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

/**
 * @swagger
 * /api/v1/trail/convert:
 *   post:
 *     summary: Convert an uploaded file to a trail without persisting it
 *     description: Uploads a GPX/KML/KMZ/TCX/FIT file, parses it to extract trail data, and returns the resulting trail. The trail is NOT saved to the database.
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
 *         description: Parsed (unsaved) trail
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request - Invalid or empty file
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        let gpxData: string = "";
        let customName: string | undefined = undefined;

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
            customName = (data.get("name") as string | undefined) ?? undefined;
        } else if (contentType.includes("application/json")) {
            // 2. Handle JSON / Direct String Input
            const body = await event.request.json();
            gpxData = body.gpx || body.gpxData || "";
            customName = body.name ?? undefined;
        } else {
            // 3. Fallback: Treat raw body text as the GPX string directly
            gpxData = await event.request.text();
        }

        // Validate we actually got something
        if (!gpxData || !gpxData.trim().length) {
            throw new ClientResponseError({ status: 400, response: { message: "Empty GPX data" } });
        }

        let trail: Trail;
        try {
            ({ trail } = await gpx2trail(
                gpxData,
                customName,
                false,
                event.fetch,
            ));
        } catch (e) {
            throw new ClientResponseError({ status: 400, response: { message: "Invalid GPX content" } });
        }

        // Reverse-geocode the start point
        if (trail.lat && trail.lon) {
            try {
                const location = await searchLocationReverse(
                    trail.lat,
                    trail.lon,
                    {},
                    event.fetch,
                );
                trail.location ??= location;
            } catch (e) {
                console.warn("Reverse geocoding failed during convert", e);
            }
        }

        // Attach the raw GPX to the response
        trail.expand = { ...trail.expand, gpx_data: gpxData };
        return json(trail);
    } catch (e) {
        return handleError(e);
    }
}