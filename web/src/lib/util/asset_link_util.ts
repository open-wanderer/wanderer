import type { Asset, AssetLink } from "$lib/models/asset";
import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";
import { isURL } from "$lib/util/file_util";

interface AssetPhotoOptions {
    share?: string;
    origin?: string;
    includeGeneratedRoutePreview?: boolean;
}

export function assetsFromLinks(links?: AssetLink[], options?: AssetPhotoOptions): Asset[] {
    return linkableAssetLinks(links, options)
        ?.map((link) => link.expand?.asset)
        .filter((asset): asset is Asset => Boolean(asset)) ?? [];
}

export function assetPhotos(assets?: Asset[], options?: AssetPhotoOptions): string[] {
    return assets
        ?.filter((asset) => isDisplayablePhotoAsset(asset, options))
        .map((asset) =>
            withAssetAccessQuery(assetPhotoURL(asset), options),
        ) ?? [];
}

export function assetPhotoURL(asset: Asset): string {
    if (asset.file) {
        return `/api/v1/files/${asset.collectionId}/${asset.id}/${asset.file}`;
    }
    return `/api/v1/assets/${asset.id}/file`;
}

export function assetIdFromPhotoURL(photo: string): string | undefined {
    const remoteMatch = photo.match(/\/api\/v1\/assets\/([a-z0-9]{15})\/file/);
    if (remoteMatch) {
        return remoteMatch[1];
    }

    const fileMatch = photo.match(/\/api\/v1\/files\/[^/]+\/([a-z0-9]{15})\//);
    return fileMatch?.[1];
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

export function linkableAssetLinks(links?: AssetLink[], options?: AssetPhotoOptions): AssetLink[] {
    if (options?.includeGeneratedRoutePreview !== false) {
        return links ?? [];
    }
    return links
        ?.filter((link) => !isGeneratedRoutePreviewAsset(link.expand?.asset)) ?? [];
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
    const index = assets
        .filter((asset) => isDisplayablePhotoAsset(asset))
        .findIndex((asset) => asset.id === thumbnailAssetID);
    return index >= 0 ? index : undefined;
}

function isDisplayablePhotoAsset(asset: Asset, options?: AssetPhotoOptions): boolean {
    return asset.type === "photo" &&
        (options?.includeGeneratedRoutePreview !== false || !isGeneratedRoutePreviewAsset(asset)) &&
        Boolean(asset.file || (asset.storage_mode && asset.storage_mode !== "copy"));
}

export function isGeneratedRoutePreviewAsset(asset?: Asset): boolean {
    const generated = asset?.metadata?.generated as { kind?: string } | undefined;
    return generated?.kind === "route-preview";
}

function withAssetAccessQuery(url: string, options?: AssetPhotoOptions): string {
    url = withOrigin(url, options?.origin);
    if (!options?.share) {
        return url;
    }
    return `${url}${url.includes("?") ? "&" : "?"}${new URLSearchParams({ share: options.share })}`;
}

function withOrigin(url: string, origin?: string): string {
    if (!origin || isURL(url)) {
        return url;
    }
    return new URL(url, origin).toString();
}
