import { withTrailPreferenceMeiliFilter } from '#lib/server/category_preference_filter.ts';
import type { TrailSearchResult } from '#lib/models/trail.ts';
import { getActorResponseForHandle } from '#lib/util/activitypub_server_util.ts';
import { handleError } from '#lib/util/api_util.ts';
import { DownloadError, fetchExternalJson } from '#lib/util/secure_fetch_util.ts';
import { error, type RequestEvent } from '@sveltejs/kit';
import type { SearchResponse } from 'meilisearch';
import { ClientResponseError } from 'pocketbase';

// A page of search hits, with room for what an instance adds to them.
const MAX_REMOTE_SEARCH_BYTES = 8 * 1024 * 1024;

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
 *       502:
 *         description: The instance of a federated profile could not be reached safely
 */
export async function POST(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, "Bad request")
    }

    try {
        const { actor } = await getActorResponseForHandle(event, handle);

        const data = await event.request.json()

        let r: SearchResponse<TrailSearchResult>;
        if (actor.is_local) {
            const filter = await withTrailPreferenceMeiliFilter(event, [
                data.options?.filter,
                `author = ${actor.id}`,
            ]);
            r = await event.locals.ms.index("trails").search(data.q, { ...data.options, filter });
        } else {
            const origin = new URL(actor.iri).origin
            const url = `${origin}/api/v1/profile/${actor.preferred_username}/trails?` + event.url.searchParams
            const response = await fetchExternalJson(url, { method: 'POST', body: data, maxBytes: MAX_REMOTE_SEARCH_BYTES })

            if (!response.ok) {
                throw new ClientResponseError({ status: response.status, response: response.json });
            }
            r = response.json

            r.hits.forEach(h => {
                h.thumbnail = `${origin}/api/v1/files/trails/${h.id}/${h.thumbnail}`;
                h.domain = actor.domain
                if(h.iri == '') {
                    h.iri = `${origin}/api/v1/trails/${h.id}`
                }
            })
        }


        return Response.json(r)
    } catch (e) {
        if (e instanceof DownloadError) {
            console.warn(`federated search for ${handle} failed`, e);
            return Response.json({ message: "remote_instance_unavailable" }, { status: 502 });
        }
        return handleError(e)
    }
}
