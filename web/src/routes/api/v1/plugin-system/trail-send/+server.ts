import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/trail-send:
 *   post:
 *     summary: Send trail through plugin
 *     description: Sends an existing trail to an enabled plugin provider that supports trail transfer.
 *     tags:
 *       - Plugins
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - pluginId
 *               - trailId
 *             properties:
 *               pluginId:
 *                 type: string
 *               trailId:
 *                 type: string
 *               share:
 *                 type: string
 *                 description: Optional share token used to authorize sending a shared trail.
 *     responses:
 *       200:
 *         description: Trail sent
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 ok:
 *                   type: boolean
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       403:
 *         description: Forbidden
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const r = await event.locals.pb.send("/plugins/trail-send", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
