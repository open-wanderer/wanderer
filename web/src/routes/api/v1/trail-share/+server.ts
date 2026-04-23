import type { Actor } from '$lib/models/activitypub/actor';
import { TrailShareCreateSchema } from '$lib/models/api/trail_share_schema';
import type { TrailShare } from '$lib/models/trail_share';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail-share:
 *   get:
 *     summary: List trail shares
 *     description: Retrieves a paginated list of trail shares with optional ActivityPub actor resolution
 *     tags:
 *       - Trail Shares
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
 *         description: ListResult<TrailShare>
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<TrailShare>(event, Collection.trail_share);
        return json(r)
    } catch (e: any) {
        handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/trail-share:
 *   put:
 *     summary: Create trail share
 *     description: Creates a new trail share. Converts ActivityPub actor IRI to ID
 *     tags:
 *       - Trail Shares
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailShareInput'
 *     responses:
 *       201:
 *         description: Trail share created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/TrailShare'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const data = await event.request.json();
        const safeData = TrailShareCreateSchema.parse(data);

        const { actor }: { actor: Actor } = await event.locals.pb.send(`/activitypub/actor?iri=${safeData.actor}`, { method: "GET", fetch: event.fetch, });
        safeData.actor = actor.id!;

        const r = await event.locals.pb.collection(Collection.trail_share).create<TrailShare>(safeData)

        return json(r);
    } catch (e) {
        return handleError(e)
    }
}