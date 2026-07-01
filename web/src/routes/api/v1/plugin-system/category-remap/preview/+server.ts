import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/category-remap/preview:
 *   post:
 *     summary: Preview plugin category remap
 *     description: Counts imported trails whose stored provider category can be mapped by the current plugin instance category mapping.
 *     tags:
 *       - Plugins
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - instanceId
 *             properties:
 *               instanceId:
 *                 type: string
 *                 description: Plugin instance record ID
 *     responses:
 *       200:
 *         description: Remap preview
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const r = await event.locals.pb.send("/plugins/category-remap/preview", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
