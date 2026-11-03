import type { List } from '#lib/models/list.ts';
import { Collection, handleError, uploadCreate } from '#lib/util/api_util.ts';
import type { RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/list/form:
 *   put:
 *     summary: Create list with file upload
 *     description: Creates a new list with file upload (avatar)
 *     tags:
 *       - Lists
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             type: object
 *     responses:
 *       201:
 *         description: List
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await uploadCreate<List>(event, Collection.lists)
        return Response.json(r);
    } catch (e) {
        return handleError(e)
    }
}