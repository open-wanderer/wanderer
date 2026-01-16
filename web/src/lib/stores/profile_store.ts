import type { FeedItem } from "$lib/models/feed";
import type { ListFilter } from "$lib/models/list";
import type { SummitLog, SummitLogFilter } from "$lib/models/summit_log";
import { defaultTrailSearchAttributes, Trail, type TrailFilter, type TrailSearchResult } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";
import type { Hits } from "meilisearch";
import type { ListResult } from "pocketbase";
import { searchResultToLists } from "./list_store";
import type { ListSearchResult } from "./search_store";
import { buildFilterText } from "./summit_log_store";
import { searchResultToTrailList } from "./trail_store";

let feed: FeedItem[] = []


export async function profile_show(handle: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f('/api/v1/profile/' + handle, {
        method: 'GET',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response = await r.json()
    return response;

}

export async function profile_lists_index(handle: string, filter: ListFilter, page: number = 1, perPage: number = 6, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f(`/api/v1/profile/${handle}/lists`, {
        method: "POST",
        body: JSON.stringify({
            q: filter.q,
            options: {
                sort: [`${filter.sort}:${filter.sortOrder == "+" ? "asc" : "desc"}`],
                hitsPerPage: perPage,
                page: page
            }
        }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const result: { page: number, totalPages: number, hits: Hits<ListSearchResult> } = await r.json();

    if (result.hits.length == 0) {
        return { items: [], ...result };
    }

    const resultLists = await searchResultToLists(result.hits)

    return { items: resultLists, ...result };

}


export async function profile_feed_index(handle: string, page: number, perPage: number = 10, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f(`/api/v1/profile/${handle}/feed?` + new URLSearchParams({
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

export async function profile_trails_index(handle: string, filter: TrailFilter, page: number = 1, perPage: number = 1, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f(`/api/v1/profile/${handle}/trails`, {
        method: "POST",
        body: JSON.stringify({
            q: filter.q,
            options: {
                attributesToRetrieve: defaultTrailSearchAttributes,
                sort: [`${filter.sort}:${filter.sortOrder == "+" ? "asc" : "desc"}`],
                hitsPerPage: perPage,
                page: page
            }
        }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const result: { page: number, totalPages: number, hits: Hits<TrailSearchResult> } = await r.json();

    if (result.hits.length == 0) {
        return { items: [], ...result };
    }

    const resultTrails: Trail[] = await searchResultToTrailList(result.hits)

    return { items: resultTrails, ...result };

}

export async function profile_stats_index(handle: string, filter: SummitLogFilter, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const filterText = buildFilterText(filter);

    const r = await f(`/api/v1/profile/${handle}/stats?` + new URLSearchParams({
        filter: filterText,
        expand: "trail.category,author",
        sort: "+date",
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const result: SummitLog[] = await r.json();

    return result;

}