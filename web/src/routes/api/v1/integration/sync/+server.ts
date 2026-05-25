import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/integration/sync:
 *   post:
 *     summary: Trigger a manual integration sync
 *     description: >
 *       Proxies to the backend to start a manual run of all configured
 *       integrations (Strava, Komoot, Hammerhead). Returns immediately; the
 *       sync continues in the background. Responds with 409 if a sync (cron or
 *       manual) is already in progress.
 *     tags:
 *       - Integrations
 *     responses:
 *       202:
 *         description: Sync started
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       409:
 *         description: A sync is already in progress
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await event.locals.pb.send("/integration/sync", {
            method: "POST",
            fetch: event.fetch,
        });
        return json(r, { status: 202 });
    } catch (e: any) {
        return handleError(e)
    }
}
