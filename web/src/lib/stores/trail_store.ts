import type { SummitLog } from "$lib/models/summit_log";
import type { Tag } from "$lib/models/tag";
import { MAP_MAX_POLYLINES } from "$lib/config/map";
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

const DETAILED_CACHE_MAX_SIZE = Math.max(200, MAP_MAX_POLYLINES * 10);

let trails: Trail[] = []
const detailedCache = new Map<string, Trail>();
let detailedCacheKey = "";

function getDetailedCache(id: string): Trail | undefined {
    const cached = detailedCache.get(id);
    if (!cached) {
        return undefined;
    }

    detailedCache.delete(id);
    // Reinsert the entry so Map iteration order tracks recent usage for LRU eviction.
    detailedCache.set(id, cached);
    return cached;
}

function setDetailedCache(id: string, trail: Trail) {
    detailedCache.delete(id);
    detailedCache.set(id, trail);

    while (detailedCache.size > DETAILED_CACHE_MAX_SIZE) {
        const oldestKey = detailedCache.keys().next().value;
        if (!oldestKey) {
            break;
        }
        detailedCache.delete(oldestKey);
    }
}

export const trail: Writable<Trail> = writable(new Trail(""));

export const editTrail: Writable<Trail> = writable(new Trail(""));

export async function trails_search_bounding_box(
    northEast: M.LngLat,
    southWest: M.LngLat,
    filter: TrailFilter,
    page: number = 1,
    zoom: number = 11,
    perPage: number = 50,
    loadMapData: boolean = true
) {
    const user = get(currentUser)

    let filterText: string = "";

    if (filter) {
        filterText = buildFilterText(user, filter, false);
    }

    let lonFilter = `max_lon >= ${southWest.lng} AND min_lon <= ${northEast.lng}`;
    if (southWest.lng > northEast.lng) {
        lonFilter = `(max_lon >= ${southWest.lng} OR min_lon <= ${northEast.lng})`;
    }

    const geoFilter = `max_lat >= ${southWest.lat} AND min_lat <= ${northEast.lat} AND ${lonFilter}`;
    const listFilter = [filterText, geoFilter].filter(Boolean).join(" AND ");
    const cacheKey = JSON.stringify({
        q: filter.q,
        filterText,
        sort: filter.sort,
        sortOrder: filter.sortOrder,
    });
    if (cacheKey !== detailedCacheKey) {
        detailedCache.clear();
        detailedCacheKey = cacheKey;
    }

    // Step 1: Fetch paginated trails for the side list.
    const listResponse = await fetch("/api/v1/search/trails", {
        method: "POST",
        body: JSON.stringify({
            q: filter.q,
            options: {
                filter: listFilter,
                attributesToRetrieve: defaultTrailSearchAttributes,
                sort: [`${filter.sort}:${filter.sortOrder == "+" ? "asc" : "desc"}`],
                hitsPerPage: perPage,
                page,
            },
        }),
    });

    if (!listResponse.ok) {
        const response = await listResponse.json();
        throw new APIError(listResponse.status, response.message, response.detail)
    }

    const listResult: { page: number, totalPages: number, totalHits?: number, estimatedTotalHits?: number, hits: Hits<TrailSearchResult> } = await listResponse.json();
    const listTrails = listResult.hits.length > 0
        ? await searchResultToTrailList(listResult.hits)
        : [];

    trails = page > 1 ? trails.concat(listTrails) : listTrails;

    if (!loadMapData) {
        return {
            trails,
            mapTrails: [],
            clusters: undefined,
            estimatedTotalHits: listResult.estimatedTotalHits,
            totalHits: listResult.totalHits ?? listResult.estimatedTotalHits,
            totalPages: listResult.totalPages
        };
    }

    // Step 2: Fetch server-side clusters and unclustered points for the map.
    let cr = await fetch("/api/v1/search/trails/cluster", {
        method: "POST",
        body: JSON.stringify({
            southWest: { lat: southWest.lat, lng: southWest.lng },
            northEast: { lat: northEast.lat, lng: northEast.lng },
            zoom,
            q: filter.q,
            filterText
        })
    });

    if (!cr.ok) {
        const response = await cr.json();
        throw new APIError(cr.status, response.message, response.detail)
    }

    const clusterResult = await cr.json();
    const clusterFeatureCollection = clusterResult;

    const unclusteredFeatures = clusterFeatureCollection.features
        .filter((f: any) => !f.properties.cluster);

    // Extract IDs of visible unclustered points that are large enough to show details for
    const unclusteredIds = unclusteredFeatures
        .filter((f: any) => f.properties.is_large)
        .map((f: any) => f.properties.id);

    // Step 3: Identify which visible trails are MISSING from the local cache
    const missingIds = unclusteredIds.filter((id: string) => !detailedCache.has(id));

    // Step 4: Only fetch details for missing trails
    if (missingIds.length > 0) {
        const batchSize = 100; // Meilisearch filter length safety
        for (let i = 0; i < missingIds.length; i += batchSize) {
            const batch = missingIds.slice(i, i + batchSize);
            const detailBatchQuery = {
                indexUid: "trails",
                q: "",
                filter: [`id IN [${batch.map((id: string) => `'${id}'`).join(",")}]`],
                attributesToRetrieve: [...defaultTrailSearchAttributes, "polyline"],
                hitsPerPage: batchSize,
            };

            const dr = await fetch("/api/v1/search/multi", {
                method: "POST",
                body: JSON.stringify({ queries: [detailBatchQuery] }),
            });

            if (dr.ok) {
                const detailResult = await dr.json();
                const newTrails = await searchResultToTrailList(detailResult.results[0].hits);
                // Populate cache
                newTrails.forEach(t => {
                    if (t.id) setDetailedCache(t.id, t)
                });
            }
        }
    }

    // Step 5: Convert unclustered hits to lightweight Trail objects for map popups/previews.
    const mapTrails: Trail[] = unclusteredFeatures
        .map((f: any) => {
        const s = f.properties;
        const lat = f.geometry.coordinates[1];
        const lng = f.geometry.coordinates[0];

        const cached = getDetailedCache(s.id);
        if (cached) {
            return {
                ...cached,
                lat,
                lon: lng,
                bounding_box_diagonal: s.bounding_box_diagonal ?? cached.bounding_box_diagonal,
                // Strip polyline for small trails so they don't linger as lines when zoomed out
                polyline: s.is_large ? cached.polyline : undefined,
            };
        }

        // Lightweight fallback for map markers
        const t: Trail & RecordModel = {
            id: s.id,
            lat: lat,
            lon: lng,
            name: "",
            author: "",
            photos: [],
            public: true,
            completed: false,
            summit_logs: [],
            waypoints: [],
            tags: [],
            category: "",
            created: new Date(0).toISOString(),
            date: new Date(0).toISOString(),
            updated: new Date(0).toISOString(),
            description: "",
            difficulty: "easy",
            distance: 0,
            duration: 0,
            elevation_gain: 0,
            elevation_loss: 0,
            location: "",
            bounding_box_diagonal: s.bounding_box_diagonal ?? 0,
            like_count: 0,
            collectionId: "trails",
            collectionName: "trails",
            expand: { author: {} as any }
        };
        return t;
    });

    return {
        trails,
        mapTrails,
        clusters: clusterFeatureCollection,
        estimatedTotalHits: listResult.estimatedTotalHits ?? clusterResult.totalHits,
        totalHits: listResult.totalHits ?? listResult.estimatedTotalHits ?? clusterResult.totalHits,
        totalPages: listResult.totalPages
    };
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

    const formData = objectToFormData(trail, ["expand"])

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

    const createdSummitLogs: SummitLog[] = [];
    for (const summitLog of trail.expand?.summit_logs_via_trail ?? []) {
        summitLog.trail = model.id!;
        createdSummitLogs.push(await summit_logs_create(summitLog, f));
    }

    const createdWaypoints: Waypoint[] = [];
    for (const wp of trail.expand?.waypoints_via_trail ?? []) {
        wp.trail = model.id!;
        createdWaypoints.push(await waypoints_create({
            ...wp,
            marker: undefined,
        }, f, user));
    }

    if (!model.expand) {
        model.expand = {};
    }

    if (createdSummitLogs.length) {
        model.expand.summit_logs_via_trail = [
            ...(model.expand.summit_logs_via_trail ?? []),
            ...createdSummitLogs,
        ];
    }

    if (createdWaypoints.length) {
        model.expand.waypoints_via_trail = [
            ...(model.expand.waypoints_via_trail ?? []),
            ...createdWaypoints,
        ];
    }

    return model;

}

export async function trails_update(oldTrail: Trail, newTrail: Trail, photos?: File[], gpx?: File | Blob | null, exclude?: (keyof Trail)[]) {
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

    const formData = objectToFormData(newTrail, ["expand", ...(exclude ?? [])])

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


    const updateUrl = `/api/v1/trail/form/${newTrail.id}?` + new URLSearchParams({
        expand: "category,waypoints_via_trail,summit_logs_via_trail,trail_share_via_trail,tags",
    });

    let r = await fetch(updateUrl, {
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

export async function trails_update_metadata(
    currentTrail: Trail,
    patch: Pick<Partial<Trail>, "name" | "description" | "tags"> & {
        expand?: Pick<NonNullable<Trail["expand"]>, "tags">;
    },
) {
    const tagIds: string[] | undefined = patch.expand?.tags
        ? []
        : patch.tags;

    for (const tag of patch.expand?.tags ?? []) {
        if (!tag.id) {
            const model = await tags_create(tag);
            tagIds!.push(model.id!);
        } else {
            tagIds!.push(tag.id);
        }
    }

    const searchParams = new URLSearchParams(
        tagIds !== undefined ? { expand: "tags" } : {},
    );
    const query = searchParams.toString();
    const url = `/api/v1/trail/${currentTrail.id}${query ? `?${query}` : ""}`;
    const payload = {
        name: patch.name ?? currentTrail.name,
        ...(patch.description !== undefined
            ? { description: patch.description }
            : {}),
        ...(tagIds !== undefined ? { tags: tagIds } : {}),
    };

    const r = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const model: Trail = await r.json();
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
        const created = Number(h.created || 0);
        const date = Number(h.date || 0);
        const t: Trail & RecordModel = {
            collectionId: "trails",
            collectionName: "trails",
            updated: new Date(created * 1000).toISOString(),
            author: h.author_name,
            name: h.name,
            photos: h.thumbnail ? [h.thumbnail] : [],
            public: h.public,
            completed: h.completed,
            summit_logs: [],
            waypoints: [],
            tags: h.tags ?? [],
            category: h.category,
            created: new Date(created * 1000).toISOString(),
            date: new Date(date * 1000).toISOString(),
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
            bounding_box_diagonal: h.bounding_box_diagonal ?? 0,
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
