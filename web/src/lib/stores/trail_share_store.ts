import type { TrailShare } from "#lib/models/trail_share.ts";
import { APIError } from "#lib/util/api_util.ts";
import { type ListResult } from "pocketbase";
import { writable, type Writable } from "svelte/store";

export const shares: Writable<TrailShare[]> = writable([])


export async function trail_share_index(data: { trail?: string, actor?: string, actorIRI?: string }, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let filter = ""
    if (data.trail) {
        filter += `trail='${data.trail}'`
    } else if (data.actor) {
        filter += `actor='${data.actor}'`
    } else if (data.actorIRI) {
        filter += `actor.iri='${data.actorIRI}'`
    }

    const r = await f('/api/v1/trail-share?' + new URLSearchParams({
        filter: filter,
        expand: "actor"
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: ListResult<TrailShare> = await r.json();

    shares.set(response.items);

    return response.items;

}

export async function trail_share_create(share: TrailShare) {
    let r = await fetch('/api/v1/trail-share', {
        method: 'PUT',
        body: JSON.stringify(share),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function trail_share_update(share: TrailShare) {
    let r = await fetch('/api/v1/trail-share/' + share.id, {
        method: 'POST',
        body: JSON.stringify(share),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function trail_share_delete(share: TrailShare) {
    const r = await fetch('/api/v1/trail-share/' + share.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}