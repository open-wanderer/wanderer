import { ORIGIN } from '$app/env/private';

import type { Actor } from '#lib/models/activitypub/actor.ts';
import type { UserAnonymous } from "#lib/models/user.ts";
import { splitUsername } from '#lib/util/activitypub_util.ts';
import { handleError } from '#lib/util/api_util.ts';
import { error, type RequestEvent } from '@sveltejs/kit';
import { type APActor, type APRoot } from 'activitypub-types';


/**
 * @swagger
 * /api/v1/activitypub/user/{handle}:
 *   get:
 *     summary: Get ActivityPub actor profile
 *     description: Retrieves an ActivityPub Person object for a user with public key
 *     tags:
 *       - ActivityPub
 *     parameters:
 *       - in: path
 *         name: handle
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: ActivityPub Person object with publicKey
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
        let fullUsername = event.params.handle;
        if (!fullUsername) {
            return Response.json({ message: "Bad request" }, { status: 400 });
        }

        fullUsername = fullUsername.replace(/^@/, "");

        const [username, domain] = splitUsername(fullUsername, ORIGIN)

        const actor: Actor = await event.locals.pb.collection("activitypub_actors").getFirstListItem(`preferred_username:lower='${username?.toLowerCase()}'&&is_local=1`)
        const user: UserAnonymous = await event.locals.pb.collection("users_anonymous").getOne(actor.user!)


        const id = `${ORIGIN}/api/v1/activitypub/user/${username}`

        const r: APRoot<APActor & { publicKey: { id: string, owner: string, publicKeyPem: string } }> = {
            "@context": [
                "https://www.w3.org/ns/activitystreams",
                "https://w3id.org/security/v1",
            ],
            id: id,
            type: "Person",
            inbox: id + '/inbox',
            outbox: id + '/outbox',
            summary: user.bio,
            name: actor.username,
            preferredUsername: actor.preferred_username,
            followers: id + '/followers',
            following: id + '/following',
            url: `${ORIGIN}/profile/@${username}`,
            published: new Date(user.created ?? "").toISOString(),
            icon: {
                type: "Image",
                url: user.avatar ? `${ORIGIN}/api/v1/files/users/${user.id}/${user.avatar}` : undefined
            },
            publicKey: {
                id: id + '#main-key',
                owner: id,
                publicKeyPem: actor.public_key
            }

        }
        const headers = new Headers()
        headers.append("Content-Type", "application/activity+json")

        return Response.json(r, { headers });
    } catch (e) {
        return handleError(e)
    }


}

