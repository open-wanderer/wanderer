import type { TrailSearchResult } from '#lib/models/trail.ts';
import type { ListSearchResult } from '#lib/stores/search_store.ts';
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
 * /api/v1/profile/{handle}/lists:
 *   post:
 *     summary: Search user lists
 *     description: Searches a user's lists via Meilisearch, with federation support
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
 *         description: Meilisearch response with list results
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

        let r: SearchResponse<ListSearchResult>;
        if (actor.is_local) {
            const authorFilter = `author = ${actor.id}`;
            const clientFilter = data.options?.filter;
            const combinedFilter = clientFilter
                ? [authorFilter, clientFilter]
                : authorFilter;
            r = await event.locals.ms.index("lists").search(data.q, { ...data.options, filter: combinedFilter });
        } else {
            const origin = new URL(actor.iri).origin
            const url = `${origin}/api/v1/profile/${actor.preferred_username}/lists?` + event.url.searchParams
            const response = await fetchExternalJson(url, { method: 'POST', body: data, maxBytes: MAX_REMOTE_SEARCH_BYTES })

            if (!response.ok) {
                throw new ClientResponseError({ status: response.status, response: response.json });
            }
            r = response.json

            r.hits.forEach(h => {
                h.avatar = h.avatar ? `${origin}/api/v1/files/lists/${h.id}/${h.avatar}` : undefined;
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
