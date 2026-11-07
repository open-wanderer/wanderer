import { ORIGIN } from '$app/env/private';

import { handleError } from '#lib/util/api_util.ts';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/activitypub/instance:
 *   get:
 *     summary: Get the instance-level ActivityPub actor
 *     description: Returns the Application-type ActivityPub actor representing this Wanderer instance. Used by peer instances to identify and federate with this server.
 *     tags:
 *       - ActivityPub
 *     responses:
 *       200:
 *         description: ActivityPub Application actor with publicKey
 *         content:
 *           application/activity+json:
 *             schema:
 *               type: object
 *       404:
 *         description: Instance actor not found (server not yet initialised)
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const iri = `${ORIGIN}/api/v1/activitypub/instance`;

        const actor = await event.locals.pb
            .collection("activitypub_actors")
            .getFirstListItem(`actor_type='instance'&&is_local=true`, { sort: "@rowid" });

        const r = {
            "@context": [
                "https://www.w3.org/ns/activitystreams",
                "https://w3id.org/security/v1",
            ],
            id: iri,
            type: "Application",
            preferredUsername: actor.preferred_username,
            name: actor.username,
            // Derived from the same ORIGIN-based id as the rest of the document, not
            // read from the row, so the document stays self-consistent even if the
            // stored row was not moved after an ORIGIN change. The Go inbox route
            // (/activitypub/instance/inbox) does not depend on the stored IRI.
            inbox: `${iri}/inbox`,
            outbox: `${iri}/outbox`,
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
