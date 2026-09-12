import type { Trail } from "$lib/models/trail";
import { Collection, handleError, uploadUpdate } from "$lib/util/api_util";
import { applyGpxToForm, trailGpxFields } from "$lib/util/gpx_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail/form/{id}:
 *   post:
 *     summary: Update trail with file upload
 *     description: >
 *       Updates a trail with file upload (GPX/photos) and date normalization. The record is addressed by the `id` path parameter; an `id` in the body is optional and must match it (400 `id_mismatch` otherwise).
 *       A `gpx` part replaces the track: the file is converted to GPX (FIT, KML, KMZ and TCX are accepted) and `distance`, `duration`, `elevation_gain`, `elevation_loss`, `lat` and `lon` are derived from it unless the request sets them explicitly. `bounding_box_diagonal` and `polyline` are always recomputed.
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
 *         description: Bad Request (invalid id, body `id` differs from the path, or the `gpx` file cannot be parsed)
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const data = await event.request.formData();
        await applyGpxToForm(data, trailGpxFields, true, event.fetch);
        const r = await uploadUpdate<Trail>(event, Collection.trails, data)
        enrichRecord(r);
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}


function enrichRecord(r: Trail) {
    r.date = r.date?.substring(0, 10) ?? "";
    for (const log of r.expand?.summit_logs_via_trail ?? []) {
        log.date = log.date.substring(0, 10);
    }
}