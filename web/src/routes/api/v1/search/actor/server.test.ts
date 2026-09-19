import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { getActorResponseForHandle } = vi.hoisted(() => ({
    getActorResponseForHandle: vi.fn(),
}));

vi.mock("$lib/util/activitypub_server_util", () => ({
    getActorResponseForHandle,
}));

import { GET } from "./+server";

function request(params: Record<string, string>, authenticated = true) {
    const result = { hits: [{ id: "local-actor" }], estimatedTotalHits: 1 };
    const search = vi.fn().mockResolvedValue(result);
    const index = vi.fn(() => ({ search }));
    const event = {
        url: new URL(`http://localhost/api/v1/search/actor?${new URLSearchParams(params)}`),
        locals: {
            user: authenticated ? { id: "user" } : null,
            pb: { authStore: { record: authenticated ? { actor: "own-actor" } : null } },
            ms: { index },
        },
    } as unknown as RequestEvent;
    return { event, index, search, result };
}

describe("actor search parameters", () => {
    beforeEach(() => vi.resetAllMocks());

    it("rejects a missing query with 400 before any lookup", async () => {
        const { event, index } = request({});

        await expect(GET(event)).rejects.toMatchObject({ status: 400 });
        expect(index).not.toHaveBeenCalled();
        expect(getActorResponseForHandle).not.toHaveBeenCalled();
    });

    it.each(["ali", ""])("searches for %j with the numeric default limit", async (q) => {
        const { event, index, search, result } = request({ q });

        const response = await GET(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual(result);
        expect(index).toHaveBeenCalledWith("actors");
        expect(search).toHaveBeenCalledWith(q, { filter: "", limit: 3 });
        expect(getActorResponseForHandle).not.toHaveBeenCalled();
    });

    it.each([0, 7, Number.MAX_SAFE_INTEGER])("passes explicit limit %s as a number", async (limit) => {
        const { event, search } = request({ q: "ali", limit: String(limit) });

        const response = await GET(event);

        expect(response.status).toBe(200);
        expect(search).toHaveBeenCalledWith("ali", { filter: "", limit });
    });

    it.each(["", " ", "-1", "1.5", "seven", "NaN", "Infinity", "9007199254740992"])(
        "rejects invalid limit %j before remote or index lookup",
        async (limit) => {
            const { event, index } = request({ q: "@alice@remote.example", limit });

            await expect(GET(event)).rejects.toMatchObject({ status: 400 });
            expect(index).not.toHaveBeenCalled();
            expect(getActorResponseForHandle).not.toHaveBeenCalled();
        },
    );

    it.each<Record<string, string>>([{}, { q: "ali" }, { q: "ali", limit: "invalid" }])(
        "requires authentication before validating %j",
        async (params) => {
            const { event, index } = request(params, false);

            await expect(GET(event)).rejects.toMatchObject({ status: 401 });
            expect(index).not.toHaveBeenCalled();
            expect(getActorResponseForHandle).not.toHaveBeenCalled();
        },
    );

    it("preserves the own-actor exclusion with an explicit limit", async () => {
        const { event, search } = request({ q: "ali", includeSelf: "false", limit: "7" });

        await GET(event);

        expect(search).toHaveBeenCalledWith("ali", { filter: "id != own-actor", limit: 7 });
    });

    it("returns a resolved remote actor even when the local result limit is zero", async () => {
        const q = "@alice@remote.example";
        const actor = { id: "remote-actor", preferred_username: "alice", domain: "remote.example", is_local: false };
        getActorResponseForHandle.mockResolvedValue({ actor });
        const { event, index } = request({ q, limit: "0", includeSelf: "false" });

        const response = await GET(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toMatchObject({ hits: [actor], query: q, totalHits: 1 });
        expect(getActorResponseForHandle).toHaveBeenCalledWith(event, q);
        expect(index).not.toHaveBeenCalled();
    });

    it("falls back to local search with the parsed limit when remote lookup fails", async () => {
        const q = "@alice@remote.example";
        getActorResponseForHandle.mockRejectedValue(new Error("Remote actor unavailable"));
        const { event, search, result } = request({ q, limit: "7", includeSelf: "false" });

        const response = await GET(event);

        expect(await response.json()).toEqual(result);
        expect(search).toHaveBeenCalledWith(q, { filter: "id != own-actor", limit: 7 });
    });
});
