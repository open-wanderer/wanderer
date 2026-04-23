import { TagUpdateSchema } from "$lib/models/api/tag_schema";
import type { Tag } from "$lib/models/tag";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/tag/{id}:
 *   get:
 *     summary: Get tag
 *     tags:
 *       - Tags
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
 *         description: Tag details
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Tag'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<Tag>(event, Collection.tags)
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/tag/{id}:
 *   post:
 *     summary: Update tag
 *     tags:
 *       - Tags
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
 *             $ref: '#/components/schemas/TagUpdateInput'
 *     responses:
 *       200:
 *         description: Tag updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Tag'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<Tag>(event, TagUpdateSchema, Collection.tags)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/tag/{id}:
 *   delete:
 *     summary: Delete tag
 *     tags:
 *       - Tags
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Tag deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.tags)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
