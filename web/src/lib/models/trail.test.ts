import { describe, expect, it } from "vitest";
import { SummitLog } from "./summit_log";
import {
    Trail,
    defaultTrailDuplicateOptions,
    hasDuplicatePhotos,
    normalizeTrailDuplicateOptions,
} from "./trail";
import { Waypoint } from "./waypoint";

describe("Trail.from", () => {
    it("duplicates route and metadata without photos or summit logs", () => {
        const duplicateActor = "targetactor1234";
        const original = originalTrail();

        const duplicate = Trail.from(original, duplicateActor);

        expect(duplicate.name).toBe(original.name);
        expect(duplicate.description).toBe(original.description);
        expect(duplicate.location).toBe(original.location);
        expect(duplicate.distance).toBe(original.distance);
        expect(duplicate.duration).toBe(original.duration);
        expect(duplicate.elevation_gain).toBe(original.elevation_gain);
        expect(duplicate.elevation_loss).toBe(original.elevation_loss);
        expect(duplicate.expand?.gpx_data).toBe(original.expand?.gpx_data);
        expect(duplicate.photos).toEqual([]);
        expect(duplicate.expand?.summit_logs_via_trail).toEqual([]);

        expect(duplicate.expand?.waypoints_via_trail).toHaveLength(1);
        const waypoint = original.expand!.waypoints_via_trail![0];
        const duplicatedWaypoint = duplicate.expand!.waypoints_via_trail![0];
        expect(duplicatedWaypoint).toMatchObject({
            name: waypoint.name,
            description: waypoint.description,
            lat: waypoint.lat,
            lon: waypoint.lon,
            distance_from_start: waypoint.distance_from_start,
            author: duplicateActor,
            photos: [],
        });
    });

    it("honors optional waypoints and summit logs while leaving photos for the file clone step", () => {
        const duplicateActor = "targetactor1234";
        const original = originalTrail();

        const duplicate = Trail.from(original, duplicateActor, {
            waypoints: false,
            summitLogs: true,
            trailPhotos: true,
            waypointPhotos: true,
            summitLogPhotos: true,
        });

        expect(duplicate.photos).toEqual([]);
        expect(duplicate.expand?.waypoints_via_trail).toEqual([]);
        expect(duplicate.expand?.summit_logs_via_trail).toHaveLength(1);

        const summitLog = original.expand!.summit_logs_via_trail![0];
        const duplicatedSummitLog = duplicate.expand!.summit_logs_via_trail![0];
        expect(duplicatedSummitLog).toMatchObject({
            text: summitLog.text,
            date: summitLog.date,
            distance: summitLog.distance,
            elevation_gain: summitLog.elevation_gain,
            elevation_loss: summitLog.elevation_loss,
            duration: summitLog.duration,
            author: duplicateActor,
            photos: [],
        });
        expect(duplicatedSummitLog.expand?.gpx_data).toBe(summitLog.expand?.gpx_data);
    });

    it("normalizes duplicate photo options that depend on copied records", () => {
        const options = normalizeTrailDuplicateOptions({
            waypoints: false,
            summitLogs: false,
            trailPhotos: true,
            waypointPhotos: true,
            summitLogPhotos: true,
        });

        expect(options).toEqual({
            ...defaultTrailDuplicateOptions,
            waypoints: false,
            trailPhotos: true,
            waypointPhotos: false,
            summitLogPhotos: false,
        });
        expect(hasDuplicatePhotos(options)).toBe(true);
    });
});

function originalTrail() {
    const sourceActor = "sourceactor1234";

    const waypoint = new Waypoint(46.1, 7.2, {
        id: "waypoint0000001",
        name: "Viewpoint",
        description: "Look left.",
        icon: "circle",
        photos: ["waypoint-photo.jpg"],
    });
    waypoint.author = sourceActor;
    waypoint.distance_from_start = 1200;
    Object.assign(waypoint, {
        collectionId: "waypoints",
        collectionName: "waypoints",
    });

    const summitLog = new SummitLog("2025-06-14", {
        id: "summitlog000001",
        text: "Great day out.",
        distance: 11,
        elevation_gain: 900,
        elevation_loss: 880,
        duration: 7200,
        photos: ["summit-log-photo.jpg"],
    });
    summitLog.author = sourceActor;
    summitLog.expand = { gpx_data: "<gpx>summit-log-route</gpx>" };

    const original = new Trail("Ridge walk", {
        date: "2025-06-14",
        description: "A readable trail from another user.",
        difficulty: "difficult",
        completed: true,
        distance: 12.5,
        duration: 7800,
        elevation_gain: 980,
        elevation_loss: 970,
        thumbnail: 1,
        lat: 46.2,
        lon: 7.3,
        location: "Valais",
        public: true,
        photos: ["trail-photo.jpg"],
        tags: [{ id: "tag000000000001", name: "alpine" } as any],
        category: { id: "category0000001", name: "Hiking" },
        subcategory: {
            id: "subcategory0001",
            category: "category0000001",
            name: "Ridge",
        },
        gpx_data: "<gpx>trail-route</gpx>",
        waypoints: [waypoint],
        summit_logs: [summitLog],
        bounding_box_diagonal: 42,
    });
    original.author = sourceActor;

    return original;
}
