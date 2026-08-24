import type { List } from "$lib/models/list";
import type { Trail } from "$lib/models/trail";
import { enrichTrailAssetExpands } from "$lib/util/asset_link_util";

interface TrailResponseOptions {
    share?: string;
}

const trailAssetExpands = [
    "trail_assets_via_trail.asset",
    "waypoints_via_trail.waypoint_assets_via_waypoint.asset",
    "summit_logs_via_trail.summit_log_assets_via_summit_log.asset",
];

const listTrailAssetExpands = trailAssetExpands.map((expand) => `trails.${expand}`);

export function withTrailAssetExpands<T extends { expand?: string }>(options: T): T {
    return withRequiredExpands(options, trailAssetExpands);
}

export function withTrailAssetExpandParams(params: URLSearchParams): URLSearchParams {
    return withRequiredExpandParams(params, trailAssetExpands);
}

export function withListTrailAssetExpandParams(params: URLSearchParams): URLSearchParams {
    return withRequiredExpandParams(params, listTrailAssetExpands);
}

export function enrichTrailResponse(trail: Trail, options?: TrailResponseOptions): Trail {
    trail.date = trail.date?.substring(0, 10) ?? "";
    for (const log of trail.expand?.summit_logs_via_trail ?? []) {
        log.date = log.date.substring(0, 10);
    }
    trail.expand?.waypoints_via_trail?.sort((a, b) => (a.distance_from_start ?? 0) - (b.distance_from_start ?? 0));
    enrichTrailAssetExpands(trail, options);
    return trail;
}

export function enrichTrailListResponse<T extends { items: Trail[] }>(response: T, options?: TrailResponseOptions): T {
    for (const trail of response.items) {
        enrichTrailResponse(trail, options);
    }
    return response;
}

export function enrichListTrailResponse(list: List, options?: TrailResponseOptions): List {
    for (const trail of list.expand?.trails ?? []) {
        enrichTrailResponse(trail, options);
    }
    return list;
}

function withRequiredExpands<T extends { expand?: string }>(options: T, required: string[]): T {
    return { ...options, expand: mergedExpand(options.expand, required) };
}

function withRequiredExpandParams(params: URLSearchParams, required: string[]): URLSearchParams {
    const next = new URLSearchParams(params);
    next.set("expand", mergedExpand(next.get("expand") ?? undefined, required));
    return next;
}

function mergedExpand(current: string | undefined, required: string[]): string {
    return Array.from(new Set([
        ...(current?.split(",").map((value) => value.trim()).filter(Boolean) ?? []),
        ...required,
    ])).join(",");
}
