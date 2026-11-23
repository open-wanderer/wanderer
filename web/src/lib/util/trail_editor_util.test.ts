import { describe, expect, it } from "vitest";
import GPX from "#lib/models/gpx/gpx.ts";
import { Trail } from "#lib/models/trail.ts";
import { Waypoint } from "#lib/models/waypoint.ts";
import { copyTrailEditorValues, initialTrailFormValues } from "./trail_editor_util";

describe("initialTrailFormValues", () => {
    it.each([false, true])("preserves an existing trail's public value of %s", (isPublic) => {
        const saved = new Trail("Saved", { id: "trail0000000001", public: isPublic });

        const form = initialTrailFormValues(saved, { public: !isPublic, category: "" });

        expect(form.public).toBe(isPublic);
        expect(saved.public).toBe(isPublic);
    });

    it.each([false, true])("uses the privacy default of %s for a new trail", (isPublic) => {
        const trail = new Trail("New", { public: !isPublic });

        const form = initialTrailFormValues(trail, { public: isPublic, category: "" });

        expect(form.public).toBe(isPublic);
        expect(trail.public).toBe(!isPublic);
    });

    it.each([
        { category: "existing", fallback: "fallback", expected: "existing" },
        { category: undefined, fallback: "fallback", expected: "fallback" },
        { category: "", fallback: "fallback", expected: "fallback" },
        { category: undefined, fallback: "", expected: "" },
    ])("initializes category $category with fallback $fallback as $expected", ({ category, fallback, expected }) => {
        const trail = new Trail("Trail");
        trail.category = category;

        const form = initialTrailFormValues(trail, { public: false, category: fallback });

        expect(form.category).toBe(expected);
        expect(trail.category).toBe(category);
    });

    it.each([
        [undefined, ""],
        ["", ""],
        ["existing", "existing"],
    ])("normalizes subcategory %s to %s", (subcategory, expected) => {
        const trail = new Trail("Trail");
        trail.subcategory = subcategory;

        const form = initialTrailFormValues(trail, { public: false, category: "" });

        expect(form.subcategory).toBe(expected);
        expect(trail.subcategory).toBe(subcategory);
    });

    it.each([
        [undefined, undefined],
        ["", undefined],
        ["2026-11-23", "2026-11-23"],
        ["2026-11-23 14:30:00.000Z", "2026-11-23"],
        ["2026-11-23T23:30:00-08:00", "2026-11-23"],
    ])("normalizes completion date %s to %s", (completedAt, expected) => {
        const trail = new Trail("Trail", { completed_at: completedAt });

        const form = initialTrailFormValues(trail, { public: false, category: "" });

        expect(form.completed_at).toBe(expected);
        expect(trail.completed_at).toBe(completedAt);
    });

    it("isolates initial photo and waypoint arrays while preserving files and GPX methods", () => {
        const file = new File(["photo"], "photo.jpg");
        const waypoint = new Waypoint(47, 8);
        waypoint.photos = ["waypoint-photo"];
        waypoint._photos = [file];
        const saved = new Trail("Saved", {
            id: "trail0000000001",
            photos: ["first.jpg", "second.jpg"],
            waypoints: [waypoint],
        });
        saved.expand!.gpx = new GPX({});

        const form = initialTrailFormValues(saved, { public: false, category: "" });
        const formWaypoint = form.expand!.waypoints_via_trail![0];
        expect(formWaypoint).toBeInstanceOf(Waypoint);
        expect(formWaypoint._photos).not.toBe(waypoint._photos);
        expect(formWaypoint._photos![0]).toBe(file);

        form.photos.splice(0, 1);
        formWaypoint.photos.splice(0, 1);
        formWaypoint._photos!.splice(0, 1);
        form.expand!.waypoints_via_trail!.splice(0, 1);

        expect(form.photos).toEqual(["second.jpg"]);
        expect(saved.photos).toEqual(["first.jpg", "second.jpg"]);
        expect(saved.expand!.waypoints_via_trail).toEqual([waypoint]);
        expect(waypoint.photos).toEqual(["waypoint-photo"]);
        expect(waypoint._photos).toEqual([file]);
        expect(form.expand!.gpx).toBeInstanceOf(GPX);
        expect(form.expand!.gpx).not.toBe(saved.expand!.gpx);
        expect(typeof form.expand!.gpx!.getTotals).toBe("function");
    });
});

describe("copyTrailEditorValues", () => {
    it("isolates editable arrays while preserving files and GPX methods", () => {
        const file = new File(["photo"], "photo.jpg");
        const waypoint = new Waypoint(47, 8);
        waypoint.photos = ["waypoint-photo"];
        waypoint._photos = [file];
        const saved = new Trail("Saved", { photos: ["trail-photo"], waypoints: [waypoint] });
        saved.expand!.gpx = new GPX({});
        const form = copyTrailEditorValues(saved);

        form.photos.splice(0, 1);
        form.expand!.waypoints_via_trail![0].photos.splice(0, 1);
        form.expand!.waypoints_via_trail!.splice(0, 1);

        expect(saved.photos).toEqual(["trail-photo"]);
        expect(saved.expand!.waypoints_via_trail).toHaveLength(1);
        expect(waypoint.photos).toEqual(["waypoint-photo"]);
        expect(copyTrailEditorValues(waypoint)._photos![0]).toBe(file);
        expect(form.expand!.gpx).toBeInstanceOf(GPX);
        expect(form.expand!.gpx).not.toBe(saved.expand!.gpx);
        expect(typeof form.expand!.gpx!.getTotals).toBe("function");
    });
});
