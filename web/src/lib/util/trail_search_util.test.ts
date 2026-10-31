import { describe, expect, it } from "vitest";
import { buildTrailSearchBoundsFilter, isTrailSearchBounds, trailSearchBoundsFromUrl } from "./trail_search_util";

describe("trail search bounds", () => {
    it("intersects route bounds with ordinary and antimeridian map views", () => {
        expect(buildTrailSearchBoundsFilter({ northEast: { lat: 49, lng: 11 }, southWest: { lat: 45, lng: 5 } }))
            .toBe("max_lat >= 45 AND min_lat <= 49 AND max_lon >= 5 AND min_lon <= 11");
        expect(buildTrailSearchBoundsFilter({ northEast: { lat: 20, lng: -170 }, southWest: { lat: -20, lng: 170 } }))
            .toBe("max_lat >= -20 AND min_lat <= 20 AND (max_lon >= 170 OR min_lon <= -170)");
    });
    it("restores map shortcuts with validated ordinary or antimeridian bounds", () => {
        expect(trailSearchBoundsFromUrl(new URLSearchParams("tl_lat=49&tl_lon=11&br_lat=45&br_lon=5")))
            .toEqual({ northEast: { lat: 49, lng: 11 }, southWest: { lat: 45, lng: 5 } });
        expect(isTrailSearchBounds({ northEast: { lat: 20, lng: -170 }, southWest: { lat: -20, lng: 170 } })).toBe(true);
    });
    it.each([
        "tl_lat=49&tl_lon=11&br_lat=45",
        "tl_lat=&tl_lon=11&br_lat=45&br_lon=5",
        "tl_lat=NaN&tl_lon=11&br_lat=45&br_lon=5",
        "tl_lat=91&tl_lon=11&br_lat=45&br_lon=5",
        "tl_lat=49&tl_lon=181&br_lat=45&br_lon=5",
        "tl_lat=40&tl_lon=11&br_lat=45&br_lon=5",
    ])("ignores invalid map shortcuts %s", (query) => {
        expect(trailSearchBoundsFromUrl(new URLSearchParams(query))).toBeUndefined();
    });
});
