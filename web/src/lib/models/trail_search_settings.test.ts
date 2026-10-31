import { describe, expect, it } from "vitest";
import { normalizeTrailSearchFilterSections, trailSearchFilters } from "./trail_search_settings";

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
});
