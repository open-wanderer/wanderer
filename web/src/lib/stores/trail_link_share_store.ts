import type { TrailLinkShare } from "$lib/models/trail_link_share";
import { APIError } from "$lib/util/api_util";
import { type ListResult } from "pocketbase";
import { writable, type Writable } from "svelte/store";

export const linkShares: Writable<TrailLinkShare[]> = writable([])


export async function trail_link_share_index(trail?: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let filter = ""
    filter += `trail='${trail}'`

    const r = await f('/api/v1/trail-link-share?' + new URLSearchParams({
        filter: filter,
        expand: "actor"
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: ListResult<TrailLinkShare> = await r.json();

    linkShares.set(response.items);

    return response.items;

}

export async function trail_link_share_create(share: TrailLinkShare) {
    let r = await fetch('/api/v1/trail-link-share', {
        method: 'PUT',
        body: JSON.stringify(share),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function trail_link_share_update(share: TrailLinkShare) {
    let r = await fetch('/api/v1/trail-link-share/' + share.id, {
        method: 'POST',
        body: JSON.stringify(share),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function trail_link_share_delete(share: TrailLinkShare) {
    const r = await fetch('/api/v1/trail-link-share/' + share.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}