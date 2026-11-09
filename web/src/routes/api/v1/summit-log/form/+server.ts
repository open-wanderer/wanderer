import type { SummitLog } from '#lib/models/summit_log.ts';
import { Collection, handleError, uploadCreate } from '#lib/util/api_util.ts';
import type { RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/summit-log/form:
 *   put:
 *     summary: Create summit log with file upload
 *     description: Creates a new summit log with file upload (photos/GPX) and date normalization
 *     tags:
 *       - Summit Logs
 *     parameters:
 *       - in: query
 *         name: federationBatch
 *         description: Optional token holding the new entry announcement until photos are attached; renew and complete through the federation save API.
 *         schema:
 *           type: string
 *           minLength: 16
 *           maxLength: 128
 *           pattern: '^[a-zA-Z0-9_-]+$'
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
        const batchId = event.url.searchParams.get("federationBatch");
        if (batchId !== null && !/^[a-zA-Z0-9_-]{16,128}$/.test(batchId)) {
            return Response.json({ message: "invalid federation save batch" }, { status: 400 });
        }
        const r = await uploadCreate<SummitLog>(event, Collection.summit_logs, batchId ? { federationBatch: batchId } : undefined)
        r.date = r.date?.substring(0, 10) ?? "";

        return Response.json(r);
    } catch (e) {
        return handleError(e)
    }
}
