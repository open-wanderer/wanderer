import type { Trail } from '$lib/models/trail';
import { Collection, handleError, uploadCreate } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail/form:
 *   put:
 *     summary: Create trail with file upload
 *     description: Creates a new trail with file upload (GPX/photos) and date normalization
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             $ref: '#/components/schemas/TrailCreateInput'
 *     responses:
 *       201:
 *         description: Trail created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {        
        const r = await uploadCreate<Trail>(event, Collection.trails)
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