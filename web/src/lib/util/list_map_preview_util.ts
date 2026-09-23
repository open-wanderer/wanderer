export type PreviewTrailGeometry = {
    id: string;
    polyline?: string;
    lat?: number;
    lon?: number;
    min_lat?: number;
    max_lat?: number;
    min_lon?: number;
    max_lon?: number;
};

export type PreviewListBounds = {
    min_lat: number;
    max_lat: number;
    min_lon: number;
    max_lon: number;
};

export function unionTrailBounds(
    trails: PreviewTrailGeometry[],
): PreviewListBounds | undefined {
    let minLat = 90;
    let maxLat = -90;
    let minLon = 180;
    let maxLon = -180;
    let ok = false;

    for (const trail of trails) {
        const south = trail.min_lat ?? trail.lat;
        const north = trail.max_lat ?? trail.lat;
        const west = trail.min_lon ?? trail.lon;
        const east = trail.max_lon ?? trail.lon;
        if (
            south == null ||
            north == null ||
            west == null ||
            east == null ||
            (south === 0 && north === 0 && west === 0 && east === 0)
        ) {
            continue;
        }
        minLat = Math.min(minLat, south);
        maxLat = Math.max(maxLat, north);
        minLon = Math.min(minLon, west);
        maxLon = Math.max(maxLon, east);
        ok = true;
    }

    if (!ok) {
        return undefined;
    }

    return {
        min_lat: minLat,
        max_lat: maxLat,
        min_lon: minLon,
        max_lon: maxLon,
    };
}

/** Strip polylines beyond the global budget. Mutates trail objects in place. */
export function applyPolylineBudget(
    trails: PreviewTrailGeometry[],
    maxPolylines: number,
): { trails: PreviewTrailGeometry[]; truncated: boolean } {
    let used = 0;
    let truncated = false;

    for (const trail of trails) {
        if (!trail.polyline) {
            continue;
        }
        if (used >= maxPolylines) {
            trail.polyline = undefined;
            truncated = true;
            continue;
        }
        used += 1;
    }

    return { trails, truncated };
}

export function meiliIdInFilter(ids: string[]): string {
    return `id IN [${ids.map((id) => `'${id.replaceAll("'", "\\'")}'`).join(",")}]`;
}
