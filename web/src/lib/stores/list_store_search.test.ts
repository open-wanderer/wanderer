import { afterEach, describe, expect, it, vi } from "vitest";
import type { ListFilter } from "$lib/models/list";
import { lists_search_filter } from "./list_store";

vi.mock("./trail_store", () => ({ fetchGPX: vi.fn() }));
vi.mock("./user_store", async () => {
    const { writable } = await import("svelte/store");
    return { currentUser: writable({ id: "user", actor: "owner" }) };
});

const filter: ListFilter = { q: "Alps", public: true, shared: true, sort: "created", sortOrder: "+" };
function response(id: string, page = 1, totalHits = 50) {
    return Response.json({ page, totalPages: 2, totalHits,
        hits: [{ id, name: id, author: "owner", created: 1700000000, trails: 0, shares: [] }] });
}
afterEach(() => vi.unstubAllGlobals());

describe("native list search", () => {
    it("preserves explicit filters and sorting while appending result pages", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(response("first"))
            .mockResolvedValueOnce(response("second", 2));
        const selected = { ...filter, q: "Bern", author: "other", public: false, shared: false };
        await lists_search_filter(selected, 1, 30, request);
        const body = JSON.parse(request.mock.calls[0][1].body);
        expect(body.q).toBe("Bern");
        expect(body.options).toMatchObject({ page: 1, hitsPerPage: 30, sort: ["created:asc"] });
        expect(body.options.filter).toContain("author = other");
        expect(body.options.filter).toContain("public = false");
        expect(body.options.filter).toContain("AND NOT shares = owner");
        expect((await lists_search_filter(selected, 2, 30, request)).items.map(item => item.id)).toEqual(["first", "second"]);
        expect(request).toHaveBeenCalledTimes(2);
        expect(JSON.parse(request.mock.calls[1][1].body).options).toMatchObject({ page: 2, hitsPerPage: 30, filter: body.options.filter });
    });

    it("does not let a late first page replace the newer search's accumulated pages", async () => {
        let resolveOld!: (response: Response) => void;
        const oldResponse = new Promise<Response>(resolve => { resolveOld = resolve; });
        const request = vi.fn()
            .mockReturnValueOnce(oldResponse)
            .mockResolvedValueOnce(response("new-first"))
            .mockResolvedValueOnce(response("new-second", 2));
        vi.stubGlobal("fetch", request);
        const oldSearch = lists_search_filter({ ...filter, q: "Old" });
        await lists_search_filter({ ...filter, q: "New" });
        resolveOld(response("old-first"));
        await oldSearch;
        expect((await lists_search_filter({ ...filter, q: "New" }, 2)).items.map(item => item.id))
            .toEqual(["new-first", "new-second"]);
    });

    it("reports failed searches instead of returning empty results", async () => {
        const request = vi.fn().mockResolvedValue(Response.json({ message: "Unavailable" }, { status: 503 }));
        await expect(lists_search_filter(filter, 1, 30, request)).rejects.toMatchObject({ status: 503 });
    });
});
