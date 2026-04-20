import { ListUpdateSchema } from "$lib/models/api/list_schema";
import type { List } from "$lib/models/list";
import { Collection, handleError, remove, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/list/{id}:
 *   get:
 *     summary: Get list
 *     description: Retrieves a list by ID
 *     tags:
 *       - Lists
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
 *         description: List
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 *   post:
 *     summary: Update list
 *     description: Updates a list by ID
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
 *         application/json:
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
 *   delete:
 *     summary: Delete list
 *     description: Deletes a list by ID
 *     tags:
 *       - Lists
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Success
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const { url, params } = event;

    try {
        let list: List = await event.locals.pb.send(`/remote/list/${params.id}?` + url.searchParams, {
            method: "GET",
            fetch: event.fetch,
        })

        return json(list)
    } catch (e: any) {
        return handleError(e);
    }
}

export async function POST(event: RequestEvent) {
    try {
        const r = await update<List>(event, ListUpdateSchema, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

