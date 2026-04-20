import { error, json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/search/{index}:
 *   post:
 *     summary: Search Meilisearch index
 *     description: Performs a search on a specific Meilisearch index
 *     tags:
 *       - Search
 *     parameters:
 *       - in: path
 *         name: index
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - q
 *             properties:
 *               q:
 *                 type: string
 *               options:
 *                 type: object
 *     responses:
 *       200:
 *         description: Meilisearch search results
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    const data = await event.request.json()

    try {
        const r = await event.locals.ms.index(event.params.index as string).search(data.q, data.options);
        return json(r);
    } catch (e: any) {
        console.error(e);

        throw error(e.httpStatus, e)
    }
}