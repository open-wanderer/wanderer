import type { Actor } from '$lib/models/activitypub/actor';
import { splitUsername } from '$lib/util/activitypub_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { ClientResponseError, type ListResult } from "pocketbase"

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
        const q = event.url.searchParams.get("q")

        const [user, domain] = splitUsername(q!)

        let filter = `username~'${user}'`;

        if (event.url.searchParams.get("includeSelf") == "false" && event.locals.pb.authStore.record) {
            filter += `&& id != "${event.locals.pb.authStore.record.actor}"`
        }

        const response = await event.locals.pb.collection("activitypub_actors").getList(1, 3, { filter: filter })

        try {
            const { actor, error } = await event.locals.pb.send(`/activitypub/actor?resource=acct:${q}&follows=false`, { method: "GET", fetch: event.fetch, });

            if (!response.items.find(i => i.iri == actor.iri)) {
                response.items.push(actor)
            }

        } catch (e) {

        }

        return json({ items: response.items })


    } catch (e) {
        if (e instanceof Error && e.message == "fetch failed") {
            return error(404, "Not found")
        }
        return handleError(e)
    }
}
