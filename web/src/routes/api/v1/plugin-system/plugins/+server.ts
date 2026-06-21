import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/plugins:
 *   get:
 *     summary: List installed plugins
 *     description: Refreshes the plugin cache and lists locally installed plugin providers with runtime status and manifest metadata.
 *     tags:
 *       - Plugins
 *     responses:
 *       200:
 *         description: Installed plugin providers
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 items:
 *                   type: array
 *                   items:
 *                     type: object
 *                     properties:
 *                       id:
 *                         type: string
 *                       type:
 *                         type: string
 *                       name:
 *                         type: string
 *                       version:
 *                         type: string
 *                       runtime:
 *                         type: string
 *                       capabilities:
 *                         type: array
 *                         items:
 *                           type: string
 *                       status:
 *                         type: string
 *                         enum: [available, disabled, error]
 *                       error:
 *                         type: string
 *                       manifest:
 *                         type: object
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await event.locals.pb.send("/plugins", {
            method: "GET",
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
