import { Waypoint } from "$lib/models/waypoint";
import { APIError } from "$lib/util/api_util";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";
import type { AuthRecord } from "pocketbase";

export const waypoint: Writable<Waypoint> = writable(new Waypoint(0, 0));

export async function waypoints_create(waypoint: Waypoint, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch, user?: AuthRecord) {
    user ??= get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    waypoint.author = user.actor

    let r = await f('/api/v1/waypoint', {
        method: 'PUT',
        body: JSON.stringify(waypoint),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    if (waypoint._photos && waypoint._photos.length) {
        let model: Waypoint = await r.json();

        const formData = new FormData()

        for (const photo of waypoint._photos) {
            formData.append("photos", photo)
        }

        r = await fetch(`/api/v1/waypoint/${model.id!}/file`, {
            method: 'POST',
            body: formData,
        })

        if (!r.ok) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response.detail)
        }
    }

    return await r.json();

}

export async function waypoints_update(oldWaypoint: Waypoint, newWaypoint: Waypoint) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }
    newWaypoint.author = user.id

    let r = await fetch('/api/v1/waypoint/' + newWaypoint.id, {
        method: 'POST',
        body: JSON.stringify(newWaypoint),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const formData = new FormData()

    for (const photo of newWaypoint._photos ?? []) {
        formData.append("photos", photo)
    }

    const deletedPhotos = oldWaypoint.photos.filter(oldPhoto => !newWaypoint.photos.find(newPhoto => newPhoto === oldPhoto));

    for (const deletedPhoto of deletedPhotos) {
        formData.append("photos-", deletedPhoto.replace(/^.*[\\/]/, ''));
    }

    r = await fetch(`/api/v1/waypoint/${newWaypoint.id!}/file`, {
        method: 'POST',
        body: formData,
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

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