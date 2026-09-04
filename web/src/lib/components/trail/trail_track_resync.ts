import GPX from "$lib/models/gpx/gpx";
import type { Trail } from "$lib/models/trail";
import type {
    TrackResyncPreview,
    TrackResyncResult,
} from "$lib/stores/plugin_instance_store";
import { fetchGPX } from "$lib/stores/trail_store";

export type TrailTrackResyncTarget = Readonly<{
    trailId: string;
    trail: Trail;
    preview: TrackResyncPreview;
    retryDeadline: number;
}>;

export type TrailTrackResyncSnapshot = Readonly<{
    key: string;
    trail: Readonly<Trail>;
    preview: Readonly<TrackResyncPreview>;
}>;

export type TrailTrackResyncPatch = TrackResyncResult["track"] & {
    expand?: {
        gpx_data: string;
        gpx: GPX;
    };
};

export function snapshotTrackResyncTarget(
    target: TrailTrackResyncTarget,
): TrailTrackResyncSnapshot {
    const trail = Object.freeze({
        ...target.trail,
        photos: [...target.trail.photos],
        tags: [...target.trail.tags],
        expand: target.trail.expand
            ? Object.freeze({ ...target.trail.expand })
            : undefined,
    });
    const preview = Object.freeze({ ...target.preview });
    const key = JSON.stringify([
        target.trailId,
        preview.provider ?? "",
        preview.externalId ?? "",
    ]);
    return Object.freeze({ key, trail, preview });
}

export async function prepareTrackResyncPatch(
    target: TrailTrackResyncSnapshot,
    track: TrackResyncResult["track"],
): Promise<{ patch: TrailTrackResyncPatch; pageStale: boolean }> {
    let refreshed: { gpx_data: string; gpx: GPX } | undefined;
    let pageStale = false;
    if (target.trail.expand?.gpx_data || target.trail.expand?.gpx) {
        try {
            const gpxData = await fetchGPX(
                { ...target.trail, gpx: track.gpx },
                async (input, init) => {
                    const response = await fetch(input, init);
                    if (!response.ok) throw new Error(`GPX ${response.status}`);
                    return response;
                },
            );
            if (!gpxData.trim()) throw new Error("Empty GPX response");
            refreshed = { gpx_data: gpxData, gpx: GPX.parse(gpxData) };
        } catch (error) {
            console.error(error);
            pageStale = true;
        }
    }

    const patch: TrailTrackResyncPatch = {
        ...track,
        ...(refreshed ? { expand: refreshed } : {}),
    };
    return { patch, pageStale };
}

export function applyTrackResyncPatch(
    trail: Readonly<Trail>,
    patch: TrailTrackResyncPatch,
): Trail {
    return {
        ...trail,
        ...patch,
        expand: patch.expand
            ? { ...trail.expand, ...patch.expand }
            : trail.expand,
    } as Trail;
}
