export type TrailSearchBounds = {
    northEast: { lat: number; lng: number };
    southWest: { lat: number; lng: number };
};

export function isTrailSearchBounds(value: unknown): value is TrailSearchBounds {
    if (!value || typeof value !== "object") return false;
    const { northEast, southWest } = value as TrailSearchBounds;
    const valid = (point: TrailSearchBounds["northEast"]) => point
        && typeof point.lat === "number" && Number.isFinite(point.lat) && Math.abs(point.lat) <= 90
        && typeof point.lng === "number" && Number.isFinite(point.lng) && Math.abs(point.lng) <= 180;
    return Boolean(valid(northEast) && valid(southWest) && northEast.lat >= southWest.lat);
}

export function trailSearchBoundsFromUrl(params: URLSearchParams): TrailSearchBounds | undefined {
    const keys = ["tl_lat", "tl_lon", "br_lat", "br_lon"];
    if (!keys.every(key => params.has(key) && params.get(key)!.trim())) return;
    const bounds = {
        northEast: { lat: Number(params.get("tl_lat")), lng: Number(params.get("tl_lon")) },
        southWest: { lat: Number(params.get("br_lat")), lng: Number(params.get("br_lon")) },
    };
    return isTrailSearchBounds(bounds) ? bounds : undefined;
}

/** Match routes intersecting the visible map, including antimeridian views. */
export function buildTrailSearchBoundsFilter({ northEast, southWest }: TrailSearchBounds): string {
    const longitude = southWest.lng > northEast.lng
        ? `(max_lon >= ${southWest.lng} OR min_lon <= ${northEast.lng})`
        : `max_lon >= ${southWest.lng} AND min_lon <= ${northEast.lng}`;
    return `max_lat >= ${southWest.lat} AND min_lat <= ${northEast.lat} AND ${longitude}`;
}
