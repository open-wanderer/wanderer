import { env } from '$env/dynamic/private';
import { env as publicEnv } from '$env/dynamic/public';

import type { Activity } from '$lib/models/activitypub/activity';
import type { Actor } from '$lib/models/activitypub/actor';
import { RecordListOptionsSchema } from '$lib/models/api/base_schema';
import { splitUsername } from '$lib/util/activitypub_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import type { APActivity, APOrderedCollectionPage, APRoot } from 'activitypub-types';
import type { ListResult } from 'pocketbase';


/**
 * @swagger
 * /api/v1/activitypub/user/{handle}/outbox:
 *   get:
 *     summary: Get ActivityPub outbox collection
 *     description: Retrieves an OrderedCollection paginated list of activities published by this user
 *     tags:
 *       - ActivityPub
 *     parameters:
 *       - in: path
 *         name: handle
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *     responses:
 *       200:
 *         description: ActivityPub OrderedCollectionPage
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


    try {
        const safeSearchParams = RecordListOptionsSchema.parse(Object.fromEntries(event.url.searchParams));

        const page = safeSearchParams.page ?? 1
        const perPage = safeSearchParams.perPage ?? 10

        let fullUsername = event.params.handle;
        if (!fullUsername) {
            return json({ message: "Bad request" }, { status: 400 });
        }

        const [username, domain] = splitUsername(fullUsername, env.ORIGIN)


        const actor: Actor = await event.locals.pb.collection("activitypub_actors").getFirstListItem(`preferred_username:lower='${username?.toLowerCase()}'&&isLocal=true`)

        const filter = `actor='${actor.iri}'&&type='Create'${safeSearchParams.filter ? '&&' + safeSearchParams.filter : ''}`
        const activities: ListResult<Activity> = await event.locals.pb.collection("activitypub_activities").getList(page, perPage, { sort: safeSearchParams.sort ?? "-created", filter })

        const id = actor.iri;

        const hasNextPage = page * perPage < activities.totalItems;

        const outbox: APRoot<APOrderedCollectionPage> = {
            "@context": [
                "https://www.w3.org/ns/activitystreams",
            ],
            type: "OrderedCollectionPage",
            first: id + "/outbox?page=1",
            ...(page > 1 ? { prev: `${id}/outbox?page=${page - 1}` } : {}),
            ...(hasNextPage ? { next: `${id}/outbox?page=${page + 1}` } : {}),
            partOf: id + "/outbox",
            totalItems: activities.totalItems,
            orderedItems: activities.items.map<APActivity>(a => ({
                id: a.iri,
                actor: a.actor,
                type: a.type,
                to: a.to,
                cc: a.cc,
                published: a.published,
                object: a.object
            }))
        }

        const headers = new Headers()
        headers.append("Content-Type", "application/activity+json")

        return json(outbox, { headers });
    } catch (e) {
        return handleError(e)
    }


}

