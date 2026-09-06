import type { Trail } from "$lib/models/trail";
import { Collection, handleError, uploadUpdate } from "$lib/util/api_util";
import { fromFile, gpx2trail } from "$lib/util/gpx_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

/**
 * @swagger
 * /api/v1/trail/form/{id}:
 *   post:
 *     summary: Update trail with file upload
 *     description: Updates a trail with file upload (GPX/photos) and date normalization
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             $ref: '#/components/schemas/TrailUpdateInput'
 *     responses:
 *       200:
 *         description: Trail updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const data = await event.request.formData();
        await updateGPXStats(data, event);
        const r = await uploadUpdate<Trail>(event, Collection.trails, data)
        enrichRecord(r);
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}

async function updateGPXStats(data: FormData, event: RequestEvent) {
    const file = data.get("gpx");
    if (!(file instanceof Blob)) {
        return;
    }

    const { gpxData } = await fromFile(file);
    if (!gpxData.length) {
        throw new ClientResponseError({ status: 400, response: { message: "Empty file" } });
    }

    try {
        const { trail } = await gpx2trail(gpxData, undefined, true, event.fetch);
        data.set("distance", String(trail.distance ?? 0));
        data.set("duration", String(trail.duration ?? 0));
        data.set("elevation_gain", String(trail.elevation_gain ?? 0));
        data.set("elevation_loss", String(trail.elevation_loss ?? 0));
    } catch (e: any) {
        console.error(e);
        throw new ClientResponseError({ status: 400, response: { message: "Invalid file" } });
    }
}


function enrichRecord(r: Trail) {
    r.date = r.date?.substring(0, 10) ?? "";
    for (const log of r.expand?.summit_logs_via_trail ?? []) {
        log.date = log.date.substring(0, 10);
    }
}
