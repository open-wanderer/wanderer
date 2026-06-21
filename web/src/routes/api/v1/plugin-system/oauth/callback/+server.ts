import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/oauth/callback:
 *   post:
 *     summary: Complete plugin OAuth flow
 *     description: Exchanges an OAuth authorization code and stores the resulting plugin credentials.
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
 *               - code
 *               - state
 *             properties:
 *               instanceId:
 *                 type: string
 *               code:
 *                 type: string
 *               state:
 *                 type: string
 *     responses:
 *       200:
 *         description: OAuth callback handled
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
        const r = await event.locals.pb.send("/plugins/oauth/callback", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e) {
        return handleError(e);
    }
}
