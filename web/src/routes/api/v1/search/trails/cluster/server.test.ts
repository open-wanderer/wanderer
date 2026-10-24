import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { POST } from "./+server";

const bounds = { northEast: { lat: 49, lng: 11 }, southWest: { lat: 45, lng: 5 } };
const hits = [
    { id: "first", _geo: { lat: 47, lng: 7 }, bounding_box_diagonal: 100 },
    { id: "second", _geo: { lat: 47, lng: 8 }, bounding_box_diagonal: 200 },
];

function setup(outsideHits: object[] = []) {
    const multiSearch = vi.fn(async (_body: any) => ({ results: [
        { hits, estimatedTotalHits: 2 },
        { hits: outsideHits },
    ] }));
    const event = {
        request: new Request("http://localhost/api/v1/search/trails/cluster", {
            method: "POST",
            body: JSON.stringify({ ...bounds, zoom: 14, q: "Bern", filterText: "public = true AND category_id = 'hiking' AND _geoRadius(47, 8, 25000)" }),
        }),
        locals: {
            user: { id: "user1" },
            ms: { multiSearch },
            pb: { filter: vi.fn(() => ""), collection: vi.fn(() => ({ getFullList: async () => [{ category: "hidden", visible: false }] })) },
        },
    } as unknown as RequestEvent;
    return { event, multiSearch };
}

describe("native map search", () => {
    it("fetches markers and checks outside matches in one bounded multi-search", async () => {
        const { event, multiSearch } = setup();
        const result = await (await POST(event)).json();
        expect(multiSearch).toHaveBeenCalledOnce();
        const { queries } = multiSearch.mock.calls[0][0];
        expect(queries).toHaveLength(2);
        for (const query of queries) {
            expect(query.q).toBe("Bern");
            expect(query.filter).toContain("public = true AND category_id = 'hiking' AND _geoRadius(47, 8, 25000)");
            expect(query.filter).toContain("(category_id IS NULL OR category_id NOT IN ['hidden'])");
        }
        expect(queries[0].filter).toContain("max_lat >= 45 AND min_lat <= 49 AND max_lon >= 5 AND min_lon <= 11");
        expect(queries[0].limit).toBe(10000);
        expect(queries[1].filter).toContain("NOT (max_lat >= 45 AND min_lat <= 49 AND max_lon >= 5 AND min_lon <= 11)");
        expect(queries[1].limit).toBe(1);
        expect(result.totalHits).toBe(2);
        expect(result.hasOutsideBounds).toBe(false);
        expect(result.features.map((feature: any) => feature.properties.id).sort()).toEqual(["first", "second"]);
    });

    it("retains the viewport filter when a matching trail exists outside it", async () => {
        const { event } = setup([{ id: "outside" }]);
        expect((await (await POST(event)).json()).hasOutsideBounds).toBe(true);
    });
});
