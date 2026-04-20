import { ListCreateSchema } from '$lib/models/api/list_schema';
import type { List } from '$lib/models/list';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/list:
 *   get:
 *     summary: List all lists
 *     tags:
 *       - Lists
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
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: List of lists
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
        const r = await list<List>(event, Collection.lists);
        for (const t of r.items) {
            if (!t.author || !event.locals.pb.authStore.record) {
                continue;
            }
          
        }
        return json(r)
    } catch (e) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/list:
 *   put:
 *     summary: Create list
 *     tags:
 *       - Lists
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/ListInput'
 *     responses:
 *       201:
 *         description: List created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/List'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<List>(event, ListCreateSchema, Collection.lists)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}