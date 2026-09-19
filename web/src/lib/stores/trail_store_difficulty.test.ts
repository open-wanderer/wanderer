import { Trail, type TrailFilter, type TrailSearchResult } from "$lib/models/trail";
import { LngLat } from "maplibre-gl";
import { afterEach, describe, expect, it, vi } from "vitest";
import { searchResultToTrailList, trails_search_bounding_box, trails_search_filter } from "./trail_store";

function searchHit(difficulty: unknown): TrailSearchResult {
    return {
        id: "trail0000000001",
        author: "actor0000000001",
        author_name: "Hiker",
        author_avatar: "",
        name: "Trail",
        description: "",
        location: "",
        distance: 1000,
        elevation_gain: 100,
        elevation_loss: 100,
        duration: 600,
        difficulty: difficulty as TrailSearchResult["difficulty"],
        category: "",
        completed: false,
        date: 0,
        created: 0,
        public: true,
        thumbnail: "",
        like_count: 0,
        gpx: "trail.gpx",
        bounding_box_diagonal: 100,
        _geo: { lat: 47, lng: 8 },
    };
}

function trailFilter(difficulty: TrailFilter["difficulty"]): TrailFilter {
    return {
        q: "",
        category: [],
        subcategory: [],
        tags: [],
        difficulty,
        near: { radius: 2000 },
        distanceMin: 0,
        distanceMax: 100000,
        distanceLimit: 100000,
        elevationGainMin: 0,
        elevationGainMax: 10000,
        elevationGainLimit: 10000,
        elevationLossMin: 0,
        elevationLossMax: 10000,
        elevationLossLimit: 10000,
        sort: "created",
        sortOrder: "-",
    };
}

afterEach(() => vi.unstubAllGlobals());

describe("unknown trail difficulty", () => {
    it("does not invent a difficulty for a new or duplicated trail", () => {
        const original = new Trail("Unrated trail");

        expect(original.difficulty).toBeUndefined();
        expect(Trail.from(original, "actor0000000001").difficulty).toBeUndefined();
    });

    it.each(["easy", "moderate", "difficult"] as const)("preserves an explicit %s rating", (difficulty) => {
        const original = new Trail("Rated trail", { difficulty });

        expect(original.difficulty).toBe(difficulty);
        expect(Trail.from(original, "actor0000000001").difficulty).toBe(difficulty);
    });

    it.each([null, undefined, "", "unknown", "0", "1", false, -1, 3])(
        "does not turn unknown index value %s into a known difficulty",
        async (difficulty) => {
            const [trail] = await searchResultToTrailList([searchHit(difficulty)]);

            expect(trail.difficulty).toBeUndefined();
        },
    );

    it.each([[0, "easy"], [1, "moderate"], [2, "difficult"]] as const)(
        "maps index value %s to %s",
        async (value, difficulty) => {
            const [trail] = await searchResultToTrailList([searchHit(value)]);

            expect(trail.difficulty).toBe(difficulty);
        },
    );

    it.each([
        { selected: [], expected: undefined },
        { selected: [0, 1, 2], expected: undefined },
        { selected: [2, 0, 1], expected: undefined },
        { selected: [0], expected: "difficulty IN [0]" },
        { selected: [1, 2], expected: "difficulty IN [1,2]" },
    ] satisfies { selected: TrailFilter["difficulty"]; expected?: string }[])(
        "only restricts difficulty when a subset is selected: $selected",
        async ({ selected, expected }) => {
            const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) =>
                Response.json({ hits: [], page: 1, totalPages: 0 }),
            );

            await trails_search_filter(trailFilter(selected), 1, 20, request);

            expect(request).toHaveBeenCalledOnce();
            const filter: string = JSON.parse(String(request.mock.calls[0][1]?.body)).options.filter;
            expect(filter).toContain("distance >= 0");
            if (expected) {
                expect(filter).toContain(expected);
                expect(filter).not.toContain("difficulty IS NULL");
            } else {
                expect(filter).not.toContain("difficulty");
            }
        },
    );

    it("keeps an unrated map marker unknown when no details are loaded", async () => {
        const request = vi.fn(async (url: RequestInfo | URL) => {
            if (String(url).endsWith("/cluster")) {
                return Response.json({
                    features: [{
                        geometry: { coordinates: [8, 47] },
                        properties: { id: "unrated-marker", cluster: false, is_large: false },
                    }],
                    totalHits: 1,
                });
            }
            return Response.json({ hits: [], page: 1, totalPages: 0 });
        });
        vi.stubGlobal("fetch", request);

        const result = await trails_search_bounding_box(
            new LngLat(9, 48), new LngLat(7, 46), trailFilter([0, 1, 2]),
        );

        expect(request).toHaveBeenCalledTimes(2);
        expect(result.mapTrails).toHaveLength(1);
        expect(result.mapTrails[0].difficulty).toBeUndefined();
    });
});
