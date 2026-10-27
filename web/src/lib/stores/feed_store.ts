import type { FeedItem } from "#lib/models/feed.ts";
import { APIError } from "#lib/util/api_util.ts";
import type { ListResult } from "pocketbase";

let feed: FeedItem[] = []


export async function feed_index(page: number, perPage: number = 10, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f(`/api/v1/feed?` + new URLSearchParams({
        page: page.toString(),
        perPage: perPage.toString(),
        sort: '-created'
    }), {
        method: 'GET',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedFeed: ListResult<FeedItem> = await r.json()

    const result = page > 1 ? [...feed, ...fetchedFeed.items] : fetchedFeed.items

    feed = result;

    return { ...fetchedFeed, items: result };

}
