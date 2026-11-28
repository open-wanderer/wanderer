import type { Asset, AssetLink } from "#lib/models/asset.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import type { Trail } from "#lib/models/trail.ts";
import type { Waypoint } from "#lib/models/waypoint.ts";
import { isURL } from "#lib/util/file_util.ts";

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

    if (trail.expand.assets_via_trail !== undefined || trail.expand.trail_assets_via_trail !== undefined) {
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
    if (waypoint.expand.assets_via_waypoint !== undefined || waypoint.expand.waypoint_assets_via_waypoint !== undefined) {
        waypoint.expand.assets_via_waypoint = linkedAssets(
            waypoint.expand.assets_via_waypoint,
            waypoint.expand.waypoint_assets_via_waypoint,
        );
        waypoint.photos = assetPhotos(waypoint.expand.assets_via_waypoint, options);
    }

}

export function enrichSummitLogAssetExpands(log: SummitLog, options?: AssetPhotoOptions) {
    if (!log.expand) {
        log.expand = {};
    }
    if (log.expand.assets_via_summit_log !== undefined || log.expand.summit_log_assets_via_summit_log !== undefined) {
        log.expand.assets_via_summit_log = linkedAssets(
            log.expand.assets_via_summit_log,
            log.expand.summit_log_assets_via_summit_log,
        );
        log.photos = assetPhotos(log.expand.assets_via_summit_log, options);
    }

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

// Picks the marked photo, else the first one. A generated route
// preview, even a marked one, is only used when the trail has no other photo.
function thumbnailPhotoIndex(assets?: Asset[], links?: AssetLink[]): number | undefined {
    if (!assets?.length || !links?.length) {
        return undefined;
    }
    const markedAssetIDs = new Set(links.filter((link) => link.is_thumbnail).map((link) => link.asset));
    const photos = assets.filter((asset) => isDisplayablePhotoAsset(asset));
    const previews = photos.map((asset) => isGeneratedRoutePreviewAsset(asset));
    for (const preview of [false, true]) {
        const marked = photos.findIndex((asset, i) => previews[i] === preview && markedAssetIDs.has(asset.id));
        const index = marked >= 0 ? marked : previews.indexOf(preview);
        if (index >= 0) {
            return index;
        }
    }
    return undefined;
}

function isDisplayablePhotoAsset(asset: Asset, options?: AssetPhotoOptions): boolean {
    return asset.type === "photo" &&
        (options?.includeGeneratedRoutePreview !== false || !isGeneratedRoutePreviewAsset(asset)) &&
        Boolean(asset.file || (asset.storage_mode && asset.storage_mode !== "copy"));
}

// Generic route_* upload names are ordinary photos. The migration marks
// legacy previews; only the editor's reserved name is recognized at runtime.
export function isGeneratedRoutePreviewAsset(asset?: Asset): boolean {
    if (!asset) {
        return false;
    }
    if (assetMetadataString(asset.metadata, "generated", "kind") === "route-preview") {
        return true;
    }
    const filename = assetFilename(asset).toLowerCase();
    const hasLegacyGeneratedName = /^(?:wanderer-route-preview\.(?:webp|png)|wanderer_route_preview_[a-z0-9]{10}(?:_[a-z0-9]{10})*\.(?:webp|png))$/.test(filename);
    return hasLegacyGeneratedName && !asset.external_provider && !asset.taken_at;
}

function assetFilename(asset: Asset): string {
    return assetMetadataString(asset.metadata, "remote", "filename")
        || assetMetadataString(asset.metadata, "original_filename")
        || asset.file
        || asset.external_id
        || asset.id;
}

function assetMetadataString(metadata: Record<string, unknown> | undefined, ...path: string[]): string {
    let value: unknown = metadata;
    for (const key of path) {
        if (!value || typeof value !== "object") {
            return "";
        }
        value = (value as Record<string, unknown>)[key];
    }
    return typeof value === "string" ? value.trim() : "";
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
