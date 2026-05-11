import type { Actor } from "$lib/models/activitypub/actor";
import type { Follow } from "$lib/models/follow";
import { APIError } from "$lib/util/api_util";
import { type ListResult } from "pocketbase";

let follows: Actor[] = [];

export async function follows_index(page: number = 1, perPage: number = 10, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const r = await f(`/api/v1/follow?` + new URLSearchParams({
        page: page.toString(),
        perPage: perPage.toString(),
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedFollows: ListResult<Actor> = await r.json();

    const result = page > 1 ? [...follows, ...fetchedFollows.items] : fetchedFollows.items

    follows = result;

    return { ...fetchedFollows, items: result };
}

export async function follows_a_b(a: string, b: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/follow?' + new URLSearchParams({
        filter: `follower='${a}'&&followee='${b}'`,
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: ListResult<Follow> = await r.json();

    return response.items.at(0) ?? null;
}

export async function follows_create(followee: string) {
    let r = await fetch('/api/v1/follow', {
        method: 'PUT',
        body: JSON.stringify({ followee }),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function follows_update(follow: Follow) {
    let r = await fetch('/api/v1/follow/' + follow.id, {
        method: 'POST',
        body: JSON.stringify(follow),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function follows_delete(follow: Follow) {
    const r = await fetch('/api/v1/follow/' + follow.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}