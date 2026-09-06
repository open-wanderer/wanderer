import { env as publicEnv } from '$env/dynamic/public';
import { env as privateEnv } from '$env/dynamic/private';

import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import type { APActivity } from 'activitypub-types';

/**
 * @swagger
 * /api/v1/activitypub/user/{handle}/inbox:
 *   post:
 *     summary: Receive ActivityPub activities
 *     description: Receives and processes incoming ActivityPub activities (Create, Update, Delete, Follow, etc.)
 *     tags:
 *       - ActivityPub
 *     parameters:
 *       - in: path
 *         name: handle
 *         required: true
 *         schema:
 *           type: string
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
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {


    try {
        const bodyText = await event.request.text()
        const activity: APActivity = JSON.parse(bodyText)
        if (!activity.actor) {
            return json("Bad request", { status: 400 });
        }

        // Clone original headers to ensure no loss
        const originalHeaders: Record<string, string> = {};
        event.request.headers.forEach((value, key) => {
            originalHeaders[key] = value
        });

        // Add forwarded path. Set after the loop so a client cannot smuggle its own value.
        originalHeaders['X-Forwarded-Path'] = event.url.pathname;

        // Authenticate this internal hop to the backend so the header above is only
        // ever trusted when it originates from this frontend proxy.
        originalHeaders['X-Internal-Secret'] = privateEnv.POCKETBASE_PROXY_SECRET ?? '';

        const success = await event.locals.pb.send("/activitypub/activity/process", {
            method: "POST",
            fetch: event.fetch,
            headers: originalHeaders,
            body: bodyText
        })

        if (success === false) {
            return json("Invalid header signature", { status: 400 });
        }

        const headers = new Headers()
        headers.append("Content-Type", "application/activity+json")

        return json("", { status: 200, headers });
    } catch (e) {
        return handleError(e)
    }


}

