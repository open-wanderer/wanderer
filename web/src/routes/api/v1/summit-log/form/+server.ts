import type { SummitLog } from '$lib/models/summit_log';
import { Collection, handleError, uploadCreate } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/summit-log/form:
 *   put:
 *     summary: Create summit log with file upload
 *     description: Creates a new summit log with file upload (photos/GPX) and date normalization
 *     tags:
 *       - Summit Logs
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             $ref: '#/components/schemas/SummitLogInput'
 *     responses:
 *       201:
 *         description: Summit log created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/SummitLog'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await uploadCreate<SummitLog>(event, Collection.summit_logs)
        r.date = r.date?.substring(0, 10) ?? "";

        return json(r);
    } catch (e) {
        return handleError(e)
    }
}
