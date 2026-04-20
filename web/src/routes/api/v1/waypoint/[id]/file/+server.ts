import type { Waypoint } from "$lib/models/waypoint";
import { Collection, handleError, upload } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/waypoint/{id}/file:
 *   post:
 *     summary: Upload waypoint file
 *     description: Uploads a file (photo) for a waypoint
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
 *               file:
 *                 type: string
 *                 format: binary
 *     responses:
 *       200:
 *         description: File uploaded, waypoint updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Waypoint'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await upload<Waypoint>(event, Collection.waypoints);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}