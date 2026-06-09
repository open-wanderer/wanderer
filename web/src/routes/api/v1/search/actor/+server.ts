import type { Actor, ActorSearchResult } from '$lib/models/activitypub/actor';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { isValidPubHandle, splitUsername } from '$lib/util/activitypub_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { ClientResponseError, type ListResult } from "pocketbase"
import type { SearchResponse } from "meilisearch";

/**
 * @swagger
 * /api/v1/search/actor:
 *   get:
 *     summary: Search actors
 *     description: |
 *       Searches for ActivityPub actors by username. If the query is a valid
 *       federated handle (e.g. `user@domain.tld`), it attempts a direct
 *       ActivityPub lookup first and returns that result immediately. If the
 *       handle lookup fails, or if the query is not a handle, it falls back to
 *       a local Meilisearch index query. Returns a Meilisearch-shaped
 *       `SearchResponse` in both cases.
 *     tags:
 *       - Search
 *     parameters:
 *       - in: query
 *         name: q
 *         required: true
 *         schema:
 *           type: string
 *         description: |
 *           Search query. Can be a plain username substring or a fully-qualified
 *           ActivityPub handle (`user@domain.tld`). Handles trigger a federated
 *           lookup before falling back to local search.
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *           default: 3
 *         description: |
 *           Maximum number of results to return from the local index. Has no
 *           effect when a federated handle is resolved successfully (always
 *           returns exactly one hit).
 *       - in: query
 *         name: includeSelf
 *         schema:
 *           type: boolean
 *           default: true
 *         description: |
 *           When `false` and the request is authenticated, the authenticated
 *           user's own actor is excluded from local search results.
 *     responses:
 *       200:
 *         description: |
 *           Meilisearch-shaped response containing matching actors. The `hits`
 *           array contains actor objects with `id`, `domain`, `is_local`,
 *           `preferred_username`, `username`, and `icon` fields.
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 hits:
 *                   type: array
 *                   items:
 *                     type: object
 *                     properties:
 *                       id:
 *                         type: string
 *                       domain:
 *                         type: string
 *                       is_local:
 *                         type: boolean
 *                       preferred_username:
 *                         type: string
 *                       username:
 *                         type: string
 *                       icon:
 *                         type: string
 *                 query:
 *                   type: string
 *                 processingTimeMs:
 *                   type: integer
 *                 estimatedTotalHits:
 *                   type: integer
 *                 totalHits:
 *                   type: integer
 *                 totalPages:
 *                   type: integer
 *                 page:
 *                   type: integer
 *       400:
 *         description: Missing required `q` parameter.
 *       404:
 *         description: Federated fetch failed with a network error.
 *       500:
 *         description: Internal server error.
 */
export async function GET(event: RequestEvent) {
    if (!event.locals.user) {
        return error(401, "Unauthorized")
    }
    try {

        if (!event.url.searchParams.has("q")) {
            return error(404, "Bad request: missing required parameter 'q'")

        }
        const q = event.url.searchParams.get("q")!
        const limit = event.url.searchParams.get("limit")

        if (isValidPubHandle(q)) {
            try {
                const { actor } = await getActorResponseForHandle(event, q!);

                const actorSearchResult = <ActorSearchResult>{
                    id: actor.id,
                    domain: actor.domain,
                    is_local: actor.is_local,
                    preferred_username: actor.preferred_username,
                    username: actor.username,
                    iri: actor.iri,
                    icon: actor.icon
                };

                return json(<SearchResponse>{
                    hits: [actorSearchResult],
                    processingTimeMs: 0,
                    query: q,
                    estimatedTotalHits: 1,
                    totalHits: 1,
                    totalPages: 1,
                    page: 1,
                })
            } catch (e) {
                // Actor could not be found via the handle
                // At least search our local registry
            }
        }

        let filterText = "";

        if (event.url.searchParams.get("includeSelf") == "false" && event.locals.pb.authStore.record) {
            filterText = `id != ${event.locals.pb.authStore.record.actor}`
        }

        const r = await event.locals.ms.index("actors").search(q, { filter: filterText, limit: limit ?? 3 });


        return json(r)


    } catch (e) {
        if (e instanceof Error && e.message == "fetch failed") {
            return error(404, "Not found")
        }
        return handleError(e)
    }
}
