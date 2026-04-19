import { Collection, handleError, upload } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import type { List } from "$lib/models/list";

/**
 * @swagger
 * /api/v1/list/{id}/file:
 *   post:
 *     summary: Upload list file
 *     description: Uploads a file (cover image) for a list
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
 *             properties:
 *               file:
 *                 type: string
 *                 format: binary
 *     responses:
 *       200:
 *         description: File uploaded, list updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/List'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await upload<List>(event, Collection.lists);
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}