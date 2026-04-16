import type { Category } from "$lib/models/category";
import { Collection, handleError, list } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/category:
 *   get:
 *     summary: List categories
 *     description: Retrieves a paginated list of activity categories
 *     tags:
 *       - Categories
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *       - in: query
 *         name: sort
 *         schema:
 *           type: string
 *       - in: query
 *         name: filter
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: List of categories
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */

export async function GET(event: RequestEvent) {
    try {
        const r = await list<Category>(event, Collection.categories);
        return json(r)
    } catch (e) {
        return handleError(e)
    }

}