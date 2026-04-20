import { CommentCreateSchema } from '$lib/models/api/comment_schema';
import type { Comment } from '$lib/models/comment';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import { type ListResult } from "pocketbase";

/**
 * @swagger
 * /api/v1/comment:
 *   get:
 *     summary: List comments
 *     tags:
 *       - Comments
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
 *           description: Federated query parameter
 *     responses:
 *       200:
 *         description: List of comments
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
            const comments = await list<Comment>(event, Collection.comments);
            return json(comments)
        } else {
            const { actor, error } = await event.locals.pb.send(`/activitypub/actor?resource=acct:${event.url.searchParams.get("handle")}`, { method: "GET", fetch: event.fetch, });
            event.url.searchParams.delete("handle")
            const localComments = await list<Comment>(event, Collection.comments);
            if (actor.isLocal) {
                return json(localComments)
            }

            const deduplicationMap: Record<string, Comment> = {}

            localComments.items.forEach(c => {
                if (c.iri) {
                    const id = c.iri.substring(c.iri.length - 15)
                    deduplicationMap[id] = c
                } else if (c.id) {
                    deduplicationMap[c.id] = c
                }
            })
            const origin = new URL(actor.iri).origin
            const url = `${origin}/api/v1/comment`

            const response = await event.fetch(url + '?' + event.url.searchParams, { method: 'GET' })
            if (!response.ok) {
                const errorResponse = await response.json()
                console.error(errorResponse)

            }
            const remoteComments: ListResult<Comment> = await response.json()

            remoteComments.items = remoteComments.items.filter(c => {
                const iriId = c.iri?.substring(c.iri.length - 15) ?? ""
                if (deduplicationMap[c.id!] != undefined) {
                    deduplicationMap[c.id!] = { ...c, author: deduplicationMap[c.id!].author }
                    return false
                } else if (deduplicationMap[iriId] != undefined) {
                    deduplicationMap[iriId] = { ...c, author: deduplicationMap[iriId].author }
                    return false
                }
                return true
            })

            remoteComments.items.forEach(c => {
                if (!c.iri?.length) {
                    c.iri = `${url}/${c.id}`
                }
            })

            const allCommentItems = <ListResult<Comment>>{
                items: localComments.items.concat(remoteComments.items),
                page: localComments.page,
                perPage: localComments.perPage,
                totalItems: localComments.items.length + remoteComments.items.length,
                totalPages: Math.ceil((localComments.items.length + remoteComments.items.length) / localComments.perPage)
            }

            allCommentItems.items = allCommentItems.items.sort((a, b) => {
                return new Date(b.created ?? 0).getTime() - new Date(a.created ?? 0).getTime()
            })

            return json(allCommentItems)
        }
    } catch (e) {
        return handleError(e)
    }
}

export async function PUT(event: RequestEvent) {
    try {
        const r = await create<Comment>(event, CommentCreateSchema, Collection.comments)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}