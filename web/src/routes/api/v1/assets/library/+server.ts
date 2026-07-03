import type { Asset } from "$lib/models/asset";
import GPX from "$lib/models/gpx/gpx";
import { RecordIdValueSchema } from "$lib/models/api/base_schema";
import { Collection, handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const LibraryRequestSchema = z.object({
    trailId: RecordIdValueSchema.optional(),
    waypointId: RecordIdValueSchema.optional(),
    summitLogId: RecordIdValueSchema.optional(),
    lat: z.number().optional(),
    lon: z.number().optional(),
    trailData: z.string().optional(),
    takenAfter: z.string().datetime().optional(),
    takenBefore: z.string().datetime().optional(),
    doubleRadius: z.boolean().optional(),
});

interface LibraryCandidate {
    source: "wanderer";
    providerId: string;
    externalProvider?: string;
    externalId?: string;
    assetId: string;
    originalFileName: string;
    takenAt: string;
    lat: number;
    lon: number;
    pointLat?: number;
    pointLon?: number;
    distance: number;
    distanceFromStart: number;
    city: string;
    country: string;
    thumbnailUrl: string;
}

interface ExternalAssetRef {
    provider: string;
    id: string;
}

export async function POST(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }
        if (!event.locals.user.actor) {
            return json({ message: "Actor not found" }, { status: 401 });
        }

        const data = LibraryRequestSchema.parse(await event.request.json().catch(() => ({})));
        const linkedAssetIds = await linkedAssetIdsForTarget(event, data);
        const trailContext = trailContextFromTrailData(data.trailData);
        const trackPoints = trailContext.points;
        const timeWindow = requestTimeWindow(data);
        const assetFilters = ["author = {:author}", "type = 'photo'"];
        const assetFilterParams: Record<string, string> = { author: event.locals.user.actor };
        if (timeWindow?.start) {
            assetFilters.push("taken_at >= {:takenAfter}");
            assetFilterParams.takenAfter = timeWindow.start.toISOString();
        }
        if (timeWindow?.end) {
            assetFilters.push("taken_at <= {:takenBefore}");
            assetFilterParams.takenBefore = timeWindow.end.toISOString();
        }
        const assets = await event.locals.pb.collection(Collection.assets).getFullList<Asset>({
            filter: event.locals.pb.filter(assetFilters.join(" && "), assetFilterParams),
            sort: "-taken_at,-created",
            requestKey: null,
        });

        const maxDistance = (data.lat !== undefined && data.lon !== undefined) || trackPoints.length > 0
            ? (data.doubleRadius ? 2000 : 1000)
            : Number.POSITIVE_INFINITY;
        const candidatePool = assets
            .filter((asset) => asset.file || (asset.storage_mode && asset.storage_mode !== "copy"))
            .filter((asset) => !isGeneratedRoutePreviewAsset(asset))
            .filter((asset) => assetWithinTimeWindow(asset, timeWindow))
            .filter((asset) => asset.lat !== undefined && asset.lon !== undefined)
            .map((asset) => toCandidate(asset, data.lat, data.lon, trackPoints))
            .filter((candidate) => candidate.distance <= maxDistance);
        const existingExternalRefs = candidatePool
            .map(externalAssetRef)
            .filter((ref): ref is ExternalAssetRef => Boolean(ref));
        const candidates = candidatePool
            .filter((candidate) => !linkedAssetIds.has(candidate.assetId))
            .sort((a, b) => {
                if (trackPoints.length && a.distanceFromStart !== b.distanceFromStart) {
                    return a.distanceFromStart - b.distanceFromStart;
                }
                if (a.distance !== b.distance) {
                    return a.distance - b.distance;
                }
                return Date.parse(b.takenAt || "0") - Date.parse(a.takenAt || "0");
            });

        return json({
            hasTimestamps: candidates.some((candidate) => Boolean(candidate.takenAt)),
            candidates,
            existingExternalRefs,
            hasMore: false,
            takenAfter: "",
        });
    } catch (e: any) {
        return handleError(e);
    }
}

function externalAssetRef(candidate: LibraryCandidate): ExternalAssetRef | undefined {
    if (!candidate.externalProvider || !candidate.externalId) {
        return;
    }
    return {
        provider: candidate.externalProvider,
        id: candidate.externalId,
    };
}

async function linkedAssetIdsForTarget(
    event: RequestEvent,
    data: z.infer<typeof LibraryRequestSchema>,
): Promise<Set<string>> {
    const linked = new Set<string>();
    const targets = [
        { collection: Collection.trail_assets, field: "trail", id: data.trailId },
        { collection: Collection.waypoint_assets, field: "waypoint", id: data.waypointId },
        { collection: Collection.summit_log_assets, field: "summit_log", id: data.summitLogId },
    ];

    for (const target of targets) {
        if (!target.id) {
            continue;
        }
        const links = await event.locals.pb.collection(target.collection).getFullList<{ asset: string }>({
            filter: event.locals.pb.filter(`${target.field} = {:target}`, {
                target: target.id,
            }),
            requestKey: null,
        });
        for (const link of links) {
            linked.add(link.asset);
        }
    }
    return linked;
}

interface TrackPoint {
    lat: number;
    lon: number;
    distance: number;
}

interface TrailContext {
    points: TrackPoint[];
}

interface TimeWindow {
    start?: Date;
    end?: Date;
}

function toCandidate(asset: Asset, lat?: number, lon?: number, trackPoints: TrackPoint[] = []): LibraryCandidate {
    const assetLat = Number(asset.lat ?? 0);
    const assetLon = Number(asset.lon ?? 0);
    const nearest = trackPoints.length ? nearestTrackPoint(trackPoints, assetLat, assetLon) : undefined;
    const hasDistance = nearest !== undefined || (lat !== undefined && lon !== undefined && asset.lat !== undefined && asset.lon !== undefined);
    return {
        source: "wanderer",
        providerId: asset.external_provider || "wanderer",
        externalProvider: asset.external_provider,
        externalId: asset.external_id,
        assetId: asset.id,
        originalFileName: assetFilename(asset),
        takenAt: asset.taken_at ?? asset.created ?? "",
        lat: assetLat,
        lon: assetLon,
        pointLat: nearest?.lat,
        pointLon: nearest?.lon,
        distance: nearest ? nearest.distanceToPhoto : (hasDistance ? haversineMeters(lat!, lon!, assetLat, assetLon) : 0),
        distanceFromStart: nearest?.distanceFromStart ?? 0,
        city: "",
        country: "",
        thumbnailUrl: assetThumbnailUrl(asset),
    };
}

function assetFilename(asset: Asset): string {
    const remote = asset.metadata?.remote as { filename?: string } | undefined;
    return asset.file || remote?.filename || asset.external_id || asset.id;
}

function assetThumbnailUrl(asset: Asset): string {
    if (asset.file) {
        return `/api/v1/files/${asset.collectionId}/${asset.id}/${asset.file}`;
    }
    return `/api/v1/assets/${asset.id}/file`;
}

function isGeneratedRoutePreviewAsset(asset: Asset): boolean {
    const generated = asset.metadata?.generated as { kind?: string } | undefined;
    if (generated?.kind === "route-preview") {
        return true;
    }

    const filename = assetFilename(asset).toLowerCase();
    const hasLegacyGeneratedName =
        /^route_[a-z0-9]{8,}\.(webp|png|jpe?g)$/.test(filename) ||
        filename.startsWith("wanderer-route-preview");
    return hasLegacyGeneratedName && !asset.external_provider && !asset.taken_at;
}

function haversineMeters(lat1: number, lon1: number, lat2: number, lon2: number): number {
    const radius = 6371000;
    const toRad = Math.PI / 180;
    const dLat = (lat2 - lat1) * toRad;
    const dLon = (lon2 - lon1) * toRad;
    const a =
        Math.sin(dLat / 2) * Math.sin(dLat / 2) +
        Math.cos(lat1 * toRad) * Math.cos(lat2 * toRad) *
        Math.sin(dLon / 2) * Math.sin(dLon / 2);
    return 2 * radius * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function trailContextFromTrailData(trailData?: string): TrailContext {
    if (!trailData) {
        return { points: [] };
    }
    try {
        const gpx = GPX.parse(trailData);
        const points: TrackPoint[] = [];
        let distance = 0;
        let previous: TrackPoint | undefined;
        for (const track of gpx.trk ?? []) {
            for (const segment of track.trkseg ?? []) {
                previous = undefined;
                for (const point of segment.trkpt ?? []) {
                    const lat = Number(point.$.lat);
                    const lon = Number(point.$.lon);
                    if (!Number.isFinite(lat) || !Number.isFinite(lon)) {
                        continue;
                    }
                    if (previous) {
                        distance += haversineMeters(previous.lat, previous.lon, lat, lon);
                    }
                    const current = { lat, lon, distance };
                    points.push(current);
                    previous = current;
                }
            }
        }
        return {
            points: decimateTrackPoints(points, 2000),
        };
    } catch (e) {
        console.warn("Unable to parse trail data for asset library", e);
        return { points: [] };
    }
}

function requestTimeWindow(data: z.infer<typeof LibraryRequestSchema>): TimeWindow | undefined {
    const start = parseRequestDate(data.takenAfter);
    const end = parseRequestDate(data.takenBefore);
    if (!start && !end) {
        return undefined;
    }
    return { start, end };
}

function parseRequestDate(value?: string): Date | undefined {
    if (!value) {
        return undefined;
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return undefined;
    }
    return date;
}

function assetWithinTimeWindow(asset: Asset, window: TimeWindow | undefined): boolean {
    if (!window) {
        return true;
    }
    if (!asset.taken_at) {
        return false;
    }
    const takenAt = new Date(asset.taken_at);
    if (Number.isNaN(takenAt.getTime())) {
        return false;
    }
    if (window.start && takenAt < window.start) {
        return false;
    }
    if (window.end && takenAt > window.end) {
        return false;
    }
    return true;
}

function decimateTrackPoints(points: TrackPoint[], limit: number): TrackPoint[] {
    if (limit <= 0 || points.length <= limit) {
        return points;
    }
    const step = Math.ceil(points.length / limit);
    return points.filter((_, index) => index % step === 0);
}

function nearestTrackPoint(points: TrackPoint[], lat: number, lon: number) {
    let best: { lat: number; lon: number; distanceFromStart: number; distanceToPhoto: number } | undefined;
    for (const point of points) {
        const distanceToPhoto = haversineMeters(point.lat, point.lon, lat, lon);
        if (!best || distanceToPhoto < best.distanceToPhoto) {
            best = {
                lat: point.lat,
                lon: point.lon,
                distanceFromStart: point.distance,
                distanceToPhoto,
            };
        }
    }
    return best;
}
