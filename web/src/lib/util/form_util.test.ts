import { describe, expect, it } from "vitest";
import { invalidFields } from "./form_util";

describe("invalidFields", () => {
    it("finds the fields with an error", () => {
        expect(invalidFields({ name: ["Required"], gpx: null, tags: [] })).toEqual(["name"]);
    });

    it("walks nested fields", () => {
        expect(invalidFields({ expand: { gpx_data: ["Invalid"], tags: null } })).toEqual(["expand.gpx_data"]);
    });

    it("walks arrays of objects", () => {
        expect(invalidFields({ waypoints: [{ lat: null }, { lat: ["Invalid"] }] })).toEqual(["waypoints.1.lat"]);
    });

    it("is empty without errors", () => {
        expect(invalidFields({ name: null })).toEqual([]);
    });
});
