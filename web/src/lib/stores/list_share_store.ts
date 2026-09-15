import type { Actor } from "$lib/models/activitypub/actor";
import type { List } from "$lib/models/list";
import { ListShare } from "$lib/models/list_share";
import { TrailShare } from "$lib/models/trail_share";
import { APIError } from "$lib/util/api_util";
import { type ListResult } from "pocketbase";
import { writable, type Writable } from "svelte/store";
import { trail_share_create, trail_share_index } from "./trail_share_store";

export const shares: Writable<ListShare[]> = writable([])


export async function list_share_index(list: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/list-share?' + new URLSearchParams({
        filter: `list='${list}'`,
        expand: "actor"
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: ListResult<ListShare> = await r.json();

    shares.set(response.items);

    return response;

}

export async function list_share_create(share: ListShare) {
    let r = await fetch('/api/v1/list-share', {
        method: 'PUT',
        body: JSON.stringify(share),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function list_share_create_for_actor(
    list: List,
    actor: Pick<Actor, "iri" | "is_local">,
    currentActorId?: string,
) {
    await list_share_create(new ListShare(actor.iri, list.id!, "view"));

    if (actor.is_local) {
        const existingShares = await trail_share_index({ actorIRI: actor.iri });
        const sharedTrailIds = new Set(existingShares.map((share) => share.trail));
        for (const trail of list.expand?.trails ?? []) {
            if (
                !trail.public &&
                trail.author === currentActorId &&
                !sharedTrailIds.has(trail.id!)
            ) {
                await trail_share_create(new TrailShare(actor.iri, trail.id!, "view"));
            }
        }
    }

    return list_share_index(list.id!);
}

export async function list_share_update(share: ListShare) {
    let r = await fetch('/api/v1/list-share/' + share.id, {
        method: 'POST',
        body: JSON.stringify(share),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function list_share_delete(share: ListShare) {
    const r = await fetch('/api/v1/list-share/' + share.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}
