import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/oauth/revoke:
 *   post:
 *     summary: Revoke plugin OAuth credentials
 *     description: Revokes OAuth credentials stored for a plugin instance.
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
 *               authContext:
 *                 type: string
 *     responses:
 *       200:
 *         description: OAuth credentials revoked
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
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const r = await event.locals.pb.send("/plugins/oauth/revoke", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e) {
        return handleError(e);
    }
}
