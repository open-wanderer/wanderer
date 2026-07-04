import type { Asset, AssetLink } from "$lib/models/asset";
import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";

interface AssetPhotoOptions {
    share?: string;
    origin?: string;
}

export function assetsFromLinks(links?: AssetLink[]): Asset[] {
    return links
        ?.map((link) => link.expand?.asset)
        .filter((asset): asset is Asset => Boolean(asset)) ?? [];
}

export function assetPhotos(assets?: Asset[], options?: AssetPhotoOptions): string[] {
    return assets
        ?.filter((asset) => asset.type === "photo" && (asset.file || (asset.storage_mode && asset.storage_mode !== "copy")))
        .map((asset) =>
            withAssetAccessQuery(asset.file
                ? `/api/v1/files/${asset.collectionId}/${asset.id}/${asset.file}`
                : `/api/v1/assets/${asset.id}/file`, options),
        ) ?? [];
}

export function enrichTrailAssetExpands(trail: Trail, options?: AssetPhotoOptions) {
    if (!trail.expand) {
        trail.expand = {};
    }

    trail.expand.assets_via_trail = linkedAssets(
        trail.expand.assets_via_trail,
        trail.expand.trail_assets_via_trail,
    );
    trail.photos = assetPhotos(trail.expand.assets_via_trail, options);
    const thumbnailIndex = thumbnailPhotoIndex(
        trail.expand.assets_via_trail,
        trail.expand.trail_assets_via_trail,
    );
    if (thumbnailIndex !== undefined) {
        trail.thumbnail = thumbnailIndex;
    }

    for (const waypoint of trail.expand.waypoints_via_trail ?? []) {
        enrichWaypointAssetExpands(waypoint, options);
    }

    for (const log of trail.expand.summit_logs_via_trail ?? []) {
        enrichSummitLogAssetExpands(log, options);
    }
}

export function enrichWaypointAssetExpands(waypoint: Waypoint, options?: AssetPhotoOptions) {
    if (!waypoint.expand) {
        waypoint.expand = {};
    }
    waypoint.expand.assets_via_waypoint = linkedAssets(
        waypoint.expand.assets_via_waypoint,
        waypoint.expand.waypoint_assets_via_waypoint,
    );
    waypoint.photos = assetPhotos(waypoint.expand.assets_via_waypoint, options);
}

export function enrichSummitLogAssetExpands(log: SummitLog, options?: AssetPhotoOptions) {
    if (!log.expand) {
        log.expand = {};
    }
    log.expand.assets_via_summit_log = linkedAssets(
        log.expand.assets_via_summit_log,
        log.expand.summit_log_assets_via_summit_log,
    );
    log.photos = assetPhotos(log.expand.assets_via_summit_log, options);
}

function linkedAssets(existing?: Asset[], links?: AssetLink[]): Asset[] {
    const linked = assetsFromLinks(links);
    return linked.length || links ? linked : existing ?? [];
}

function thumbnailPhotoIndex(assets?: Asset[], links?: AssetLink[]): number | undefined {
    if (!assets?.length || !links?.length) {
        return undefined;
    }
    const thumbnailAssetID = links.find((link) => link.is_thumbnail)?.asset;
    if (!thumbnailAssetID) {
        return undefined;
    }
    const index = assets.findIndex((asset) => asset.id === thumbnailAssetID);
    return index >= 0 ? index : undefined;
}

function withAssetAccessQuery(url: string, options?: AssetPhotoOptions): string {
    url = withOrigin(url, options?.origin);
    if (!options?.share) {
        return url;
    }
    return `${url}${url.includes("?") ? "&" : "?"}${new URLSearchParams({ share: options.share })}`;
}

function withOrigin(url: string, origin?: string): string {
    if (!origin || isAbsoluteURL(url)) {
        return url;
    }
    return new URL(url, origin).toString();
}

function isAbsoluteURL(url: string): boolean {
    try {
        const parsed = new URL(url);
        return parsed.protocol === "http:" || parsed.protocol === "https:";
    } catch {
        return false;
    }
}
