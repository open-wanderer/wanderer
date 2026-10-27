import type { TrailLike } from "#lib/models/trail_like.ts";
import { APIError } from "#lib/util/api_util.ts";
import { type ListResult } from "pocketbase";

export async function trail_like_index(data: { trail?: string, actor?: string }, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let filter = ""
    if (data.trail) {
        filter += `trail='${data.trail}'`
    } else if (data.actor) {
        filter += `actor='${data.actor}'`
    }

    const r = await f('/api/v1/trail-like?' + new URLSearchParams({
        filter: filter,
        expand: "actor"
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: ListResult<TrailLike> = await r.json();

    return response.items;
}

export async function trail_like_create(like: TrailLike) {
    let r = await fetch('/api/v1/trail-like', {
        method: 'PUT',
        body: JSON.stringify(like),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: TrailLike =  await r.json()

    return response
}

export async function trail_like_delete(like: TrailLike) {
    const r = await fetch('/api/v1/trail-like/delete', {
        method: 'POST',
        body: JSON.stringify(like)
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}