import type { SummitLog } from "$lib/models/summit_log";
import type { Tag } from "$lib/models/tag";
import { defaultTrailSearchAttributes, Trail, type TrailFilter, type TrailFilterValues, type TrailSearchResult } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";
import { APIError } from "$lib/util/api_util";
import { deepEqual } from "$lib/util/deep_util";
import { getFileURL, objectToFormData } from "$lib/util/file_util";
import * as M from "maplibre-gl";
import type { Hits } from "meilisearch";
import { type AuthRecord, type ListResult, type RecordModel } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { summit_logs_create, summit_logs_delete, summit_logs_update } from "./summit_log_store";
import { tags_create } from "./tag_store";
import { currentUser } from "./user_store";
import { waypoints_create, waypoints_delete, waypoints_update } from "./waypoint_store";

let trails: Trail[] = []
export const trail: Writable<Trail> = writable(new Trail(""));

export const editTrail: Writable<Trail> = writable(new Trail(""));

export async function trails_index(perPage: number = 21, random: boolean = false, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/trail?' + new URLSearchParams({
        "perPage": perPage.toString(),
        expand: "category,waypoints_via_trail,summit_logs_via_trail,tags",
        sort: random ? "@random" : "",
    }), {
        method: 'GET',
    })
    const response: ListResult<Trail> = await r.json()

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    trails = response.items
    return response.items;

}

export async function trails_recommend(size: number, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/trail/recommend?' + new URLSearchParams({
        "size": size.toString(),
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
    const response: Hits<TrailSearchResult> = await r.json()

    return searchResultToTrailList(response);

}

export async function trails_search_filter(filter: TrailFilter, page: number = 1, perPage: number, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const user = get(currentUser)

    let filterText: string = buildFilterText(user, filter, true);


    let r = await f("/api/v1/search/trails", {
        method: "POST",
        body: JSON.stringify({
            q: filter.q,
            options: {
                filter: filterText,
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

export async function trails_search_bounding_box(northEast: M.LngLat, southWest: M.LngLat, filter: TrailFilter, page: number = 1, includePolyline: boolean = true) {
    const user = get(currentUser)

    let filterText: string = "";

    if (filter) {
        filterText = buildFilterText(user, filter, false);
    }

    let r = await fetch("/api/v1/search/trails", {
        method: "POST",
        body: JSON.stringify({
            q: "",
            options: {
                filter: [
                    `_geoBoundingBox([${northEast.lat}, ${northEast.lng}], [${southWest.lat}, ${southWest.lng}])`,
                    filterText
                ],
                sort: [`${filter.sort}:${filter.sortOrder == "+" ? "asc" : "desc"}`,],
                attributesToRetrieve: [...defaultTrailSearchAttributes, ...(includePolyline ? ["polyline"] : [])],
                hitsPerPage: 500,
                page: page
            }
        }),
    });
    const result: { page: number, totalPages: number, hits: Hits<TrailSearchResult> } = await r.json();

    if (result.hits.length == 0) {
        trails = [];
        return { trails: [], ...result }
    }

    const resultTrails: Trail[] = await searchResultToTrailList(result.hits)

    trails = page > 1 ? trails.concat(resultTrails) : resultTrails

    return { trails, ...result };


}

export async function trails_show(id: string, handle?: string, share?: string, loadGPX?: boolean, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const r = await f(`/api/v1/trail/${id}?` + new URLSearchParams({
        expand: "category,waypoints_via_trail,summit_logs_via_trail,summit_logs_via_trail.author,trail_share_via_trail.actor,trail_like_via_trail,tags,author",
        ...(handle ? { handle } : {}),
        ...(share ? { share } : {})
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: Trail = await r.json()

    if (loadGPX) {
        if (!response.expand) {
            response.expand = {}
        }
        const gpxData: string = await fetchGPX(response, f);
        if (!response.expand) {
            response.expand = {};
        }
        response.expand.gpx_data = gpxData;


        for (const log of response.expand.summit_logs_via_trail ?? []) {
            const gpxData: string = await fetchGPX(log, f);

            if (!log.expand) {
                log.expand = {};
            }
            log.expand.gpx_data = gpxData;
        }
    }

    response.expand!.waypoints_via_trail = response.expand!.waypoints_via_trail || [];
    response.expand!.summit_logs_via_trail = response.expand!.summit_logs_via_trail?.sort((a: SummitLog, b: SummitLog) => Date.parse(a.date) - Date.parse(b.date)) || [];

    trail.set(response);

    return response as Trail;
}

export async function trails_create(trail: Trail, photos: File[], gpx: File | Blob | null, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch, user?: AuthRecord) {
    user ??= get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    for (const tag of trail.expand?.tags ?? []) {
        if (!tag.id) {
            const model = await tags_create(tag)
            trail.tags.push(model.id!)
        } else {
            trail.tags.push(tag.id)
        }
    }

    trail.author = user.actor

    const formData = objectToFormData(trail)

    if (gpx) {
        formData.set("gpx", gpx);
    }

    for (const photo of photos) {
        formData.set("photos", photo)
    }

    let r = await f(`/api/v1/trail/form?` + new URLSearchParams({
        expand: "category,waypoints_via_trail,summit_logs_via_trail,trail_share_via_trail,tags",
    }), {
        method: 'PUT',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    let model: Trail = await r.json();

    for (const summitLog of trail.expand?.summit_logs_via_trail ?? []) {
        summitLog.trail = model.id!;
        await summit_logs_create(summitLog, f);
    }

    for (const wp of trail.expand?.waypoints_via_trail ?? []) {
        wp.trail = model.id!;
        await waypoints_create({
            ...wp,
            marker: undefined,
        }, f, user);
    }

    return model;

}

export async function trails_update(oldTrail: Trail, newTrail: Trail, photos?: File[], gpx?: File | Blob | null) {
    newTrail.author = oldTrail.author

    const waypointUpdates = compareObjectArrays<Waypoint>(oldTrail.expand?.waypoints_via_trail ?? [], newTrail.expand?.waypoints_via_trail ?? []);

    for (const addedWaypoint of waypointUpdates.added) {
        addedWaypoint.trail = newTrail.id!
        const model = await waypoints_create({
            ...addedWaypoint,
            marker: undefined,
        },);
    }

    for (const updatedWaypoint of waypointUpdates.updated) {
        const oldWaypoint = oldTrail.expand?.waypoints_via_trail?.find(w => w.id == updatedWaypoint.id);
        const model = await waypoints_update(oldWaypoint!, {
            ...updatedWaypoint,
            marker: undefined,
        });
    }

    for (const deletedWaypoint of waypointUpdates.deleted) {
        const success = await waypoints_delete(deletedWaypoint);
    }

    const summitLogUpdates = compareObjectArrays<SummitLog>(oldTrail.expand?.summit_logs_via_trail ?? [], newTrail.expand?.summit_logs_via_trail ?? []);

    for (const summitLog of summitLogUpdates.added) {
        summitLog.trail = newTrail.id!
        const model = await summit_logs_create(summitLog);
    }

    for (const updatedSummitLog of summitLogUpdates.updated) {
        const oldSummitLog = oldTrail.expand?.summit_logs_via_trail?.find(w => w.id == updatedSummitLog.id);

        const model = await summit_logs_update(oldSummitLog!, updatedSummitLog);
    }

    for (const deletedSummitLog of summitLogUpdates.deleted) {
        const success = await summit_logs_delete(deletedSummitLog);
    }

    const tagUpdates = compareObjectArrays<Tag>(oldTrail.expand?.tags ?? [], newTrail.expand?.tags ?? []);

    for (const tag of tagUpdates.added) {
        if (!tag.id) {
            const model = await tags_create(tag)
            newTrail.tags.push(model.id!)
        } else {
            newTrail.tags.push(tag.id)
        }
    }

    for (const tag of tagUpdates.deleted) {
        newTrail.tags = newTrail.tags.filter(t => t != tag.id);
    }

    const formData = objectToFormData(newTrail, ["expand"])

    if (gpx) {
        formData.append("gpx", gpx);
    }

    if (photos) {
        for (const photo of photos) {
            formData.append("photos+", photo)
        }
    }

    const deletedPhotos = oldTrail.photos.filter(oldPhoto => !newTrail.photos.find(newPhoto => newPhoto === oldPhoto));

    for (const deletedPhoto of deletedPhotos) {
        formData.append("photos-", deletedPhoto.replace(/^.*[\\/]/, ''));
    }


    let r = await fetch(`/api/v1/trail/form/${newTrail.id}?` + new URLSearchParams({
        expand: "category,waypoints_via_trail,summit_logs_via_trail,trail_share_via_trail,tags",
    }), {
        method: 'POST',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    let model: Trail = await r.json();

    for (const log of model.expand?.summit_logs_via_trail ?? []) {
        if (!log.expand) {
            log.expand = {};
        }
    }

    trail.set(model);

    return model;
}


export async function trails_delete(trail: Trail) {
    const r = await fetch('/api/v1/trail/' + trail.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    return await r.json();

}

export async function trails_get_filter_values(f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch): Promise<TrailFilterValues> {
    const r = await f('/api/v1/trail/filter', {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

}

export async function trails_get_bounding_box(f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch): Promise<TrailFilterValues> {
    const r = await f('/api/v1/trail/bounding-box', {
        method: 'GET',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

}

export async function trails_upload(file: File, ignoreDuplicates: boolean = false, onProgress?: (progress: number) => void) {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        const fd = new FormData();
        fd.append("name", file.name);
        fd.append("file", file);
        fd.append("ignoreDuplicates", ignoreDuplicates ? "true" : "false")

        xhr.open("PUT", "/api/v1/trail/upload", true);

        xhr.upload.onprogress = function (event) {
            if (event.lengthComputable) {
                const percentComplete = (event.loaded / event.total) * 100;
                onProgress?.(percentComplete)
            }
        };

        xhr.onload = async () => {
            const responseText = xhr.responseText;
            const response = responseText ? JSON.parse(responseText) : null;

            if (xhr.status >= 200 && xhr.status < 300) {
                resolve(response);
            } else {
                reject(new APIError(xhr.status, response?.message || "Upload failed", response));
            }
        };

        xhr.onerror = () => {
            reject(new APIError(xhr.status, xhr.statusText));
        };

        xhr.send(fd);
    });
}

export async function fetchGPX(trail: { gpx?: string } & Record<string, any>, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    if (!trail.gpx) {
        return "";
    }
    const gpxUrl = getFileURL(trail, trail.gpx);
    const response: Response = await f(gpxUrl);
    const gpxData = await response.text();

    return gpxData
}

export async function searchResultToTrailList(hits: Hits<TrailSearchResult>): Promise<Trail[]> {
    const trails: Trail[] = []
    for (const h of hits) {
        const t: Trail & RecordModel = {
            collectionId: "trails",
            collectionName: "trails",
            updated: new Date(h.created * 1000).toISOString(),
            author: h.author_name,
            name: h.name,
            photos: h.thumbnail ? [h.thumbnail] : [],
            public: h.public,
            summit_logs: [],
            waypoints: [],
            tags: h.tags ?? [],
            category: h.category,
            created: new Date(h.created * 1000).toISOString(),
            date: new Date(h.date * 1000).toISOString(),
            description: h.description,
            difficulty: h.difficulty == 0 ? "easy" : h.difficulty == 1 ? "moderate" : "difficult",
            distance: h.distance,
            duration: h.duration,
            elevation_gain: h.elevation_gain,
            elevation_loss: h.elevation_loss,
            id: h.id,
            lat: h._geo.lat,
            lon: h._geo.lng,
            location: h.location,
            gpx: h.gpx,
            polyline: h.polyline,
            domain: h.domain,
            iri: h.iri,
            thumbnail: 0,
            like_count: h.like_count,
            expand: {
                author: {
                    collectionId: "activitypub_actors",
                    isLocal: (h.domain?.length ?? 0) == 0,
                    id: h.author,
                    icon: h.author_avatar,
                    preferred_username: h.author_name,
                    domain: h.domain,
                } as any,
                trail_share_via_trail: h.shares?.map(s => ({
                    permission: "view",
                    trail: h.id,
                    actor: s,
                })),
                trail_like_via_trail: h.likes?.map(l => ({
                    trail: h.id,
                    actor: l,
                }))
            }
        }


        trails.push(t)
    }

    return trails
}

function buildFilterText(user: AuthRecord, filter: TrailFilter, includeGeo: boolean): string {
    let filterText: string = "";

    filterText += `distance >= ${Math.floor(filter.distanceMin)} AND elevation_gain >= ${Math.floor(filter.elevationGainMin)} AND elevation_loss >= ${Math.floor(filter.elevationLossMin)}`

    if (filter.distanceMax < filter.distanceLimit) {
        filterText += ` AND distance <= ${Math.ceil(filter.distanceMax)}`
    }

    if (filter.elevationGainMax < filter.elevationGainLimit) {
        filterText += ` AND elevation_gain <= ${Math.ceil(filter.elevationGainMax)}`
    }

    if (filter.elevationLossMax < filter.elevationLossLimit) {
        filterText += ` AND elevation_loss <= ${Math.ceil(filter.elevationLossMax)}`
    }

    if (filter.difficulty.length > 0) {
        filterText += ` AND difficulty IN [${filter.difficulty.join(",")}]`
    }

    if (filter.author?.length) {
        filterText += ` AND author = ${filter.author}`
    }

    if (filter.public !== undefined || filter.private !== undefined || filter.shared !== undefined) {
        filterText += " AND ("

        const showPublic = filter.public === undefined || filter.public === true;
        const showPrivate = filter.private === undefined || filter.private === true;
        const showShared = filter.shared !== undefined && filter.shared === true;

        if (showPublic === true) {
            filterText += "(public = TRUE";
            if (showPrivate === true && (!filter.author?.length || filter.author == user?.actor)) {
                filterText += ` OR author = ${user?.actor}`;
            }
            filterText += ")";
        }
        else if (!filter.author?.length || filter.author == user?.actor) {
            filterText += "public = FALSE";
            filterText += ` AND author = ${user?.actor}`;
        }

        if (filter.shared !== undefined) {
            if (filter.shared === true) {
                filterText += ` OR shares = ${user?.actor}`
            } else {
                filterText += ` AND NOT shares = ${user?.actor}`

            }
        }

        filterText += ")";
    }

    /*
    if (filter.public !== undefined || filter.shared !== undefined) {
        filterText += " AND ("
        if (filter.public !== undefined) {
            filterText += `(public = ${filter.public}`

            if (!filter.author?.length || filter.author == user?.actor) {
                filterText += ` OR author = ${user?.actor}`
            }
            filterText += ")"
        }

        if (filter.shared !== undefined) {
            if (filter.shared === true) {
                filterText += ` OR shares = ${user?.actor}`
            } else {
                filterText += ` AND NOT shares = ${user?.actor}`

            }
        }
        filterText += ")"
    }
*/

    if (filter.liked === true) {
        filterText += ` AND likes = ${user?.actor}`
    }

    if (filter.startDate) {
        filterText += ` AND date >= ${new Date(filter.startDate).getTime() / 1000}`
    }

    if (filter.endDate) {
        filterText += ` AND date <= ${new Date(filter.endDate).getTime() / 1000}`
    }

    if (filter.category.length > 0) {
        const categoryValues = filter.category.map(category => `'${category}'`).join(", ");
        filterText += ` AND category IN [${categoryValues}]`;
    }

    if (filter.tags.length > 0) {
        filterText += ` AND (${filter.tags.map(t => `tags = '${t}'`).join(" OR ")})`;
    }

    if (filter.completed !== undefined) {
        filterText += ` AND completed = ${filter.completed}`;
    }

    if (filter.near.lat && filter.near.lon && includeGeo) {
        filterText += ` AND _geoRadius(${filter.near.lat}, ${filter.near.lon}, ${filter.near.radius})`
    }
    if (filter.near.lat && filter.near.lon && includeGeo) {
        filterText += ` AND _geoRadius(${filter.near.lat}, ${filter.near.lon}, ${filter.near.radius})`
    }

    return filterText
}

function compareObjectArrays<T extends { id?: string }>(oldArray: T[], newArray: T[]) {
    const newObjects = [];
    const updatedObjects = [];
    const unchangedObjects = [];
    for (const newObj of newArray) {
        const oldObj = oldArray.find(oldObj => oldObj.id === newObj.id)
        if (!oldObj) {
            newObjects.push(newObj);
        } else if (!deepEqual(newObj, oldObj)) {
            updatedObjects.push(newObj);
        } else {
            unchangedObjects.push(newObj);
        }
    }
    const deletedObjects = oldArray.filter(oldObj => !newArray.find(newObj => newObj.id === oldObj.id));

    return {
        added: newObjects,
        deleted: deletedObjects,
        updated: updatedObjects,
        unchanged: unchangedObjects,
    };
}
