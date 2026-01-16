import { error, json, RequestEvent } from '@sveltejs/kit';
import type PocketBase from "PocketBase";
// @ts-ignore
import { env } from "$env/dynamic/private"
// @ts-ignore
import { env as publicEnv } from '$env/dynamic/public';
// @ts-ignore
import { handleError } from "$lib/util/api_util";
// @ts-ignore
import { splitUsername } from "$lib/util/activitypub_util";
// @ts-ignore
import type { WebfingerResponse } from '$lib/models/activitypub/webfinger_response';

export async function GET(event: RequestEvent) {

    try {
        const resource = event.url.searchParams.get("resource")

        if (!resource) {
            return error(400, "Bad request");
        }

        let iri: string;
        let profile: string;
        let handle: string;
        const hostname = new URL(env.ORIGIN).hostname;
        if (resource.startsWith("acct:")) {
            const [username, domain] = splitUsername(resource.replace("acct:", ""))

            if (hostname !== domain) {
                return json({ message: "Not found" }, { status: 404 });
            }
            iri = `${env.ORIGIN}/api/v1/activitypub/user/${username.toLowerCase()}`
            profile = `${env.ORIGIN}/profile/@${username.toLowerCase()}`
            handle = resource;

        } else if (new RegExp(`${env.ORIGIN}\/profile\/@.*`, "g").test(resource)) {
            const username = resource.split('/').pop()!.replace("@", "")
            iri = `${env.ORIGIN}/api/v1/activitypub/user/${username.toLowerCase()}`
            profile = `${env.ORIGIN}/profile/@${username.toLowerCase()}`
            handle = `@${username}@${hostname}`
        } else {
            return error(400, "Bad request");
        }

        const actor = await event.locals.pb.collection("activitypub_actors").getFirstListItem(`iri='${iri}'`)

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

        return json(r)
    } catch (err) {
        return handleError(err)
    }
}