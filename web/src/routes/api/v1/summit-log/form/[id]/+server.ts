import type { SummitLog } from "$lib/models/summit_log";
import { Collection, handleError, uploadUpdate } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/summit-log/form/{id}:
 *   post:
 *     summary: Update summit log with file upload
 *     description: >
 *       Updates a summit log with file upload (photos/GPX) and date normalization. The record is addressed by the `id` path parameter; an `id` in the body is optional and must match it (400 `id_mismatch` otherwise).
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
 *             $ref: '#/components/schemas/SummitLogUpdateInput'
 *     responses:
 *       200:
 *         description: Summit log updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/SummitLog'
 *       400:
 *         description: Bad Request (invalid id, or body `id` differs from the path)
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await uploadUpdate<SummitLog>(event, Collection.summit_logs)
        r.date = r.date?.substring(0, 10) ?? "";
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}
