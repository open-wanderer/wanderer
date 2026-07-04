import type { Asset, AssetLink } from "$lib/models/asset";
import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";
import { Collection } from "$lib/util/api_util";
import type { RequestEvent } from "@sveltejs/kit";

type LegacyPhotoCollection = "trails" | "waypoints" | "summit_logs";

interface LegacyPhotoTarget {
    collection: Collection;
    targetField: "trail" | "waypoint" | "summit_log";
}

const legacyPhotoTargets: Record<LegacyPhotoCollection, LegacyPhotoTarget> = {
    trails: { collection: Collection.trail_assets, targetField: "trail" },
    waypoints: { collection: Collection.waypoint_assets, targetField: "waypoint" },
    summit_logs: { collection: Collection.summit_log_assets, targetField: "summit_log" },
};

export async function applyLegacyPhotoNamesForMissingAssetExpands(event: RequestEvent, trail: Trail) {
    if (!trail.expand?.trail_assets_via_trail && !trail.expand?.assets_via_trail?.length && trail.id) {
        const state = await legacyPhotoStateForTarget(event, "trails", trail.id);
        trail.photos = state.photos;
        if (state.thumbnail !== undefined) {
            trail.thumbnail = state.thumbnail;
        }
    }

    for (const waypoint of trail.expand?.waypoints_via_trail ?? []) {
        if (!waypoint.expand?.waypoint_assets_via_waypoint && !waypoint.expand?.assets_via_waypoint?.length && waypoint.id) {
            waypoint.photos = (await legacyPhotoStateForTarget(event, "waypoints", waypoint.id)).photos;
        }
    }

    for (const log of trail.expand?.summit_logs_via_trail ?? []) {
        if (!log.expand?.summit_log_assets_via_summit_log && !log.expand?.assets_via_summit_log?.length && log.id) {
            log.photos = (await legacyPhotoStateForTarget(event, "summit_logs", log.id)).photos;
        }
    }
}

export async function resolveLegacyPhotoFile(event: RequestEvent, collection: string, record: string, file: string): Promise<Response | undefined> {
    if (!isLegacyPhotoCollection(collection)) {
        return undefined;
    }

    const assetId = assetIdFromLegacyPhotoName(file);
    if (!assetId) {
        return undefined;
    }

    const target = legacyPhotoTargets[collection];
    const links = await event.locals.pb.collection(target.collection).getFullList<AssetLink>({
        expand: "asset",
        filter: event.locals.pb.filter(`asset = {:asset} && ${target.targetField} = {:record}`, {
            asset: assetId,
            record,
        }),
        requestKey: null,
    });
    const asset = links[0]?.expand?.asset;
    if (!asset) {
        return undefined;
    }

    return fetchAssetFile(event, asset);
}

async function legacyPhotoStateForTarget(
    event: RequestEvent,
    collection: LegacyPhotoCollection,
    record: string,
): Promise<{ photos: string[]; thumbnail?: number }> {
    const target = legacyPhotoTargets[collection];
    const links = await event.locals.pb.collection(target.collection).getFullList<AssetLink>({
        expand: "asset",
        filter: event.locals.pb.filter(`${target.targetField} = {:record}`, { record }),
        requestKey: null,
        sort: "+created",
    });
    const photos: string[] = [];
    let thumbnail: number | undefined;

    for (const link of links) {
        const asset = link.expand?.asset;
        if (!asset || asset.type !== "photo") {
            continue;
        }
        if (link.is_thumbnail) {
            thumbnail = photos.length;
        }
        photos.push(legacyPhotoName(asset));
    }

    return { photos, thumbnail };
}

function legacyPhotoName(asset: Asset): string {
    return `${asset.id}-${asset.file || "photo.jpg"}`;
}

function assetIdFromLegacyPhotoName(file: string): string | undefined {
    return file.match(/^([a-z0-9]{15})(?:[.-]|$)/)?.[1];
}

async function fetchAssetFile(event: RequestEvent, asset: Asset): Promise<Response> {
    const headers: HeadersInit = {};
    const token = event.locals.pb.authStore.token;
    if (token) {
        headers.Authorization = `Bearer ${token}`;
    }

    const query = new URLSearchParams(Object.fromEntries(event.url.searchParams));
    let url: string;
    if (asset.file) {
        const parts = [
            "api",
            "files",
            encodeURIComponent(asset.collectionId),
            encodeURIComponent(asset.id),
            encodeURIComponent(asset.file),
        ];
        url = event.locals.pb.buildURL(parts.join("/") + (query.size ? `?${query}` : ""));
    } else {
        url = event.locals.pb.buildURL(`assets/${asset.id}/file${query.size ? `?${query}` : ""}`);
    }

    const response = await event.fetch(url, { headers });
    return new Response(response.body, {
        headers: response.headers,
        status: response.status,
    });
}

function isLegacyPhotoCollection(collection: string): collection is LegacyPhotoCollection {
    return collection === "trails" || collection === "waypoints" || collection === "summit_logs";
}
