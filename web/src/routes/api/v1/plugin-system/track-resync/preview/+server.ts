import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-system/track-resync/preview:
 *   post:
 *     summary: Preview a trail track resync
 *     description: Reports whether the track of an imported trail can be fetched again from its plugin.
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
 *             properties:
 *               trailId:
 *                 type: string
 *                 description: Trail record ID
 *     responses:
 *       200:
 *         description: Whether the trail can be resynced
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 available:
 *                   type: boolean
 *                 reason:
 *                   type: string
 *                   description: Why not, when available is false (trail_not_found, not_owner, not_imported, ambiguous, instance_disabled, plugin_unavailable, no_detail_capability)
 *                 provider:
 *                   type: string
 *                 externalId:
 *                   type: string
 *                 kindRequired:
 *                   type: boolean
 *                   description: The reference does not record whether the item is a planned route or a completed activity; the resync request must state kind
 *                 suggestedKind:
 *                   type: string
 *                   enum: [planned, completed]
 *                 retryAfterSeconds:
 *                   type: integer
 *                   description: Seconds the provider asked to wait after an earlier failure
 *                 originUnverified:
 *                   type: boolean
 *                   description: The reference predates merge tracking; a merge could have moved it onto this trail
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
        const r = await event.locals.pb.send("/plugins/track-resync/preview", {
            method: "POST",
            body,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
