import type { Actor } from '$lib/models/activitypub/actor';
import { FollowCreateSchema } from '$lib/models/api/follow_schema';
import type { Follow } from '$lib/models/follow';
import { Collection, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/follow:
 *   get:
 *     summary: List follows
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
 *     responses:
 *       200:
 *         description: List of follows
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
        const follows = await list<Follow>(event, Collection.follows);
        return json(follows)
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
