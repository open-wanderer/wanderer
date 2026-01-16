import { env } from '$env/dynamic/private';
import type { Profile } from '$lib/models/profile';
import { splitUsername } from '$lib/util/activitypub_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, { message: "Bad request" })
    }

    if(splitUsername(handle)[1] !== undefined && !event.locals.user) {
        return error(401, { message: "Unauthorized" })
    }

    try {
        const { actor, error } = await event.locals.pb.send(`/activitypub/actor?resource=acct:${handle}&follows=true`, { method: "GET", fetch: event.fetch, });

        const profile: Profile = {
            id: actor.id!,
            username: actor.username,
            preferredUsername: actor.preferred_username,
            acct: handle,
            createdAt: actor.published ?? "",
            bio: actor.summary ?? "",
            uri: actor.iri,
            followers: actor.followerCount ?? 0,
            following: actor.followingCount ?? 0,
            icon: actor.icon ?? "",
            error
        }

        return json({ profile, actor: actor })
    } catch (e) {
        if (e instanceof Error && e.message == "fetch failed") {
            return error(404, "Not found")
        }
        return handleError(e)
    }
}
