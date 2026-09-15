import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { List } from "$lib/models/list";
import type { Trail } from "$lib/models/trail";
import { list_share_create_for_actor, shares } from "./list_share_store";

const owner = "owner0000000000";
const actor = { iri: "https://example.com/recipient", is_local: false };
const trail = (id: string, publicTrail = false, author = owner) =>
    ({ id, public: publicTrail, author }) as Trail;

function mixedList() {
    const list = new List("Mixed trails", [
        trail("private00000000"),
        trail("shared000000000"),
        trail("public000000000", true),
        trail("foreign00000000", false, "other0000000000"),
    ], { public: true, author: owner });
    list.id = "list00000000000";
    return list;
}

beforeEach(() => shares.set([]));
afterEach(() => vi.unstubAllGlobals());

describe("sharing a list with an actor", () => {
    it("shares a public list remotely and refreshes it without private trail requests", async () => {
        const list = mixedList();
        const created = { id: "share0000000000", list: list.id, actor: actor.iri, permission: "view" };
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            if (String(input).startsWith("/api/v1/trail-share")) {
                return Response.json({ message: "Private remote shares are forbidden" }, { status: 400 });
            }
            return Response.json(options?.method === "PUT" ? {} : { items: [created] });
        });
        vi.stubGlobal("fetch", request);

        const result = await list_share_create_for_actor(list, actor, owner);

        expect(request).toHaveBeenCalledTimes(2);
        expect(request.mock.calls[0]).toEqual(["/api/v1/list-share", {
            method: "PUT",
            body: JSON.stringify({ actor: actor.iri, list: list.id, permission: "view" }),
        }]);
        expect(String(request.mock.calls[1][0])).toMatch(/^\/api\/v1\/list-share\?/);
        expect(request.mock.calls[1][1]?.method).toBe("GET");
        expect(result.items).toEqual([created]);
        expect(get(shares)).toEqual([created]);
    });

    it("keeps automatically sharing only unshared private trails owned by the sender locally", async () => {
        const list = mixedList();
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json({}))
            .mockResolvedValueOnce(Response.json({ items: [{ trail: "shared000000000" }] }))
            .mockResolvedValueOnce(Response.json({}))
            .mockResolvedValueOnce(Response.json({ items: [] }));
        vi.stubGlobal("fetch", request);

        await list_share_create_for_actor(list, { ...actor, is_local: true }, owner);

        expect(request).toHaveBeenCalledTimes(4);
        expect(String(request.mock.calls[1][0])).toMatch(/^\/api\/v1\/trail-share\?/);
        expect(new URL(String(request.mock.calls[1][0]), "http://localhost").searchParams.get("filter"))
            .toBe(`actor.iri='${actor.iri}'`);
        expect(request.mock.calls[2]).toEqual(["/api/v1/trail-share", {
            method: "PUT",
            body: JSON.stringify({ actor: actor.iri, trail: "private00000000", permission: "view" }),
        }]);
        expect(String(request.mock.calls[3][0])).toMatch(/^\/api\/v1\/list-share\?/);
    });

    it("stops when creating the list share fails", async () => {
        const request = vi.fn().mockResolvedValue(Response.json({ message: "Forbidden" }, { status: 400 }));
        vi.stubGlobal("fetch", request);

        await expect(list_share_create_for_actor(mixedList(), actor, owner)).rejects.toMatchObject({ status: 400 });

        expect(request).toHaveBeenCalledOnce();
        expect(get(shares)).toEqual([]);
    });
});
