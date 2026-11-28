import { committedAssetRecord } from "#lib/util/committed_asset_record.ts";
import { assetIdFromPhotoURL, expandedPhotoAssets, replaceCopiedAssets, shownAssetLinks } from "#lib/util/asset_link_util.ts";
import { assets_attach_to_target, assets_delete_removed, assets_set_trail_thumbnail, has_asset_attachments, type CompletedAssetAttachments } from "./asset_store";
import { withFederationSaveBatch } from "#lib/util/federation_save_util.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import type { Tag } from "#lib/models/tag.ts";
import { MAP_MAX_POLYLINES } from "#lib/config/map.ts";
import { defaultTrailSearchAttributes, Trail, type TrailBoundingBox, type TrailFilter, type TrailFilterValues, type TrailSearchResult } from "#lib/models/trail.ts";
import type { Waypoint } from "#lib/models/waypoint.ts";
import { APIError } from "#lib/util/api_util.ts";
import { deepEqual } from "#lib/util/deep_util.ts";
import { getFileURL, objectToFormData } from "#lib/util/file_util.ts";
import { noSubcategoryFilterCategory, trailFilterDateBoundary } from "#lib/util/trail_filter_util.ts";
import { TRAIL_SEARCH_LIMIT } from "#lib/config/search.ts";
import { buildTrailSearchBoundsFilter, type TrailSearchBounds } from "#lib/util/trail_search_util.ts";
import * as M from "maplibre-gl";
import type { Hits } from "meilisearch";
import { type AuthRecord, type ListResult, type RecordModel } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { summit_logs_create, summit_logs_delete, summit_logs_update } from "./summit_log_store";
import { categories } from "./category_store";
import { subcategories } from "./subcategory_store";
import { tags_create } from "./tag_store";
import { currentUser } from "./user_store";
import { waypoints_create, waypoints_delete, waypoints_update } from "./waypoint_store";

export async function trails_index(perPage: number = 21, random: boolean = false, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/trail?' + new URLSearchParams({
        "perPage": perPage.toString(),
        expand: "category,subcategory,subcategory.category,waypoints_via_trail,waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail,summit_logs_via_trail.summit_log_assets_via_summit_log.asset,tags,trail_assets_via_trail.asset",
        sort: random ? "@random" : "",
    }), {
        method: 'GET',
    })
    const response: ListResult<Trail> = await r.json()

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

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

type TrailSearchResponse = {
    page: number;
    hitsPerPage: number;
    totalPages: number;
    totalHits: number;
    hits: Hits<TrailSearchResult>;
};

async function searchTrailResults(
    filter: TrailFilter,
    page: number,
    perPage: number,
    f: typeof fetch,
    bounds?: TrailSearchBounds,
): Promise<TrailSearchResponse> {
    let filterText = buildFilterText(get(currentUser), filter, true);
    if (bounds) {
        filterText += ` AND (${buildTrailSearchBoundsFilter(bounds)})`;
    }

    const sort = filter.sort === "relevance" && !filter.q.trim() ? "created" : filter.sort;
    const response = await f("/api/v1/search/trails", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            q: filter.q,
            options: {
                filter: filterText,
                attributesToRetrieve: defaultTrailSearchAttributes,
                ...(sort !== "relevance" ? {
                    sort: [`${sort}:${filter.sortOrder === "+" ? "asc" : "desc"}`],
                } : {}),
                hitsPerPage: perPage,
                page,
            },
        }),
    });

    if (!response.ok) {
        const cause = await response.json();
        throw new APIError(response.status, cause.message, cause.detail);
    }
    const result: TrailSearchResponse = await response.json();
    return {
        ...result,
        totalPages: Math.min(result.totalPages, Math.ceil(TRAIL_SEARCH_LIMIT / perPage)),
    };
}

export async function trails_search_filter(
    filter: TrailFilter,
    page = 1,
    perPage: number,
    f: typeof fetch = fetch,
    bounds?: TrailSearchBounds,
) {
    const result = await searchTrailResults(filter, page, perPage, f, bounds);
    return { ...result, items: await searchResultToTrailList(result.hits) };
}

/** Refresh the loaded prefix after a bulk action without losing its page position. */
export async function trails_search_filter_pages(filter: TrailFilter, pages: number, perPage = 48, f: typeof fetch = fetch) {
    const result = await trails_search_filter(filter, 1, Math.min(TRAIL_SEARCH_LIMIT, Math.max(1, pages) * perPage), f);
    return { ...result, hitsPerPage: perPage,
        page: Math.max(1, Math.ceil(result.items.length / perPage)),
        totalPages: Math.ceil(Math.min(result.totalHits, TRAIL_SEARCH_LIMIT) / perPage) };
}

/** Find the extent of every matching trail, independently of pagination. */
export async function trails_search_bounds(filter: TrailFilter, f: typeof fetch = fetch): Promise<TrailSearchBounds | undefined> {
    const response = await f("/api/v1/search/trails", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            q: filter.q,
            options: {
                filter: `(${buildFilterText(get(currentUser), filter, true)}) AND NOT (min_lat = 0 AND max_lat = 0 AND min_lon = 0 AND max_lon = 0)`,
                limit: 0,
                facets: ["min_lat", "max_lat", "min_lon", "max_lon"],
            },
        }),
    });
    if (!response.ok) {
        const cause = await response.json();
        throw new APIError(response.status, cause.message, cause.detail ?? cause.data);
    }
    const { facetStats } = await response.json();
    const south = facetStats?.min_lat?.min;
    const north = facetStats?.max_lat?.max;
    const west = facetStats?.min_lon?.min;
    const east = facetStats?.max_lon?.max;
    if (![south, north, west, east].every(Number.isFinite)) return;
    const latitude = (north + south) / 2;
    const longitude = (east + west) / 2;
    // Add ten percent on each side and leave room around trails without tracks.
    const halfHeight = Math.max(.005, (north - south) * .6);
    const halfWidth = Math.max(.005, (east - west) * .6);
    return {
        northEast: { lat: Math.min(90, latitude + halfHeight), lng: Math.min(180, longitude + halfWidth) },
        southWest: { lat: Math.max(-90, latitude - halfHeight), lng: Math.max(-180, longitude - halfWidth) },
    };
}

const DETAILED_CACHE_MAX_SIZE = Math.max(200, MAP_MAX_POLYLINES * 10);

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

/** Fetch markers and polylines independently of the already loaded result list. */
export async function trails_search_map_data(
    northEast: M.LngLat,
    southWest: M.LngLat,
    filter: TrailFilter,
    zoom: number = 11,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const user = get(currentUser)

    let filterText: string = "";

    if (filter) {
        filterText = buildFilterText(user, filter, true);
    }

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

    // Fetch server-side clusters and unclustered points for the map.
    let cr = await f("/api/v1/search/trails/cluster", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            southWest: { lat: southWest.lat, lng: southWest.lng },
            northEast: { lat: northEast.lat, lng: northEast.lng },
            zoom,
            q: filter.q,
            filterText,
        })
    });

    if (!cr.ok) {
        const response = await cr.json();
        throw new APIError(cr.status, response.message, response.detail)
    }

    const clusterFeatureCollection = await cr.json();

    const unclusteredFeatures = clusterFeatureCollection.features
        .filter((f: any) => !f.properties.cluster);

    // Extract IDs of visible unclustered points that are large enough to show details for
    const unclusteredIds = unclusteredFeatures
        .filter((f: any) => f.properties.is_large)
        .map((f: any) => f.properties.id);

    // Identify which visible trails are MISSING from the local cache
    const missingIds = unclusteredIds.filter((id: string) => !detailedCache.has(id));

    // Only fetch details for missing trails
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

            const dr = await f("/api/v1/search/multi", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
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

    // Convert unclustered hits to lightweight Trail objects for map popups/previews.
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

    return { mapTrails, clusters: clusterFeatureCollection };
}

export async function trails_show(id: string, handle?: string, share?: string, loadGPX?: boolean, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const r = await f(`/api/v1/trail/${id}?` + new URLSearchParams({
        expand: "category,subcategory,subcategory.category,waypoints_via_trail,waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail,summit_logs_via_trail.summit_log_assets_via_summit_log.asset,summit_logs_via_trail.author,trail_share_via_trail.actor,trail_like_via_trail,tags,author,trail_assets_via_trail.asset",
        ...(handle ? { handle } : {}),
        ...(share ? { share } : {})
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail ?? response.data)
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
    return createTrailWithAssets(trail, photos, gpx, f, user);
}

async function createTrailWithAssets(trail: Trail, photos: File[], gpx: File | Blob | null, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response>, user: AuthRecord | undefined) {
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

    const formData = objectToFormData(trail, ["photos", "expand", "_assetLinks", "thumbnail"])

    if (gpx) {
        formData.set("gpx", gpx);
    }

    let r = await f(`/api/v1/trail/form?` + new URLSearchParams({
        expand: "category,subcategory,subcategory.category,trail_assets_via_trail.asset,waypoints_via_trail,waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail,summit_logs_via_trail.summit_log_assets_via_summit_log.asset,trail_share_via_trail,tags",
    }), {
        method: 'PUT',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        const failure = new APIError(r.status, response.message, response.detail);
        const committed = committedAssetRecord<Trail>(failure);
        if (committed) {
            trail.id = committed.id;
            failure.detail = { ...failure.detail, savedTrail: { ...trail, ...committed, photos: [], expand: {} } };
        }
        throw failure;
    }

    let model: Trail = await r.json();
    const save = async () => {
    const existingPhotos = model.photos;
    const thumbnailSelection = snapshotTrailPhotoSelection(trail);

    try {
        try {
            model.photos = await assets_attach_to_target({
                files: photos,
                assetIds: shownAssetLinks(trail._assetLinks, trail.photos),
                onLinked: (assets) => { replaceCopiedAssets(trail, assets); replaceCopiedAssets(thumbnailSelection, assets); },
                target: { trail: model.id },
                existingPhotos: model.photos,
                f,
            });
        } catch (error) {
            rememberPartialTrailAttachments(model, trail, photos, existingPhotos, error, thumbnailSelection);
            throw error;
        }
        const thumbnailAsset = chosenThumbnailAsset(thumbnailSelection, photos, existingPhotos, model.photos);
        trail.id = model.id;
        trail.photos = [...(model.photos ?? [])];
        trail._assetLinks = undefined;
        keepThumbnailOn(trail, thumbnailAsset);
        photos.splice(0);
        model.expand ??= {};

        for (const summitLog of trail.expand?.summit_logs_via_trail ?? []) {
            summitLog.trail = model.id!;
            try {
                const savedLog = await summit_logs_create(summitLog, f, user);
                rememberSavedSummitLog(model, savedLog);
            } catch (error) {
                rememberPartiallySavedSummitLog(model, error);
                throw error;
            }
        }

        for (const wp of trail.expand?.waypoints_via_trail ?? []) {
            wp.trail = model.id!;
            try {
                const savedWaypoint = await waypoints_create(wp, f, user);
                rememberSavedWaypoint(model, wp, savedWaypoint, true);
            } catch (error) {
                rememberPartiallySavedWaypoint(model, wp, error);
                throw error;
            }
        }

        const refreshed = await f(`/api/v1/trail/${model.id}?` + new URLSearchParams({
            expand: "category,trail_assets_via_trail.asset,waypoints_via_trail,waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail,summit_logs_via_trail.summit_log_assets_via_summit_log.asset,trail_share_via_trail,tags",
        }));
        if (refreshed.ok) {
            model = await refreshed.json();
        }
        await assets_set_trail_thumbnail(model.id!, savedPhotoOfAsset(model.photos, thumbnailAsset), f);
        model = await trails_show(model.id!, undefined, undefined, false, f);
    } catch (error) {
        // The trail already exists, even when a later photo or waypoint request
        // fails. Let the editor retry against its ID instead of creating it again.
        const failure = error instanceof APIError
            ? error
            : new APIError(500, error instanceof Error ? error.message : "error-saving-trail");
        failure.detail = { ...failure.detail, savedTrail: model };
        throw failure;
    }

    return model;
    };
    return trail.public ? withFederationSaveBatch("trail", model.id, save, f) : save();
}


function rememberPartialTrailAttachments(baseline: Trail, pending: Trail, files: File[] | undefined, existingPhotos: string[] | undefined, error: unknown, selection: Trail) {
    if (!(error instanceof APIError) || !error.detail?.completedAttachments) return;
    const attachedPhotos: string[] = error.detail.attachedPhotos ?? [];
    const thumbnailAsset = chosenThumbnailAsset(selection, files, existingPhotos, attachedPhotos);
    rememberCompletedTrailAttachments(baseline, pending, files, error.detail.completedAttachments, attachedPhotos);
    keepThumbnailOn(pending, thumbnailAsset);
}

function rememberCompletedTrailAttachments(baseline: Trail, pending: Trail, files: File[] | undefined, completed: CompletedAssetAttachments, attachedPhotos: string[]) {
    // In an update, the attached photos start with the form's photos. One the
    // form links by ID is the saved trail's only once linking completed:
    // listed earlier, dropping it before the retry would send a removal, and
    // that deletes a photo linked by ID without other links.
    const unlinked = new Set(completed.assetIds ? [] : pending._assetLinks);
    if (completed.files) files?.splice(0);
    else if (completed.uploadedFiles) files?.splice(0, completed.uploadedFiles);
    if (completed.assetIds) pending._assetLinks = undefined;
    baseline.photos = [...new Set([
        ...(baseline.photos ?? []),
        ...attachedPhotos.filter((photo) => !unlinked.has(assetIdFromPhotoURL(photo) ?? "")),
    ])];
    pending.photos = [...new Set([...(pending.photos ?? []), ...attachedPhotos])];
}

// Preserve the selected identity before attachment, replacing its source ID
// when an owned copy succeeds.
function snapshotTrailPhotoSelection(trail: Trail): Trail {
    return { ...trail, photos: [...(trail.photos ?? [])], _assetLinks: trail._assetLinks ? [...trail._assetLinks] : undefined, expand: trail.expand ? { ...trail.expand } : undefined };
}

// The editor's thumbnail index counts the photos PhotoPicker shows, then the
// new uploads. The saved trail lists its photos in the order they were linked,
// and a duplicate links its copied photos after the uploads, so the index is
// resolved to an asset. Attaching returns the trail's photos first, then one
// photo per upload in order.
function chosenThumbnailAsset(trail: Trail, files: File[] | undefined, existingPhotos: string[] | undefined, attachedPhotos: string[]): string | undefined {
    const uploaded = attachedPhotos
        .filter((photo) => !existingPhotos?.includes(photo))
        .slice(0, files?.length ?? 0);
    const photo = shownTrailPhotos(trail).concat(uploaded)[trail.thumbnail ?? 0];
    return photo ? assetIdFromPhotoURL(photo) : undefined;
}

// The photos PhotoPicker shows. On save, the editor also adds duplicated
// photos linked by ID to `photos`, but shows them as pending cards until
// they are linked. They are the `_assetLinks` that are not yet
// the trail's assets.
export function shownTrailPhotos(trail: Trail): string[] {
    const trailAssets = new Set(expandedPhotoAssets(trail, "trail").map((asset) => asset.id));
    const picked = new Set(trail._assetLinks?.filter((id) => !trailAssets.has(id)));
    return (trail.photos ?? []).filter((photo) => !picked.has(assetIdFromPhotoURL(photo) ?? ""));
}

// Attaching rewrites `photos`: a new trail lists its uploads first, and
// photos linked by ID become normal photos. The editor keeps this form when a later
// step fails, so a chosen index follows its photo for the next attempt.
function keepThumbnailOn(trail: Trail, assetId: string | undefined) {
    if (trail.thumbnail === undefined || assetId === undefined) return;
    const index = assetId ? shownTrailPhotos(trail).findIndex((photo) => assetIdFromPhotoURL(photo) === assetId) : -1;
    trail.thumbnail = index >= 0 ? index : 0;
}

// Marks only a photo the saved trail has. Otherwise the mark is cleared and
// the trail shows its first photo, rather than the request failing.
function savedPhotoOfAsset(photos: string[] | undefined, assetId: string | undefined): string | undefined {
    return assetId ? photos?.find((photo) => assetIdFromPhotoURL(photo) === assetId) : undefined;
}

function rememberPartiallySavedWaypoint(baseline: Trail, pending: Waypoint, error: unknown) {
    const saved = error instanceof APIError ? error.detail?.savedWaypoint as Waypoint | undefined : undefined;
    if (saved?.id) {
        rememberSavedWaypoint(baseline, pending, saved, false);
        if (error instanceof APIError && error.detail?.remainingAttachments) {
            Object.assign(pending, error.detail.remainingAttachments);
        }
        if (error instanceof APIError && Array.isArray(error.detail?.remainingPhotos)) {
            pending.photos = [...error.detail.remainingPhotos];
        }
    }
}

function rememberSavedWaypoint(baseline: Trail, pending: Waypoint, saved: Waypoint, complete: boolean) {
    const marker = pending.marker;
    if (complete) {
        Object.assign(pending, saved, { marker });
        delete pending._photos;
        delete pending._assetLinks;
    } else {
        // Keep the failed selection available for retry while protecting photos
        // that the server already attached from being treated as removals.
        pending.id = saved.id;
        pending.photos = [...new Set([...(pending.photos ?? []), ...(saved.photos ?? [])])];
    }
    const persisted = complete ? { ...pending } : { ...saved, marker };
    persisted.photos = [...(persisted.photos ?? [])];
    baseline.expand ??= {};
    baseline.expand.waypoints_via_trail = [
        ...(baseline.expand.waypoints_via_trail ?? []).filter((waypoint) => waypoint.id !== saved.id),
        persisted,
    ];
}

function rememberSavedSummitLog(baseline: Trail, saved: SummitLog) {
    baseline.expand ??= {};
    baseline.expand.summit_logs_via_trail = [
        ...(baseline.expand.summit_logs_via_trail ?? []).filter((log) => log.id !== saved.id),
        { ...saved, photos: [...(saved.photos ?? [])], expand: saved.expand ? { ...saved.expand } : undefined },
    ];
}

function rememberPartiallySavedSummitLog(baseline: Trail, error: unknown) {
    const saved = error instanceof APIError ? error.detail?.savedSummitLog as SummitLog | undefined : undefined;
    if (saved?.id) rememberSavedSummitLog(baseline, saved);
}

export async function trails_update(oldTrail: Trail, newTrail: Trail, photos?: File[], gpx?: File | Blob | null, exclude?: (keyof Trail)[], loadGPX = false) {
    const completions: (() => Promise<void>)[] = [];
    const save = async () => {
        try {
            return await updateTrailWithAssets(oldTrail, newTrail, photos, gpx, exclude, loadGPX, (complete) => completions.push(complete));
        } finally {
            for (const complete of completions) await complete();
        }
    };
    return oldTrail.public || (newTrail.public && !exclude?.includes("public"))
        ? withFederationSaveBatch("trail", oldTrail.id, save) : save();
}

async function updateTrailWithAssets(oldTrail: Trail, newTrail: Trail, photos: File[] | undefined, gpx: File | Blob | null | undefined, exclude: (keyof Trail)[] | undefined, loadGPX: boolean, deferComplete: (complete: () => Promise<void>) => void) {
    newTrail.photos ??= oldTrail.photos;
    newTrail.author = oldTrail.author
    const photoRemovals: (() => Promise<void>)[] = [];

    const waypointUpdates = compareObjectArrays<Waypoint>(oldTrail.expand?.waypoints_via_trail ?? [], newTrail.expand?.waypoints_via_trail ?? []);

    for (const addedWaypoint of waypointUpdates.added) {
        addedWaypoint.trail = newTrail.id!
        try {
            const model = await waypoints_create(addedWaypoint, undefined, undefined);
            rememberSavedWaypoint(oldTrail, addedWaypoint, model, true);
        } catch (error) {
            rememberPartiallySavedWaypoint(oldTrail, addedWaypoint, error);
            throw error;
        }
    }

    for (const updatedWaypoint of waypointUpdates.updated) {
        const oldWaypoint = oldTrail.expand?.waypoints_via_trail?.find(w => w.id == updatedWaypoint.id);
        try {
            const previousPhotos = [...(oldWaypoint!.photos ?? [])];
            const model = await waypoints_update(oldWaypoint!, updatedWaypoint, (remove) => {
                photoRemovals.push(async () => {
                    try {
                        await remove();
                        const stored = oldTrail.expand?.waypoints_via_trail?.find(waypoint => waypoint.id === updatedWaypoint.id);
                        if (stored) stored.photos = [...(updatedWaypoint.photos ?? [])];
                    } catch (error) {
                        rememberPartiallySavedWaypoint(oldTrail, updatedWaypoint, error);
                        throw error;
                    }
                });
            });
            rememberSavedWaypoint(oldTrail, updatedWaypoint, model, true);
            const stored = oldTrail.expand?.waypoints_via_trail?.find(waypoint => waypoint.id === model.id);
            if (stored) stored.photos = [...new Set([...previousPhotos, ...(model.photos ?? [])])];
        } catch (error) {
            rememberPartiallySavedWaypoint(oldTrail, updatedWaypoint, error);
            throw error;
        }
    }

    const summitLogUpdates = compareObjectArrays<SummitLog>(oldTrail.expand?.summit_logs_via_trail ?? [], newTrail.expand?.summit_logs_via_trail ?? []);

    for (const summitLog of summitLogUpdates.added) {
        summitLog.trail = newTrail.id!
        try {
            const model = await summit_logs_create(summitLog, undefined, undefined);
            rememberSavedSummitLog(oldTrail, model);
        } catch (error) {
            rememberPartiallySavedSummitLog(oldTrail, error);
            throw error;
        }
    }

    for (const updatedSummitLog of summitLogUpdates.updated) {
        const oldSummitLog = oldTrail.expand?.summit_logs_via_trail?.find(w => w.id == updatedSummitLog.id);

        try {
            const previousPhotos = [...(oldSummitLog!.photos ?? [])];
            const model = await summit_logs_update(oldSummitLog!, updatedSummitLog, (remove) => {
                photoRemovals.push(async () => {
                    try {
                        await remove();
                        const stored = oldTrail.expand?.summit_logs_via_trail?.find(log => log.id === updatedSummitLog.id);
                        if (stored) stored.photos = [...(updatedSummitLog.photos ?? [])];
                    } catch (error) {
                        rememberPartiallySavedSummitLog(oldTrail, error);
                        throw error;
                    }
                });
            }, deferComplete);
            rememberSavedSummitLog(oldTrail, model);
            const stored = oldTrail.expand?.summit_logs_via_trail?.find(log => log.id === model.id);
            if (stored) stored.photos = [...new Set([...previousPhotos, ...(model.photos ?? [])])];
        } catch (error) {
            rememberPartiallySavedSummitLog(oldTrail, error);
            throw error;
        }
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

    const formData = objectToFormData(newTrail, ["expand", "photos", "_assetLinks", "thumbnail", ...(exclude ?? [])])

    if (gpx) {
        formData.append("gpx", gpx);
    }

    const updateUrl = `/api/v1/trail/form/${newTrail.id}?` + new URLSearchParams({
        expand: "category,subcategory,subcategory.category,trail_assets_via_trail.asset,waypoints_via_trail,waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail,summit_logs_via_trail.summit_log_assets_via_summit_log.asset,trail_share_via_trail,tags",
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

    const photoSelectionChanged = !stringArraysEqual(oldTrail.photos, newTrail.photos);
    const assetLinks = shownAssetLinks(newTrail._assetLinks, newTrail.photos);
    const thumbnailSelection = snapshotTrailPhotoSelection(newTrail);
    const shouldRefreshTrail = has_asset_attachments({
        files: photos,
        assetIds: assetLinks,
    });
    try {
        model.photos = await assets_attach_to_target({
            files: photos,
            assetIds: assetLinks,
            onLinked: (assets) => { replaceCopiedAssets(newTrail, assets); replaceCopiedAssets(thumbnailSelection, assets); },
            target: { trail: model.id },
            existingPhotos: newTrail.photos,
        });
    } catch (error) {
        rememberPartialTrailAttachments(oldTrail, newTrail, photos, newTrail.photos, error, thumbnailSelection);
        throw error;
    }
    const thumbnailAsset = chosenThumbnailAsset(thumbnailSelection, photos, newTrail.photos, model.photos);
    // Keep completed uploads consumed even if deleting old photos or refreshing
    // the trail fails afterwards. A retry must only send unfinished selections.
    rememberCompletedTrailAttachments(oldTrail, newTrail, photos, {
        files: true,
        assetIds: true,
    }, model.photos);
    keepThumbnailOn(newTrail, thumbnailAsset);
    // Every surviving target now holds its new links. Only then may a former
    // last link or a removed child be deleted without losing a moved file.
    for (const remove of photoRemovals) await remove();
    await assets_delete_removed(oldTrail.photos, newTrail.photos, { trail: newTrail.id! });
    for (const deletedWaypoint of waypointUpdates.deleted) {
        await waypoints_delete(deletedWaypoint);
        oldTrail.expand!.waypoints_via_trail = oldTrail.expand!.waypoints_via_trail!
            .filter((waypoint) => waypoint.id !== deletedWaypoint.id);
    }

    for (const deletedSummitLog of summitLogUpdates.deleted) {
        await summit_logs_delete(deletedSummitLog);
        oldTrail.expand!.summit_logs_via_trail = oldTrail.expand!.summit_logs_via_trail!
            .filter((log) => log.id !== deletedSummitLog.id);
    }


    const thumbnailChanged = newTrail.thumbnail !== undefined && oldTrail.thumbnail !== newTrail.thumbnail;
    const assetSelectionChanged = photoSelectionChanged || shouldRefreshTrail;
    oldTrail.photos = [...(newTrail.photos ?? [])];
    // From here on a retry sees no photo change, so a mark that is still due
    // must not look as if it had been sent already.
    if (thumbnailChanged || assetSelectionChanged) oldTrail.thumbnail = undefined;
    if (shouldRefreshTrail) {
        model = await trails_show(model.id!, undefined, undefined, false);
    }
    const thumbnailCandidates = model.photos ?? newTrail.photos;
    if (thumbnailCandidates !== undefined && (thumbnailChanged || assetSelectionChanged)) {
        await assets_set_trail_thumbnail(model.id!, savedPhotoOfAsset(thumbnailCandidates, thumbnailAsset));
    }
    if (assetSelectionChanged || thumbnailChanged || photoRemovals.length || waypointUpdates.deleted.length || summitLogUpdates.deleted.length) {
        model = await trails_show(model.id!, undefined, undefined, loadGPX);
    } else if (loadGPX) {
        model.expand ??= {};
        model.expand.gpx_data = await fetchGPX(model);
    }

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

export async function trails_get_bounding_box(f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch): Promise<TrailBoundingBox> {
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
    const categoryById = new Map(get(categories).map((category) => [category.id, category]));
    const subcategoryById = new Map(
        get(subcategories).map((subcategory) => [subcategory.id, subcategory]),
    );

    for (const h of hits) {
        const created = Number(h.created || 0);
        const date = Number(h.date || 0);
        const category = h.category_id ? categoryById.get(h.category_id) : undefined;
        const subcategory = h.subcategory_id
            ? subcategoryById.get(h.subcategory_id)
            : undefined;
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
            category: h.category_id ?? "",
            subcategory: h.subcategory_id ?? "",
            created: new Date(created * 1000).toISOString(),
            date: new Date(date * 1000).toISOString(),
            description: h.description,
            difficulty: h.difficulty == 0 ? "easy" : h.difficulty == 1 ? "moderate" : "difficult",
            distance: h.distance,
            duration: h.duration,
            moving_duration: h.moving_duration,
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
                category: category ?? (h.category
                    ? {
                          id: h.category_id ?? "",
                          name: h.category,
                          icon: h.category_icon,
                      }
                    : undefined),
                subcategory,
                author: {
                    collectionId: "activitypub_actors",
                    is_local: (h.domain?.length ?? 0) == 0,
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

export function buildFilterText(user: AuthRecord, filter: TrailFilter, includeGeo: boolean): string {
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
        const showPublic = filter.public === undefined || filter.public === true;
        const showPrivate = filter.private === undefined || filter.private === true;
        const visibilityParts: string[] = [];
        if (showPublic) visibilityParts.push("public = TRUE");
        if (showPrivate && user?.actor && (!filter.author?.length || filter.author === user.actor)) {
            visibilityParts.push(`(public = FALSE AND author = ${user.actor})`);
        }
        if (filter.shared === true && user?.actor) {
            visibilityParts.push(`shares = ${user.actor}`);
        }
        filterText += ` AND (${visibilityParts.length ? visibilityParts.join(" OR ") : "public = TRUE AND public = FALSE"})`;
        if (filter.shared === false && user?.actor) {
            filterText += ` AND NOT shares = ${user.actor}`;
        }
    }


    if (filter.liked === true) {
        filterText += ` AND likes = ${user?.actor}`
    }

    const startDate = trailFilterDateBoundary(filter.startDate);
    if (startDate !== undefined) {
        filterText += ` AND date >= ${startDate}`
    }

    const endDate = trailFilterDateBoundary(filter.endDate, true);
    if (endDate !== undefined) {
        filterText += ` AND date < ${endDate}`
    }

    const selectedSubcategoryIds = filter.subcategory ?? [];
    if (filter.category.length > 0 || selectedSubcategoryIds.length > 0) {
        const selectedNoSubcategoryCategoryIds = selectedSubcategoryIds
            .map(noSubcategoryFilterCategory)
            .filter((category): category is string => category !== undefined);
        const selectedRealSubcategoryIds = selectedSubcategoryIds.filter(
            (id) => noSubcategoryFilterCategory(id) === undefined,
        );
        const selectedSubcategoryParentIds = new Set(
            get(subcategories)
                .filter((subcategory) => selectedRealSubcategoryIds.includes(subcategory.id))
                .map((subcategory) => subcategory.category),
        );
        for (const categoryId of selectedNoSubcategoryCategoryIds) {
            selectedSubcategoryParentIds.add(categoryId);
        }
        const categoriesWithoutSubcategoryFilter = filter.category.filter(
            (category) => !selectedSubcategoryParentIds.has(category),
        );
        const categoryParts: string[] = [];

        if (categoriesWithoutSubcategoryFilter.length > 0) {
            const categoryValues = categoriesWithoutSubcategoryFilter
                .map(category => `'${category}'`)
                .join(", ");
            categoryParts.push(`category_id IN [${categoryValues}]`);
        }

        if (selectedNoSubcategoryCategoryIds.length > 0) {
            const noSubcategoryParts = selectedNoSubcategoryCategoryIds.map(
                (category) =>
                    `(category_id = '${category}' AND subcategory_id IS NULL)`,
            );
            categoryParts.push(...noSubcategoryParts);
        }

        if (selectedRealSubcategoryIds.length > 0) {
            const subcategoryValues = selectedRealSubcategoryIds
                .map(subcategory => `'${subcategory}'`)
                .join(", ");
            categoryParts.push(`subcategory_id IN [${subcategoryValues}]`);
        }

        if (categoryParts.length > 0) {
            filterText += ` AND (${categoryParts.join(" OR ")})`;
        }
    }

    if (filter.tags.length > 0) {
        filterText += ` AND (${filter.tags.map(t => `tags = '${t}'`).join(" OR ")})`;
    }

    if (filter.completed !== undefined) {
        filterText += ` AND completed = ${filter.completed}`;
    }

    if (includeGeo
        && typeof filter.near.lat === "number" && Number.isFinite(filter.near.lat)
        && Math.abs(filter.near.lat) <= 90
        && typeof filter.near.lon === "number" && Number.isFinite(filter.near.lon)
        && Math.abs(filter.near.lon) <= 180
        && Number.isFinite(filter.near.radius) && filter.near.radius > 0) {
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

function stringArraysEqual(a: string[] | undefined, b: string[] | undefined): boolean {
    return a === b || Boolean(a && b && a.length === b.length && a.every((value, index) => value === b[index]));
}
