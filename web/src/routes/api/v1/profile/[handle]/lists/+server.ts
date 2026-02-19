import type { TrailSearchResult } from '$lib/models/trail';
import type { ListSearchResult } from '$lib/stores/search_store';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import type { SearchResponse } from 'meilisearch';
import { ClientResponseError } from 'pocketbase';

export async function POST(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, { message: "Bad request" })
    }

    try {
        const { actor } = await getActorResponseForHandle(event, handle);

        const data = await event.request.json()

        let r: SearchResponse<ListSearchResult>;
        if (actor.isLocal) {
            r = await event.locals.ms.index("lists").search(data.q, { ...data.options, filter: `author = ${actor.id}` });
        } else {
            const origin = new URL(actor.iri).origin
            const url = `${origin}/api/v1/profile/${actor.preferred_username}/lists?` + event.url.searchParams
            const response = await event.fetch(url, { method: 'POST', body: JSON.stringify(data) })

            if (!response.ok) {
                const errorResponse = await response.json()
                throw new ClientResponseError({ status: response.status, response: errorResponse });
            }
            r = await response.json()

            r.hits.forEach(h => {
                h.avatar = h.avatar ? `${origin}/api/v1/files/lists/${h.id}/${h.avatar}` : undefined;
            })
        }


        return json(r)
    } catch (e) {
        return handleError(e)
    }
}
