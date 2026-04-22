import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail-merge:
 *   post:
 *     summary: Merge trails
 *     description: Merges a source trail into a target trail using the backend merge logic.
 *     tags:
 *       - Trail Merge
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailMergeExecuteRequest'
 *     responses:
 *       200:
 *         description: Merge acknowledged
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/TrailMergeExecuteResponse'
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
 *       403:
 *         description: Forbidden
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const response = await event.locals.pb.send("/trail-merge", {
            method: "POST",
            body,
            fetch: event.fetch,
        });

        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}
