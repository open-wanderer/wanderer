import type { Waypoint } from "$lib/models/waypoint";
import { Collection, handleError, upload } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

const fileFields = ["photos"] as const;

/**
 * @swagger
 * /api/v1/waypoint/{id}/file:
 *   post:
 *     summary: Upload waypoint file
 *     description: >
 *       Adds photos to a waypoint. The multipart field must be named `photos` (use `photos+` to append, `photos-` to remove by filename); a request without it is rejected with 400.
 *     tags:
 *       - Waypoints
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
 *             type: object
 *             properties:
 *               photos:
 *                 type: array
 *                 items:
 *                   type: string
 *                   format: binary
 *     responses:
 *       200:
 *         description: File uploaded, waypoint updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Waypoint'
 *       400:
 *         description: Bad Request (no `photos` field in the body)
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await upload<Waypoint>(event, Collection.waypoints, fileFields);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}