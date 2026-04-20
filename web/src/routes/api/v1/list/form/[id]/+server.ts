import type { List } from "$lib/models/list";
import { Collection, handleError, uploadUpdate } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/list/form/{id}:
 *   post:
 *     summary: Update list with file upload
 *     description: Updates a list with file upload (avatar)
 *     tags:
 *       - Lists
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
 *     responses:
 *       200:
 *         description: List
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {        
        const r = await uploadUpdate<List>(event, Collection.lists)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}
