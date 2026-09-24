import type { TrailFilter } from "$lib/models/trail";
import { describe, expect, it, vi } from "vitest";
import { trails_search_filter } from "./trail_store";

async function requestedFilter(near: TrailFilter["near"]): Promise<string> {
    const filter: TrailFilter = {
        q: "",
        category: [],
        subcategory: [],
        tags: [],
        difficulty: [],
        near,
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
    const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) =>
        Response.json({ hits: [], page: 1, totalPages: 0 }),
    );

    await trails_search_filter(filter, 1, 20, request);

    expect(request).toHaveBeenCalledOnce();
    const [url, options] = request.mock.calls[0];
    expect(url).toBe("/api/v1/search/trails");
    expect(options?.method).toBe("POST");
    return JSON.parse(String(options?.body)).options.filter;
}

describe("trail search radius", () => {
    it.each([
        { lat: 47, lon: 8, radius: 2000 },
        { lat: 0, lon: 8, radius: 2000 },
        { lat: 47, lon: 0, radius: 2000 },
        { lat: 0, lon: 0, radius: 2000 },
        { lat: -47, lon: -8, radius: 5000 },
        { lat: -90, lon: -180, radius: 2000 },
        { lat: 90, lon: 180, radius: 1 },
    ])("applies exactly one radius at ($lat, $lon) with $radius metres", async (near) => {
        const filter = await requestedFilter(near);

        expect(filter.match(/_geoRadius\([^)]*\)/g)).toEqual([
            `_geoRadius(${near.lat}, ${near.lon}, ${near.radius})`,
        ]);
    });

    it.each([undefined, NaN, Infinity, -Infinity, -90.01, 90.01])(
        "omits the radius for invalid latitude %s",
        async (lat) => {
            const filter = await requestedFilter({ lat, lon: 8, radius: 2000 });

            expect(filter).not.toContain("_geoRadius");
            expect(filter).toContain("distance >= 0");
        },
    );

    it.each([undefined, NaN, Infinity, -Infinity, -180.01, 180.01])(
        "omits the radius for invalid longitude %s",
        async (lon) => {
            const filter = await requestedFilter({ lat: 47, lon, radius: 2000 });

            expect(filter).not.toContain("_geoRadius");
            expect(filter).toContain("distance >= 0");
        },
    );

    it.each([0, -1, NaN, Infinity, -Infinity])(
        "omits invalid radius %s",
        async (radius) => {
            const filter = await requestedFilter({ lat: 47, lon: 8, radius });

            expect(filter).not.toContain("_geoRadius");
            expect(filter).toContain("distance >= 0");
        },
    );
});
