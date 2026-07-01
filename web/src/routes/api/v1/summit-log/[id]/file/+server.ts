import type { SummitLog } from "$lib/models/summit_log";
import { Collection, upload } from "$lib/util/api_util";
import { error, json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/summit-log/{id}/file:
 *   post:
 *     summary: Upload summit log file
 *     description: Uploads a file (photo or GPX) for a summit log
 *     tags:
 *       - Summit Logs
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
 *         description: File uploaded, summit log updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/SummitLog'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await upload<SummitLog>(event, Collection.summit_logs);
        return json(r);
    } catch (e: any) {
        throw error(e.status, e)
    }
}