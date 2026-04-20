import { error, json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/search/multi:
 *   post:
 *     summary: Multi-index search
 *     description: Performs batch searches across multiple Meilisearch indices
 *     tags:
 *       - Search
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - queries
 *             properties:
 *               queries:
 *                 type: array
 *                 items:
 *                   type: object
 *     responses:
 *       200:
 *         description: Combined Meilisearch results
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
        const r = await event.locals.ms.multiSearch({
            queries: data.queries
        });
        return json(r);
    } catch (e: any) {
        throw error(e.httpStatus, e)
    }
}