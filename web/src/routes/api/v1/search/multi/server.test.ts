import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { POST } from "./+server";

describe("batch search", () => {
    it("keeps the existing contract and applies preferences to every trail query", async () => {
        const multiSearch = vi.fn(async (_body: unknown) => ({ results: [] }));
        const getFullList = vi.fn(async () => [{ category: "hidden", subcategory: "hidden-sub", visible: false }]);
        const event = { request: new Request("http://localhost/api/v1/search/multi", { method: "POST", body: JSON.stringify({
            queries: [{ indexUid: "trails", q: "Bern", filter: "public = true" }, { indexUid: "trails", q: "" }, { indexUid: "actors", q: "Bern" }],
        }) }), locals: { user: { id: "owner" }, ms: { multiSearch },
            pb: { filter: vi.fn(() => ""), collection: vi.fn(() => ({ getFullList })) } } } as unknown as RequestEvent;
        await POST(event);
        const body = multiSearch.mock.calls[0][0] as any;
        expect(body.federation).toBeUndefined();
        for (const query of body.queries.slice(0, 2)) expect(query.filter).toContain("(category_id IS NULL OR category_id NOT IN ['hidden'])");
        expect(body.queries[2]).toEqual({ indexUid: "actors", q: "Bern" });
        expect(getFullList).toHaveBeenCalledTimes(2);
    });
});
