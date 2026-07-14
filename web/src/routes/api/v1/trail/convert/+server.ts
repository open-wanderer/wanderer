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
        const data = await event.request.formData();

        const { gpxData } = await fromFile(data.get("file") as Blob);
        if (!gpxData.length) {
            throw new ClientResponseError({ status: 400, response: { message: "Empty file" } });
        }

        let trail: Trail;
        try {
            ({ trail } = await gpx2trail(
                gpxData,
                (data.get("name") as string | undefined) ?? undefined,
                false,
                event.fetch,
            ));
        } catch (e) {
            throw new ClientResponseError({ status: 400, response: { message: "Invalid file" } });
        }

        // Reverse-geocode the start point so `location` arrives prefilled on the
        // client, mirroring the upload route. Best-effort: a geocoding failure
        // must not fail the conversion.
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

        // Return the parsed trail without persisting it (no trails_create / pb write).
        // Attach the raw GPX so the client can render the route/geometry on the map —
        // there is no saved file to fetch from /files for an unsaved trail.
        trail.expand = { ...trail.expand, gpx_data: gpxData };
        return json(trail);
    } catch (e) {
        return handleError(e);
    }
}
