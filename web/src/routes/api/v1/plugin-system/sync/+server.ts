import { handleError } from "#lib/util/api_util.ts";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/sync:
 *   post:
 *     summary: Sync a plugin instance now
 *     description: Starts a sync of one of the user's plugin instances in the background, instead of waiting for the scheduled sync. An administrator sets how often a user may do this and how many syncs may run at once.
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
 *     responses:
 *       202:
 *         description: Sync started
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 started:
 *                   type: boolean
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       403:
 *         description: Manual sync is turned off on this instance
 *       404:
 *         description: Plugin instance not found
 *       409:
 *         description: The plugin instance is already syncing
 *       429:
 *         description: Too soon after the last sync, or too many syncs running. data.retryAfter holds the seconds to wait.
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const r = await event.locals.pb.send("/plugins/sync", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r, { status: 202 });
    } catch (e) {
        return handleError(e);
    }
}
