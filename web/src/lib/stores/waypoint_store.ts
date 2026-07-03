import { Waypoint } from "$lib/models/waypoint";
import { APIError } from "$lib/util/api_util";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";
import type { AuthRecord } from "pocketbase";
import { asset_photo_url, assets_create, assets_delete_removed, assets_link } from "./asset_store";

export const waypoint: Writable<Waypoint> = writable(new Waypoint(0, 0));

export async function waypoints_create(waypoint: Waypoint, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch, user?: AuthRecord) {
    user ??= get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    waypoint.author = user.actor

    let r = await f('/api/v1/waypoint', {
        method: 'PUT',
        body: JSON.stringify({ ...waypoint, photos: [], _photos: undefined, _assetCandidates: undefined, _assetLinks: undefined }),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    let model: Waypoint = await r.json();
    if (waypoint._photos && waypoint._photos.length) {
        const assets = await assets_create(waypoint._photos, {
            waypoint: model.id,
        }, f);
        model.photos = assets.map(asset_photo_url);
    }
    if (waypoint._assetLinks?.length) {
        const assets = await assets_link(waypoint._assetLinks, {
            waypoint: model.id,
        }, f);
        model.photos = [...(model.photos ?? []), ...assets.map(asset_photo_url)];
    }

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
        body: JSON.stringify({ ...newWaypoint, photos: [], _photos: undefined, _assetCandidates: undefined, _assetLinks: undefined }),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: Waypoint = await r.json();
    await assets_delete_removed(oldWaypoint.photos, newWaypoint.photos, {
        waypoint: model.id,
    });
    if (newWaypoint._photos?.length) {
        const assets = await assets_create(newWaypoint._photos, {
            waypoint: model.id,
        });
        model.photos = [...newWaypoint.photos, ...assets.map(asset_photo_url)];
    }
    if (newWaypoint._assetLinks?.length) {
        const assets = await assets_link(newWaypoint._assetLinks, {
            waypoint: model.id,
        });
        model.photos = [...(model.photos ?? newWaypoint.photos ?? []), ...assets.map(asset_photo_url)];
    }

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
