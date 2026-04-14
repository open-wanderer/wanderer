import type { Actor } from '$lib/models/activitypub/actor';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { splitUsername } from '$lib/util/activitypub_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { ClientResponseError, type ListResult } from "pocketbase"

export async function GET(event: RequestEvent) {
    try {

        if (!event.url.searchParams.has("q")) {
            throw new ClientResponseError({ status: 400, response: "Bad request" });

        }
        const q = event.url.searchParams.get("q")

        const [user, domain] = splitUsername(q!)

        let filter = `username~'${user}'`;

        if (event.url.searchParams.get("includeSelf") == "false" && event.locals.pb.authStore.record) {
            filter += `&& id != "${event.locals.pb.authStore.record.actor}"`
        }

        const response = await event.locals.pb.collection("activitypub_actors").getList<Actor>(1, 3, { filter: filter })

        try {
            const { actor } = await getActorResponseForHandle(event, q!);

            if (!response.items.find(i => i.iri == actor.iri)) {
                response.items.push(actor)
            }

        } catch (e) {

        }

        return json({ items: response.items })


    } catch (e) {
        if (e instanceof Error && e.message == "fetch failed") {
            return error(404, "Not found")
        }
        return handleError(e)
    }
}
