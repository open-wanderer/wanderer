import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail-merge/suggest:
 *   post:
 *     summary: Suggest merge target or duplicate groups
 *     description: Returns merge target suggestions for manual selection or auto-discovery, or temporary duplicate groups for maintenance workflows.
 *     tags:
 *       - Trail Merge
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailMergeSuggestRequest'
 *     responses:
 *       200:
 *         description: Suggestion response
 *         content:
 *           application/json:
 *             schema:
 *               oneOf:
 *                 - $ref: '#/components/schemas/TrailMergeSuggestResponse'
 *                 - $ref: '#/components/schemas/TrailMergeSuggestGroupsResponse'
 *       400:
 *         description: Bad Request
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 *       401:
 *         description: Unauthorized
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const response = await event.locals.pb.send("/trail-merge/suggest", {
            method: "POST",
            body,
            fetch: event.fetch,
        });

        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}
