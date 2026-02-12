import { RecordListOptionsSchema } from '$lib/models/api/base_schema';
import type { SummitLog } from '$lib/models/summit_log';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { Collection, handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { ClientResponseError, type ListResult } from 'pocketbase';

export async function GET(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, { message: "Bad request" })
    }
    
    try {
        const { actor } = await getActorResponseForHandle(event, handle);

        const searchParams = Object.fromEntries(event.url.searchParams);
        const safeSearchParams = RecordListOptionsSchema.parse(searchParams);

        if(safeSearchParams.filter?.length) {
            safeSearchParams.filter = safeSearchParams.filter + `&&author='${actor.id}'`
        }else {
            safeSearchParams.filter = `author='${actor.id}'`
        }

        let summitLogs: SummitLog[];
        if (actor.isLocal) {
            summitLogs = await event.locals.pb.collection(Collection.summit_logs)
                .getFullList<SummitLog>(safeSearchParams.page, { ...safeSearchParams })
        } else {
            const origin = new URL(actor.iri).origin
            const summitLogURL = `${origin}/api/v1/profile/${actor.preferred_username}/stats?` + event.url.searchParams
            const response = await event.fetch(summitLogURL, { method: 'GET' })
            if (!response.ok) {
                const errorResponse = await response.json()
                throw new ClientResponseError({ status: response.status, response: errorResponse });
            }
            summitLogs = await response.json()

            summitLogs.forEach(i => {
                i.photos = i.photos.map(p =>
                    `${origin}/api/v1/files/summit_logs/${i.id}/${p}`
                )
                if (i.gpx) {
                    i.gpx =  `${origin}/api/v1/files/summit_logs/${i.id}/${i.gpx}`
                }

            })
        }


        return json(summitLogs)
    } catch (e) {
        return handleError(e)
    }
}
