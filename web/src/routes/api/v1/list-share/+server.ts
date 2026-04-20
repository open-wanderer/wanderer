import type { Actor } from '$lib/models/activitypub/actor';
import { ListShareCreateSchema } from '$lib/models/api/list_share_schema';
import type { ListShare } from '$lib/models/list_share';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/list-share:
 *   get:
 *     summary: List list shares
 *     description: Retrieves a paginated list of list shares with optional ActivityPub actor resolution
 *     tags:
 *       - List Shares
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
 *         description: ListResult<ListShare>
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<ListShare>(event, Collection.list_share);
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/list-share:
 *   put:
 *     summary: Create list share
 *     description: Creates a new list share. Converts ActivityPub actor IRI to ID
 *     tags:
 *       - List Shares
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/ListShareInput'
 *     responses:
 *       201:
 *         description: List share created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListShare'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const data = await event.request.json();
        const safeData = ListShareCreateSchema.parse(data);

        const { actor }: { actor: Actor } = await event.locals.pb.send(`/activitypub/actor?iri=${safeData.actor}`, { method: "GET", fetch: event.fetch, });
        safeData.actor = actor.id!;

        const r = await event.locals.pb.collection(Collection.list_share).create<ListShare>(safeData)

        return json(r);
    } catch (e) {
        return handleError(e)
    }
}