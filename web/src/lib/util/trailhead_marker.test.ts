import { describe, expect, it } from "vitest";
import { trailheadIcon, withTrailheadIcons } from "./trailhead_marker";

describe("trailhead markers", () => {
    const subcategories = [{ id: "via-ferrata", icon: "fa-person-falling" }, { id: "plain", icon: "" }];

    it("prefers the subcategory icon, then the category icon, then a generic icon", () => {
        expect(trailheadIcon({ category_icon: "person-hiking", subcategory_id: "via-ferrata" }, subcategories)).toBe("fa-person-falling");
        expect(trailheadIcon({ category_icon: "person-hiking", subcategory_id: "plain" }, subcategories)).toBe("fa-person-hiking");
        expect(trailheadIcon({ category_icon: "fa-bicycle", subcategory_id: "unknown" }, subcategories)).toBe("fa-bicycle");
        expect(trailheadIcon({}, subcategories)).toBe("fa-shapes");
    });

    it("names a marker image for trailheads only, not for clusters", () => {
        const result = withTrailheadIcons({ type: "FeatureCollection", features: [
            { type: "Feature", properties: { id: "a", category_icon: "person-hiking" }, geometry: { type: "Point", coordinates: [7, 46] } },
            { type: "Feature", properties: { cluster: true, point_count: 4 }, geometry: { type: "Point", coordinates: [8, 47] } },
        ] }, subcategories);
        expect(result.features[0].properties).toMatchObject({ id: "a", marker_icon: "trailhead-fa-person-hiking" });
        expect(result.features[1].properties).not.toHaveProperty("marker_icon");
    });
});
