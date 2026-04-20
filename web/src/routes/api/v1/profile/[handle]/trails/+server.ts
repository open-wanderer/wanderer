import type { TrailSearchResult } from '$lib/models/trail';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import type { SearchResponse } from 'meilisearch';
import { ClientResponseError } from 'pocketbase';

/**
 * @swagger
 * /api/v1/profile/{handle}/trails:
 *   post:
 *     summary: Search user trails
 *     description: Searches a user's trails via Meilisearch, with federation support
 *     tags:
 *       - Profiles
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
 *             required:
 *               - q
 *             properties:
 *               q:
 *                 type: string
 *               options:
 *                 type: object
 *     responses:
 *       200:
 *         description: Meilisearch response with trail results
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, { message: "Bad request" })
    }

    try {
        const { actor } = await getActorResponseForHandle(event, handle);

        const data = await event.request.json()

        let r: SearchResponse<TrailSearchResult>;
        if (actor.isLocal) {
            r = await event.locals.ms.index("trails").search(data.q, { ...data.options, filter: `author = ${actor.id}` });
        } else {
            const origin = new URL(actor.iri).origin
            const url = `${origin}/api/v1/profile/${actor.preferred_username}/trails?` + event.url.searchParams
            const response = await event.fetch(url, { method: 'POST', body: JSON.stringify(data) })

            if (!response.ok) {
                const errorResponse = await response.json()
                throw new ClientResponseError({ status: response.status, response: errorResponse });
            }
            r = await response.json()

            r.hits.forEach(h => {
                h.thumbnail = `${origin}/api/v1/files/trails/${h.id}/${h.thumbnail}`;
                h.domain = actor.domain
                if(h.iri == '') {
                    h.iri = `${origin}/api/v1/trails/${h.id}`
                }
            })
        }


        return json(r)
    } catch (e) {
        return handleError(e)
    }
}
