import { describe, expect, it, vi } from "vitest";
import type { TrailFilter } from "$lib/models/trail";
import { sanitizeTrailFilter } from "$lib/util/trail_filter_util";
import { trails_search_filter, trails_search_filter_pages } from "./trail_store";

const filter: TrailFilter = {
    q: "Bern", category: ["hiking"], subcategory: [], tags: ["alpine"], difficulty: [1],
    author: "author1", public: true, private: false, shared: false, liked: true,
    near: { lat: 46, lon: 7, radius: 25_000, name: "Fixed place" },
    distanceMin: 1_000, distanceMax: 20_000, distanceLimit: 50_000,
    elevationGainMin: 100, elevationGainMax: 1_000, elevationGainLimit: 2_000,
    elevationLossMin: 100, elevationLossMax: 1_000, elevationLossLimit: 2_000,
    sort: "distance", sortOrder: "+",
};

describe("native trail search", () => {
    function response(hits: object[] = [], page = 1, totalHits = hits.length, perPage = 48) {
        return Response.json({ hits, page, hitsPerPage: perPage, totalHits, totalPages: Math.ceil(totalHits / perPage) });
    }

    it("uses one native request and intersects text with explicit filters and the viewport", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response([], 3, 101, 12));
        const bounds = { northEast: { lat: 49, lng: 11 }, southWest: { lat: 45, lng: 5 } };
        const result = await trails_search_filter(filter, 3, 12, request, bounds);
        expect(request).toHaveBeenCalledOnce();
        expect(request.mock.calls[0][0]).toBe("/api/v1/search/trails");
        const body = JSON.parse(String(request.mock.calls[0][1]?.body));
        expect(body.q).toBe("Bern");
        expect(body.options).toMatchObject({ page: 3, hitsPerPage: 12, sort: ["distance:asc"] });
        for (const clause of ["category_id IN ['hiking']", "tags = 'alpine'", "_geoRadius(46, 7, 25000)", "max_lat >= 45 AND min_lat <= 49"]) {
            expect(body.options.filter).toContain(clause);
        }
        expect(body).not.toHaveProperty("context");
        expect(body.options).not.toHaveProperty("facets");
        expect(body.options.filter).not.toContain("id NOT IN");
        expect(result).toMatchObject({ items: [], page: 3, totalHits: 101 });
    });

    it("preserves native relevance and the engine's hit order", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response([
            { id: "z", name: "Best match", _geo: { lat: 47, lng: 8 } },
            { id: "a", name: "Weaker match", _geo: { lat: 47, lng: 8 } },
        ]));
        const result = await trails_search_filter({ ...filter, sort: "relevance" }, 1, 48, request);
        expect(result.items.map(trail => trail.id)).toEqual(["z", "a"]);
        expect(JSON.parse(String(request.mock.calls[0][1]?.body)).options).not.toHaveProperty("sort");
    });

    it.each([
        { q: "", sortOrder: "+", expected: "created:asc" },
        { q: "", sortOrder: "-", expected: "created:desc" },
        { q: "  \t ", sortOrder: "+", expected: "created:asc" },
        { q: "  \t ", sortOrder: "-", expected: "created:desc" },
    ] as const)("sorts an empty relevance query by creation date: %j", async ({ q, sortOrder, expected }) => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response());
        const empty = { ...filter, q, sort: "relevance" as const, sortOrder };
        await trails_search_filter(empty, 1, 48, request);
        expect(JSON.parse(String(request.mock.calls[0][1]?.body)).options.sort).toEqual([expected]);
        expect(empty.sort).toBe("relevance");
    });

    it("restores only the loaded prefix in one native request and handles a shorter result set", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response(
            Array.from({ length: 5 }, (_, i) => ({ id: String(i), name: "Trail", _geo: { lat: 47, lng: 8 } })), 1, 5, 20));
        const result = await trails_search_filter_pages(filter, 10, 2, request);
        expect(request).toHaveBeenCalledOnce();
        expect(JSON.parse(String(request.mock.calls[0][1]?.body)).options.hitsPerPage).toBe(20);
        expect(result).toMatchObject({ page: 3, totalPages: 3, hitsPerPage: 2, totalHits: 5 });
        expect(result.items).toHaveLength(5);
    });

    it.each([1000, 1500, 30000])("ends pagination at the reachable 1000 hits even when Meili reports %i", async totalHits => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response(
            Array.from({ length: 40 }, (_, i) => ({ id: String(960 + i), name: "Trail", _geo: { lat: 47, lng: 8 } })), 21, totalHits));
        const result = await trails_search_filter(filter, 21, 48, request);
        expect(result).toMatchObject({ page: 21, totalPages: 21, totalHits });
        expect(result.items).toHaveLength(40);
        expect(request).toHaveBeenCalledOnce();
    });

    it("caps a restored prefix at 1000 hits and keeps the original page size", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response(
            Array.from({ length: 1000 }, (_, i) => ({ id: String(i), name: "Trail", _geo: { lat: 47, lng: 8 } })), 1, 1500, 1000));
        const result = await trails_search_filter_pages(filter, 32, 48, request);
        expect(JSON.parse(String(request.mock.calls[0][1]?.body)).options.hitsPerPage).toBe(1000);
        expect(result).toMatchObject({ page: 21, totalPages: 21, hitsPerPage: 48, totalHits: 1500 });
        expect(result.items).toHaveLength(1000);
        expect(request).toHaveBeenCalledOnce();
    });

    it("loads the next page directly without revalidating the entire result set", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, options?: RequestInit) => {
            const { page } = JSON.parse(String(options?.body)).options;
            return response([{ id: `page-${page}`, name: "Trail", _geo: { lat: 47, lng: 8 } }], page, 2, 1);
        });
        const first = await trails_search_filter(filter, 1, 1, request);
        const second = await trails_search_filter(filter, 2, 1, request);
        expect(first.items[0].id).toBe("page-1");
        expect(second.items[0].id).toBe("page-2");
        expect(request).toHaveBeenCalledTimes(2);
    });

    it("persists explicit place labels when restoring filters", () => {
        expect(sanitizeTrailFilter(filter, filter).near.name).toBe("Fixed place");
    });
});
