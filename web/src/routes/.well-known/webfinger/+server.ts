import { error, type RequestEvent } from '@sveltejs/kit';
import type PocketBase from "pocketbase";
// @ts-ignore
import { ORIGIN } from "$app/env/private"
// @ts-ignore
import { handleError } from "#lib/util/api_util.ts";
// @ts-ignore
import { splitUsername } from "#lib/util/activitypub_util.ts";
// @ts-ignore
import type { WebfingerResponse } from '#lib/models/activitypub/webfinger_response.ts';

export async function GET(event: RequestEvent) {

    try {
        const resource = event.url.searchParams.get("resource")

        if (!resource) {
            return error(400, "Bad request");
        }

        let iri: string;
        let profile: string;
        let handle: string;
        const hostname = new URL(ORIGIN).hostname;
        if (resource.startsWith("acct:")) {
            const [username, domain] = splitUsername(resource.replace("acct:", ""))

            if (hostname !== domain) {
                return Response.json({ message: "Not found" }, { status: 404 });
            }
            iri = `${ORIGIN}/api/v1/activitypub/user/${username.toLowerCase()}`
            profile = `${ORIGIN}/profile/@${username.toLowerCase()}`
            handle = resource;

        } else if (new RegExp(`${ORIGIN}\/profile\/@.*`, "g").test(resource)) {
            const username = resource.split('/').pop()!.replace("@", "")
            iri = `${ORIGIN}/api/v1/activitypub/user/${username.toLowerCase()}`
            profile = `${ORIGIN}/profile/@${username.toLowerCase()}`
            handle = `@${username}@${hostname}`
        } else {
            return error(400, "Bad request");
        }

        const locals = event.locals as typeof event.locals & { pb: PocketBase };
        const actor = await locals.pb.collection("activitypub_actors").getFirstListItem(`iri='${iri}'`)

        const r: WebfingerResponse = {
            subject: handle,
            aliases: [handle, profile],
            links: [
                {
                    rel: 'self',
                    type: 'application/activity+json',
                    href: iri,
                },
                {
                    rel: 'http://webfinger.net/rel/profile-page',
                    type: 'text/html',
                    href: profile,
                },
            ],
        }

        return Response.json(r)
    } catch (err) {
        return handleError(err)
    }
}
