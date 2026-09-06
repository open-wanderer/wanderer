import { describe, expect, it } from "vitest";
import {
    applyPolylineBudget,
    meiliIdInFilter,
    unionTrailBounds,
} from "./list_map_preview_util";

describe("unionTrailBounds", () => {
    it("unions trail bounds and falls back to lat/lon", () => {
        expect(
            unionTrailBounds([
                {
                    id: "a",
                    min_lat: 10,
                    max_lat: 12,
                    min_lon: 20,
                    max_lon: 22,
                },
                { id: "b", lat: 15, lon: 25 },
            ]),
        ).toEqual({
            min_lat: 10,
            max_lat: 15,
            min_lon: 20,
            max_lon: 25,
        });
    });

    it("returns undefined when no geometry is present", () => {
        expect(unionTrailBounds([{ id: "a" }])).toBeUndefined();
    });
});

describe("applyPolylineBudget", () => {
    it("keeps only the first N polylines", () => {
        const trails = [
            { id: "1", polyline: "aaa" },
            { id: "2", polyline: "bbb" },
            { id: "3", lat: 1, lon: 2 },
            { id: "4", polyline: "ccc" },
        ];

        const result = applyPolylineBudget(trails, 2);

        expect(result.truncated).toBe(true);
        expect(result.trails.map((t) => t.polyline)).toEqual([
            "aaa",
            "bbb",
            undefined,
            undefined,
        ]);
        expect(result.trails[2].lat).toBe(1);
    });

    it("does not truncate when under budget", () => {
        const trails = [{ id: "1", polyline: "aaa" }];
        expect(applyPolylineBudget(trails, 2)).toEqual({
            trails,
            truncated: false,
        });
    });
});

describe("meiliIdInFilter", () => {
    it("builds an id IN filter", () => {
        expect(meiliIdInFilter(["abc", "d'ef"])).toBe(
            "id IN ['abc','d\\'ef']",
        );
    });
});
