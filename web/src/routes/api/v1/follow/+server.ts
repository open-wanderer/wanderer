import type { Actor } from '$lib/models/activitypub/actor';
import { FollowCreateSchema } from '$lib/models/api/follow_schema';
import type { Follow } from '$lib/models/follow';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { APIError, Collection, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import type { APOrderedCollectionPage } from 'activitypub-types';
import { ClientResponseError, type ListResult } from "pocketbase";

/**
 * @swagger
 * /api/v1/follow:
 *   get:
 *     summary: List follows
 *     description: Retrieves follows or ActivityPub follower/following collections. Supports federated queries via handle parameter
 *     tags:
 *       - Follows
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *       - in: query
 *         name: sort
 *         schema:
 *           type: string
 *       - in: query
 *         name: filter
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *       - in: query
 *         name: handle
 *         schema:
 *           type: string
 *       - in: query
 *         name: type
 *         schema:
 *           type: string
 *           enum: [followers, following]
 *     responses:
 *       200:
 *         description: List of follows or ActivityPub collection
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.url.searchParams.has("handle")) {
            const follows = await list<Follow>(event, Collection.follows);
            return json(follows)
        } else {
            const handle = event.url.searchParams.get("handle");
            const type = event.url.searchParams.get("type");

            if (!handle || (type !== "followers" && type !== "following")) {
                throw new APIError(400, "invalid params")
            }

            const { actor } = await getActorResponseForHandle(event, handle);

            const page = event.url.searchParams.get("page") ?? "1"

            let followers: APOrderedCollectionPage;

            // fetch followers locally to not run into auth issues with private profiles
            if (actor.id === event.locals.user?.actor) {
                const r = await event.fetch(actor[type as "followers" | "following"]! + '?' + new URLSearchParams({ page }))

                if (!r.ok) {
                    const errorResponse = await r.json()
                    throw new ClientResponseError({ status: r.status, response: errorResponse });
                }
                followers = await r.json()
            } else {
                followers = await event.locals.pb.send(`/activitypub/actor/${actor.id}/${type}?page=${page}`, { method: "GET", fetch: event.fetch, });
            }

            const followerActors: Actor[] = []
            for (const f of followers.orderedItems ?? []) {
                try {
                    const { actor }: { actor: Actor } = await event.locals.pb.send(`/activitypub/actor?iri=${f}`, { method: "GET", fetch: event.fetch, });
                    followerActors.push(actor)

                } catch (e) {
                    continue
                }

            }


            const result: ListResult<Actor> = {
                items: followerActors,
                page: parseInt(page),
                perPage: 10,
                totalItems: actor.followerCount ?? 0,
                totalPages: Math.ceil((actor.followerCount ?? 0) / 10)
            }
            return json(result)
        }


    } catch (e) {
        return handleError(e)
    }
}

export async function PUT(event: RequestEvent) {
    try {
        const data = await event.request.json();
        const safeData = FollowCreateSchema.parse(data);

        const followerActor: Actor = await event.locals.pb.collection("activitypub_actors").getFirstListItem(`user = '${event.locals.user.id}'`)
        const followeeActor: Actor = await event.locals.pb.collection("activitypub_actors").getOne(safeData.followee);

        const follow = await event.locals.pb.collection("follows").create({ follower: followerActor.id, followee: followeeActor.id, status: followeeActor.isLocal ? "accepted" : "pending" })

        return json(follow);
    } catch (e) {
        return handleError(e)
    }
}
