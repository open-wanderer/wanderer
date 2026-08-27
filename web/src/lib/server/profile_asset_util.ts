import type { FeedItem } from "$lib/models/feed";
import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import { enrichSummitLogAssetExpands, enrichTrailAssetExpands } from "$lib/util/asset_link_util";
import { isURL } from "$lib/util/file_util";

export function enrichFeedItemAssetPhotos(feedItems: FeedItem[], origin?: string) {
    for (const feedItem of feedItems) {
        const item = feedItem.expand?.item;
        if (!item) {
            continue;
        }

        if (feedItem.type === "trail") {
            const trail = item as Trail;
            const fallbackPhotos = normalizePhotoURLs(trail.photos, origin, "trails", feedItem.item);

            if (hasTrailAssetExpand(trail)) {
                enrichTrailAssetExpands(trail, { origin });
            } else {
                trail.photos = fallbackPhotos;
            }
        }

        if (feedItem.type === "summit_log") {
            const log = item as SummitLog;
            const fallbackPhotos = normalizePhotoURLs(log.photos, origin, "summit_logs", feedItem.item);

            if (hasSummitLogAssetExpand(log)) {
                enrichSummitLogAssetExpands(log, { origin });
            } else {
                log.photos = fallbackPhotos;
            }
        }
    }
}

export function enrichSummitLogAssetPhotos(logs: SummitLog[], origin?: string) {
    for (const log of logs) {
        const fallbackPhotos = normalizePhotoURLs(log.photos, origin, "summit_logs", log.id ?? "");

        if (hasSummitLogAssetExpand(log)) {
            enrichSummitLogAssetExpands(log, { origin });
        } else {
            log.photos = fallbackPhotos;
        }
    }
}

export function withRequiredExpand<T extends { expand?: string }>(options: T, required: string[]): T {
    const expand = Array.from(new Set([
        ...(options.expand?.split(",").map((value) => value.trim()).filter(Boolean) ?? []),
        ...required,
    ])).join(",");

    return { ...options, expand };
}

function hasTrailAssetExpand(trail: Trail): boolean {
    return Boolean(
        trail.expand
        && ("trail_assets_via_trail" in trail.expand || "assets_via_trail" in trail.expand),
    );
}

function hasSummitLogAssetExpand(log: SummitLog): boolean {
    return Boolean(
        log.expand
        && ("summit_log_assets_via_summit_log" in log.expand || "assets_via_summit_log" in log.expand),
    );
}

function normalizePhotoURLs(photos: string[] | undefined, origin: string | undefined, collection: string, recordId: string): string[] {
    return (photos ?? []).map((photo) => normalizePhotoURL(photo, origin, collection, recordId));
}

function normalizePhotoURL(photo: string, origin: string | undefined, collection: string, recordId: string): string {
    if (!origin || isURL(photo)) {
        return photo;
    }
    if (photo.startsWith("/")) {
        return new URL(photo, origin).toString();
    }
    return new URL(`/api/v1/files/${collection}/${recordId}/${photo}`, origin).toString();
}
