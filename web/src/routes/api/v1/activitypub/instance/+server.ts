import { env } from '$env/dynamic/private';

import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent) {
    try {
        const iri = `${env.ORIGIN}/api/v1/activitypub/instance`;

        const actor = await event.locals.pb
            .collection("activitypub_actors")
            .getFirstListItem(`preferred_username='instance'&&actor_type='instance'&&is_local=true`);

        const r = {
            "@context": [
                "https://www.w3.org/ns/activitystreams",
                "https://w3id.org/security/v1",
            ],
            id: iri,
            type: "Application",
            preferredUsername: actor.preferred_username,
            name: actor.username,
            inbox: actor.inbox,
            outbox: actor.outbox,
            manuallyApprovesFollowers: true,
            publicKey: {
                id: iri + '#main-key',
                owner: iri,
                publicKeyPem: actor.public_key,
            },
        };

        const headers = new Headers();
        headers.append("Content-Type", "application/activity+json");

        return json(r, { headers });
    } catch (e) {
        return handleError(e);
    }
}
