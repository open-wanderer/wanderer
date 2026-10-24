import { describe, expect, it } from "vitest";
import { normalizeTrailSearchFilterSections, trailSearchFilters } from "./trail_search_settings";
import { SettingsCreateSchema } from "./api/settings_schema";

describe("personal search filter placement", () => {
    it.each([undefined, null, false, "invalid"])("uses the standard layout for accounts without preferences (%j)", value => {
        const sections = normalizeTrailSearchFilterSections(value);
        expect(trailSearchFilters.filter(filter => sections[filter.id] === "general").map(filter => filter.id))
            .toEqual(["activity", "location", "distance", "difficulty", "completed"]);
        expect(Object.keys(sections)).toHaveLength(trailSearchFilters.length);
    });
    it("moves individual filters in either direction and defaults newly added filters", () => {
        const sections = normalizeTrailSearchFilterSections({ activity: "advanced", gain: "general", completed: "advanced" });
        expect(sections.activity).toBe("advanced");
        expect(sections.gain).toBe("general");
        expect(sections.completed).toBe("advanced");
        expect(sections.distance).toBe("general");
        expect(sections.date).toBe("advanced");
    });
    it("ignores invalid stored positions and obsolete filter identifiers", () => {
        const sections = normalizeTrailSearchFilterSections({ activity: "hidden", distance: null, unknown: "general" });
        expect(sections.activity).toBe("general");
        expect(sections.distance).toBe("general");
        expect(sections).not.toHaveProperty("unknown");
    });
    it("accepts partial placement settings and retains unrelated map behavior", () => {
        const settings = { behavior: { allowAutoGeolocate: true, mapClusteringMaxZoom: 13, showTrailStartMarker: true,
            searchFilterSections: { activity: "advanced", gain: "general" } } };
        expect(SettingsCreateSchema.parse(settings)).toEqual(settings);
    });
    it.each([
        { activity: "hidden" },
        { unknown: "general" },
        { distance: 1 },
    ])("rejects unsupported values at the settings API (%j)", searchFilterSections => {
        expect(SettingsCreateSchema.safeParse({ behavior: { allowAutoGeolocate: false, searchFilterSections } }).success).toBe(false);
    });
    it("continues accepting older settings without a search layout", () => {
        const settings = { behavior: { allowAutoGeolocate: false, mapClusteringMaxZoom: 11 } };
        expect(SettingsCreateSchema.parse(settings)).toEqual(settings);
    });
});
