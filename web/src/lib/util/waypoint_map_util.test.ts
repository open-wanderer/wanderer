import { afterEach, describe, expect, it, vi } from "vitest";
import { Waypoint } from "$lib/models/waypoint";
import {
    getWaypointPopupMedia,
    highlightElevationWaypoint,
    scrollToWaypointAnchor,
    waypointAnchorId,
} from "./waypoint_map_util";

describe("waypointAnchorId", () => {
    it("prefixes a waypoint id", () => {
        expect(waypointAnchorId("abc123abc123abc")).toBe(
            "waypoint-abc123abc123abc",
        );
    });

    it("returns undefined when the waypoint has no id", () => {
        expect(waypointAnchorId()).toBeUndefined();
        expect(waypointAnchorId("")).toBeUndefined();
    });
});

describe("getWaypointPopupMedia", () => {
    afterEach(() => {
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
    });

    it("builds file URLs for saved photos", () => {
        const waypoint = new Waypoint(46.5, 11.3, {
            id: "waypointid12345",
            photos: ["summit.jpg", "https://cdn.example/remote.jpg"],
        });
        Object.assign(waypoint, { collectionId: "waypoints" });

        expect(getWaypointPopupMedia(waypoint)).toEqual([
            {
                url: "/api/v1/files/waypoints/waypointid12345/summit.jpg",
                video: false,
            },
            { url: "https://cdn.example/remote.jpg", video: false },
        ]);
    });

    it("marks video files", () => {
        const waypoint = new Waypoint(46.5, 11.3, {
            id: "waypointid12345",
            photos: ["clip.webm"],
        });
        Object.assign(waypoint, { collectionId: "waypoints" });

        expect(getWaypointPopupMedia(waypoint)).toEqual([
            {
                url: "/api/v1/files/waypoints/waypointid12345/clip.webm",
                video: true,
            },
        ]);
    });

    it("includes unsaved local files as object URLs", () => {
        vi.spyOn(URL, "createObjectURL").mockImplementation(
            (file) => `blob:${(file as File).name}`,
        );

        const waypoint = new Waypoint(46.5, 11.3, {
            id: "waypointid12345",
            photos: ["saved.jpg"],
        });
        Object.assign(waypoint, { collectionId: "waypoints" });
        waypoint._photos = [new File(["x"], "new.jpg", { type: "image/jpeg" })];

        expect(getWaypointPopupMedia(waypoint)).toEqual([
            {
                url: "/api/v1/files/waypoints/waypointid12345/saved.jpg",
                video: false,
            },
            { url: "blob:new.jpg", video: false },
        ]);
    });

    it("returns an empty list when the waypoint has no photos", () => {
        expect(getWaypointPopupMedia(new Waypoint(0, 0))).toEqual([]);
    });
});

describe("scrollToWaypointAnchor", () => {
    afterEach(() => {
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
    });

    it("scrolls the matching element into view", () => {
        const scrollIntoView = vi.fn();
        const element = { scrollIntoView } as unknown as HTMLElement;
        vi.stubGlobal("document", {
            getElementById: (id: string) =>
                id === "waypoint-abc123abc123abc" ? element : null,
        });

        expect(scrollToWaypointAnchor("abc123abc123abc")).toBe(element);
        expect(scrollIntoView).toHaveBeenCalledWith({
            behavior: "smooth",
            block: "center",
        });
    });

    it("does nothing when the waypoint has no id", () => {
        const getElementById = vi.fn();
        vi.stubGlobal("document", { getElementById });

        expect(scrollToWaypointAnchor()).toBeNull();
        expect(getElementById).not.toHaveBeenCalled();
    });
});

describe("highlightElevationWaypoint", () => {
    afterEach(() => {
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
    });

    it("marks the matching elevation marker as active", () => {
        const previous = { classList: { remove: vi.fn(), add: vi.fn() } };
        const next = { classList: { remove: vi.fn(), add: vi.fn() } };
        vi.stubGlobal("document", {
            querySelectorAll: () => [previous],
            querySelector: (selector: string) =>
                selector.includes("abc123abc123abc") ? next : null,
        });

        highlightElevationWaypoint("abc123abc123abc");

        expect(previous.classList.remove).toHaveBeenCalledWith("wp-marker-active");
        expect(next.classList.add).toHaveBeenCalledWith("wp-marker-active");
    });
});
