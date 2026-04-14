import type { Comment } from '$lib/models/comment';
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent) {
    try {
        let comments: Comment = await event.locals.pb.send(`/remote/profile/${event.params.handle}/follows?` + event.url.searchParams, {
            method: "GET",
            fetch: event.fetch,
        })
        return json(comments)
    } catch (e) {
        return handleError(e)
    }
}