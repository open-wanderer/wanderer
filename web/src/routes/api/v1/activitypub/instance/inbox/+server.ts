import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import type { APActivity } from 'activitypub-types';

/**
 * @swagger
 * /api/v1/activitypub/instance/inbox:
 *   post:
 *     summary: Receive ActivityPub activities for the instance actor
 *     description: Receives and processes incoming ActivityPub activities directed at the instance-level actor (Follow, Accept, Undo). Forwards to the Go inbox handler with X-Forwarded-Path set for HTTP signature verification.
 *     tags:
 *       - ActivityPub
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *     responses:
 *       200:
 *         description: Activity processed
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized — invalid HTTP signature
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const activity: APActivity = await event.request.json();
        if (!activity.actor) {
            return json("Bad request", { status: 400 });
        }

        // Clone original headers to ensure no loss
        const originalHeaders: Record<string, string> = {};
        event.request.headers.forEach((value, key) => {
            originalHeaders[key] = value;
        });

        // Set the forwarded path so the Go handler can reconstruct the signed inbox IRI
        // (T-02-04: set by the trusted SvelteKit layer, not accepted from the remote client).
        originalHeaders['X-Forwarded-Path'] = event.url.pathname;

        const response = await event.locals.pb.send("/activitypub/instance/inbox", {
            method: "POST",
            fetch: event.fetch,
            headers: originalHeaders,
            body: JSON.stringify(activity)
        });

        const headers = new Headers();
        headers.append("Content-Type", "application/activity+json");

        return json(response, { status: 200, headers });
    } catch (e) {
        return handleError(e);
    }
}
