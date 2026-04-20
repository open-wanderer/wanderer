import { ListShareUpdateSchema } from "$lib/models/api/list_share_schema";
import type { ListShare } from "$lib/models/list_share";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/list-share/{id}:
 *   get:
 *     summary: Get list share
 *     description: Retrieves a list share by ID
 *     tags:
 *       - List Shares
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: ListShare
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<ListShare>(event, Collection.list_share)
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/list-share/{id}:
 *   post:
 *     summary: Update list share
 *     tags:
 *       - List Shares
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/ListShareUpdateInput'
 *     responses:
 *       200:
 *         description: List share updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListShare'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<ListShare>(event, ListShareUpdateSchema, Collection.list_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/list-share/{id}:
 *   delete:
 *     summary: Delete list share
 *     tags:
 *       - List Shares
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: List share deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.list_share)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
