import { env } from '$env/dynamic/private';

import type { Actor } from '$lib/models/activitypub/actor';
import type { UserAnonymous } from "$lib/models/user";
import { splitUsername } from '$lib/util/activitypub_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { type APActor, type APRoot } from 'activitypub-types';


export async function GET(event: RequestEvent) {

    try {
        let fullUsername = event.params.handle;
        if (!fullUsername) {
            return json({ message: "Bad request" }, { status: 400 });
        }

        fullUsername = fullUsername.replace(/^@/, "");

        const [username, domain] = splitUsername(fullUsername, env.ORIGIN)

        const actor: Actor = await event.locals.pb.collection("activitypub_actors").getFirstListItem(`preferred_username:lower='${username?.toLowerCase()}'&&isLocal=1`)
        const user: UserAnonymous = await event.locals.pb.collection("users_anonymous").getOne(actor.user!)


        const id = `${env.ORIGIN}/api/v1/activitypub/user/${username}`

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
            url: `${env.ORIGIN}/profile/@${username}`,
            published: new Date(user.created ?? "").toISOString(),
            icon: {
                type: "Image",
                url: user.avatar ? `${env.ORIGIN}/api/v1/files/users/${user.id}/${user.avatar}` : undefined
            },
            publicKey: {
                id: id + '#main-key',
                owner: id,
                publicKeyPem: actor.public_key
            }

        }
        const headers = new Headers()
        headers.append("Content-Type", "application/activity+json")

        return json(r, { headers });
    } catch (e) {
        return handleError(e)
    }


}

