
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';


/**
 * @swagger
 * /api/v1/activitypub/trail/{id}:
 *   get:
 *     summary: Get ActivityPub trail
 *     description: Retrieves an ActivityPub Trail object by ID (proxied from backend)
 *     tags:
 *       - ActivityPub
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: ActivityPub Trail object
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const id = event.params.id;

    try {
        const trail = await event.locals.pb.send("/activitypub/trail/" + id, {
            method: "GET",
            fetch: event.fetch,
        })

        const headers = new Headers()
        headers.append("Content-Type", "application/activity+json")

        return json({
            "@context": [
                "https://www.w3.org/ns/activitystreams",
            ],
            ...trail
        }, { status: 200, headers });
    } catch (e) {
        return handleError(e)
    }

}