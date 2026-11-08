import { describe, expect, it } from "vitest";
import GPX from "#lib/models/gpx/gpx.ts";
import { Trail } from "#lib/models/trail.ts";
import { Waypoint } from "#lib/models/waypoint.ts";
import { copyTrailEditorValues } from "./trail_editor_util";

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
