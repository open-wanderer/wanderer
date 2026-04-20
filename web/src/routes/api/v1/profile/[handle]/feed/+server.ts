import { RecordListOptionsSchema } from '$lib/models/api/base_schema';
import { type FeedItem } from '$lib/models/feed';
import type { Trail } from '$lib/models/trail';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { Collection, handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { ClientResponseError, type ListResult } from 'pocketbase';

/**
 * @swagger
 * /api/v1/profile/{handle}/feed:
 *   get:
 *     summary: Get user activity feed
 *     description: Retrieves activity feed for a user, with federation support
 *     tags:
 *       - Profiles
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
 *         description: FeedItem list
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, { message: "Bad request" })
    }

    try {
        const { actor } = await getActorResponseForHandle(event, handle);

        const searchParams = Object.fromEntries(event.url.searchParams);
        const safeSearchParams = RecordListOptionsSchema.parse(searchParams);

        let feed: ListResult<FeedItem>;
        if (actor.isLocal) {
            feed = await event.locals.pb.collection(Collection.profile_feed)
                .getList<FeedItem>(safeSearchParams.page, safeSearchParams.perPage, { ...safeSearchParams, filter: `actor='${actor.id}'` })
        } else {
            const origin = new URL(actor.iri).origin
            const feedURL = `${origin}/api/v1/profile/${actor.preferred_username}/feed?` + event.url.searchParams

            const response = await event.fetch(feedURL, { method: 'GET' })
            if (!response.ok) {
                const errorResponse = await response.json()
                throw new ClientResponseError({ status: response.status, response: errorResponse });
            }
            feed = await response.json()

            feed.items.forEach(f => {
                if (f.type == "trail") {
                    const trail = f.expand.item as Trail
                    trail.photos = trail.photos.map(p =>
                        `${origin}/api/v1/files/trails/${f.item}/${p}`
                    )
                }

            })
        }


        return json(feed)
    } catch (e) {
        console.error(e)
        return handleError(e)
    }
}
