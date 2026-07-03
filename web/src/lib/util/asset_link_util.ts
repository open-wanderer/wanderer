import type { Asset, AssetLink } from "$lib/models/asset";
import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";

export function assetsFromLinks(links?: AssetLink[]): Asset[] {
    return links
        ?.map((link) => link.expand?.asset)
        .filter((asset): asset is Asset => Boolean(asset)) ?? [];
}

export function assetPhotos(assets?: Asset[]): string[] {
    return assets
        ?.filter((asset) => asset.type === "photo" && (asset.file || (asset.storage_mode && asset.storage_mode !== "copy")))
        .map((asset) =>
            asset.file
                ? `/api/v1/files/${asset.collectionId}/${asset.id}/${asset.file}`
                : `/api/v1/assets/${asset.id}/file`,
        ) ?? [];
}

export function enrichTrailAssetExpands(trail: Trail) {
    if (!trail.expand) {
        trail.expand = {};
    }

    trail.expand.assets_via_trail = linkedAssets(
        trail.expand.assets_via_trail,
        trail.expand.trail_assets_via_trail,
    );
    trail.photos = assetPhotos(trail.expand.assets_via_trail);

    for (const waypoint of trail.expand.waypoints_via_trail ?? []) {
        enrichWaypointAssetExpands(waypoint);
    }

    for (const log of trail.expand.summit_logs_via_trail ?? []) {
        enrichSummitLogAssetExpands(log);
    }
}

export function enrichWaypointAssetExpands(waypoint: Waypoint) {
    if (!waypoint.expand) {
        waypoint.expand = {};
    }
    waypoint.expand.assets_via_waypoint = linkedAssets(
        waypoint.expand.assets_via_waypoint,
        waypoint.expand.waypoint_assets_via_waypoint,
    );
    waypoint.photos = assetPhotos(waypoint.expand.assets_via_waypoint);
}

export function enrichSummitLogAssetExpands(log: SummitLog) {
    if (!log.expand) {
        log.expand = {};
    }
    log.expand.assets_via_summit_log = linkedAssets(
        log.expand.assets_via_summit_log,
        log.expand.summit_log_assets_via_summit_log,
    );
    log.photos = assetPhotos(log.expand.assets_via_summit_log);
}

function linkedAssets(existing?: Asset[], links?: AssetLink[]): Asset[] {
    const linked = assetsFromLinks(links);
    return linked.length || links ? linked : existing ?? [];
}
