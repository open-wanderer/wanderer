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
 *     description: Searches for ActivityPub actors by username, combining local and federated results
 *     tags:
 *       - Search
 *     parameters:
 *       - in: query
 *         name: q
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: includeSelf
 *         schema:
 *           type: boolean
 *     responses:
 *       200:
 *         description: Array of matching actors
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 type: object
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {

        if (!event.url.searchParams.has("q")) {
            throw new ClientResponseError({ status: 400, response: "Bad request" });

        }
        const q = event.url.searchParams.get("q")!
        const limit = event.url.searchParams.get("limit")

        if (isValidPubHandle(q)) {
            try {
                const { actor } = await getActorResponseForHandle(event, q!);

                const actorSearchResult = <ActorSearchResult>{
                    id: actor.id,
                    domain: actor.domain,
                    is_local: actor.isLocal,
                    preferred_username: actor.preferred_username,
                    username: actor.username,
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
