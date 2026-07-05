import { Waypoint } from "$lib/models/waypoint";
import { APIError } from "$lib/util/api_util";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";
import type { AuthRecord } from "pocketbase";
import { assets_attach_to_target, assets_delete_removed } from "./asset_store";

export const waypoint: Writable<Waypoint> = writable(new Waypoint(0, 0));

export async function waypoints_create(waypoint: Waypoint, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch, user?: AuthRecord) {
    user ??= get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    waypoint.author = user.actor

    let r = await f('/api/v1/waypoint', {
        method: 'PUT',
        body: JSON.stringify({ ...waypoint, photos: [], _photos: undefined, _assetCandidates: undefined, _assetLinks: undefined, _assetPluginLinks: undefined }),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    let model: Waypoint = await r.json();
    model.photos = await assets_attach_to_target({
        files: waypoint._photos,
        assetIds: waypoint._assetLinks,
        pluginLinks: waypoint._assetPluginLinks,
        target: {
            trail: model.trail ?? waypoint.trail,
            waypoint: model.id,
        },
        existingPhotos: model.photos,
        f,
    });

    return model;

}

export async function waypoints_update(oldWaypoint: Waypoint, newWaypoint: Waypoint) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }
    newWaypoint.author = user.id

    let r = await fetch('/api/v1/waypoint/' + newWaypoint.id, {
        method: 'POST',
        body: JSON.stringify({ ...newWaypoint, photos: [], _photos: undefined, _assetCandidates: undefined, _assetLinks: undefined, _assetPluginLinks: undefined }),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: Waypoint = await r.json();
    await assets_delete_removed(oldWaypoint.photos, newWaypoint.photos, {
        waypoint: model.id,
    });
    model.photos = await assets_attach_to_target({
        files: newWaypoint._photos,
        assetIds: newWaypoint._assetLinks,
        pluginLinks: newWaypoint._assetPluginLinks,
        target: {
            trail: model.trail ?? newWaypoint.trail,
            waypoint: model.id,
        },
        existingPhotos: model.photos ?? newWaypoint.photos,
    });

    return model;

}

export async function waypoints_delete(waypoint: Waypoint) {
    const r = await fetch('/api/v1/waypoint/' + waypoint.id, {
        method: 'DELETE',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

}
