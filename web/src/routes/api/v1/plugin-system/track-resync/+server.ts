import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/track-resync:
 *   post:
 *     summary: Resync a trail track
 *     description: Fetches the track of an imported trail again from the plugin it came from and replaces the stored track. Runs immediately.
 *     tags:
 *       - Plugins
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - trailId
 *               - expectedProvider
 *               - expectedExternalId
 *             properties:
 *               trailId:
 *                 type: string
 *                 description: Trail record ID
 *               expectedProvider:
 *                 type: string
 *                 description: Provider shown in the confirmed preview
 *               expectedExternalId:
 *                 type: string
 *                 description: Provider item ID shown in the confirmed preview
 *               kind:
 *                 type: string
 *                 enum: [planned, completed]
 *                 description: Whether the provider item is a planned route or a completed activity. Required when the preview reported kindRequired, ignored otherwise.
 *     responses:
 *       200:
 *         description: The track was replaced
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 trailId:
 *                   type: string
 *                 provider:
 *                   type: string
 *                 externalId:
 *                   type: string
 *                 kind:
 *                   type: string
 *                   enum: [planned, completed]
 *                 warning:
 *                   type: string
 *                   description: Set when the track was stored but a follow-up such as the search index update failed (trail_update_followup_failed)
 *                 track:
 *                   type: object
 *                   description: The trail fields the new track changed
 *                   properties:
 *                     gpx:
 *                       type: string
 *                     distance:
 *                       type: number
 *                     elevation_gain:
 *                       type: number
 *                     elevation_loss:
 *                       type: number
 *                     duration:
 *                       type: number
 *                     lat:
 *                       type: number
 *                     lon:
 *                       type: number
 *                     polyline:
 *                       type: string
 *                     min_lat:
 *                       type: number
 *                     max_lat:
 *                       type: number
 *                     min_lon:
 *                       type: number
 *                     max_lon:
 *                       type: number
 *                     bounding_box_diagonal:
 *                       type: number
 *                     updated:
 *                       type: string
 *       400:
 *         description: The trail cannot be resynced; the message carries the reason (trail_not_found, not_owner, not_imported, ambiguous, instance_disabled, plugin_unavailable, no_detail_capability, kind_required)
 *       401:
 *         description: Unauthorized
 *       403:
 *         description: The trail changed owner while the track was fetched
 *       409:
 *         description: The import reference changed since preview, or the trail track/reference changed while the track was fetched; reload and try again
 *       502:
 *         description: The plugin or provider could not deliver the track
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 message:
 *                   type: string
 *                   example: provider_rejected
 *                 detail:
 *                   type: object
 *                   properties:
 *                     data:
 *                       type: object
 *                       properties:
 *                         code:
 *                           type: string
 *                           description: Plugin error code (auth_failed, invalid_grant, rate_limited, provider_unavailable, plugin_error, ...)
 *                         message:
 *                           type: string
 *                         retryAfterSeconds:
 *                           type: integer
 *                           description: Seconds to wait before trying again, when the provider or its backoff asks for it
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const r = await event.locals.pb.send("/plugins/track-resync", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
