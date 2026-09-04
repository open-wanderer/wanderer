import { afterEach, describe, expect, it, vi } from "vitest";
import { Trail } from "$lib/models/trail";
import type { TrackResyncPreview } from "$lib/stores/plugin_instance_store";
import {
    applyTrackResyncPatch,
    prepareTrackResyncPatch,
    snapshotTrackResyncTarget,
    type TrailTrackResyncPatch,
} from "./trail_track_resync";

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("trail track resync helpers", () => {
    it("applies only track fields to the latest trail state", () => {
        const trail = makeTrail();
        trail.name = "Renamed while resyncing";
        trail.description = "Edited description";
        trail.public = true;
        trail.photos = ["new-photo.jpg"];
        trail.tags = ["new-tag"];

        const patch = makeTrackPatch();
        const updated = applyTrackResyncPatch(trail, patch);

        expect(updated).not.toBe(trail);
        expect(updated).toMatchObject(patch);
        expect(updated).toMatchObject({
            name: "Renamed while resyncing",
            description: "Edited description",
            public: true,
            photos: ["new-photo.jpg"],
            tags: ["new-tag"],
        });
        expect(updated.expand).toBe(trail.expand);
    });

    it("snapshots trail and preview data before the input can change", () => {
        const trail = makeTrail();
        trail.photos = ["original-photo.jpg"];
        trail.tags = ["original-tag"];
        trail.expand = {
            ...trail.expand,
            gpx_data: "<gpx>original</gpx>",
        };
        const preview: TrackResyncPreview = {
            available: true,
            provider: "hammerhead",
            externalId: "original-external-id",
        };

        const snapshot = snapshotTrackResyncTarget({
            trailId: trail.id!,
            trail,
            preview,
            retryDeadline: 0,
        });

        trail.name = "Changed later";
        trail.photos.push("later-photo.jpg");
        trail.tags.push("later-tag");
        trail.expand!.gpx_data = "<gpx>changed</gpx>";
        preview.provider = "changed-provider";
        preview.externalId = "changed-external-id";

        expect(snapshot.trail.name).toBe("Original trail");
        expect(snapshot.trail.photos).toEqual(["original-photo.jpg"]);
        expect(snapshot.trail.tags).toEqual(["original-tag"]);
        expect(snapshot.trail.expand?.gpx_data).toBe(
            "<gpx>original</gpx>",
        );
        expect(snapshot.preview.provider).toBe("hammerhead");
        expect(snapshot.preview.externalId).toBe("original-external-id");
        expect(snapshot.key).toBe(
            JSON.stringify([
                trail.id,
                "hammerhead",
                "original-external-id",
            ]),
        );
        expect(Object.isFrozen(snapshot)).toBe(true);
        expect(Object.isFrozen(snapshot.trail)).toBe(true);
        expect(Object.isFrozen(snapshot.preview)).toBe(true);
    });

    it("returns the track patch without fetching when no GPX cache is loaded", async () => {
        const fetchMock = vi.fn(() =>
            Promise.reject(new Error("fetch must not be called")),
        );
        vi.stubGlobal("fetch", fetchMock);
        const snapshot = snapshotTrackResyncTarget({
            trailId: "trail-id",
            trail: makeTrail(),
            preview: {
                available: true,
                provider: "hammerhead",
                externalId: "external-id",
            },
            retryDeadline: 0,
        });
        const track = makeTrackPatch();

        const result = await prepareTrackResyncPatch(snapshot, track);

        expect(fetchMock).not.toHaveBeenCalled();
        expect(result).toEqual({ patch: track, pageStale: false });
        expect(result.patch).not.toHaveProperty("expand");
    });
});

function makeTrail() {
    const trail = new Trail("Original trail", {
        id: "trail-id",
        description: "Original description",
        public: false,
    });
    trail.gpx = "old-track.gpx";
    trail.updated = "2026-09-04T08:00:00Z";
    return trail;
}

function makeTrackPatch(): TrailTrackResyncPatch {
    return {
        gpx: "new-track.gpx",
        distance: 12345,
        elevation_gain: 678,
        elevation_loss: 654,
        duration: 4321,
        lat: 46.8,
        lon: 8.2,
        polyline: "encoded-track",
        min_lat: 46.7,
        max_lat: 46.9,
        min_lon: 8.1,
        max_lon: 8.3,
        bounding_box_diagonal: 20,
        updated: "2026-09-04T09:00:00Z",
    };
}
