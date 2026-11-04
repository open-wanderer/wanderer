import { ORIGIN } from '$app/env/private';
import type { RequestEvent } from '@sveltejs/kit';
import { ClientResponseError } from 'pocketbase';

import type { Comment } from '#lib/models/comment.ts';
import { isRemoteHandle } from '#lib/util/activitypub_util.ts';
import { handleError } from '#lib/util/api_util.ts';

/**
 * @swagger
 * /api/v1/profile/{handle}/follows:
 *   get:
 *     summary: Get profile follows
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
 *         name: cursor
 *         description: The `next` value of the previous page
 *         schema:
 *           type: string
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
 *     responses:
 *       200:
 *         description: List of follows for the profile
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized (remote handles require login)
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.locals.user && isRemoteHandle(event.params.handle!, ORIGIN!)) {
            throw new ClientResponseError({ status: 401, response: { message: "Unauthorized" } });
        }
        let comments: Comment = await event.locals.pb.send(`/remote/profile/${event.params.handle}/follows?` + event.url.searchParams, {
            method: "GET",
            fetch: event.fetch,
        })
        return Response.json(comments)
    } catch (e) {
        return handleError(e)
    }
}