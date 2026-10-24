import { describe, expect, it, vi } from "vitest";
import { searchActorById } from "./search_store";

describe("saved author filter resolution", () => {
    it("looks up the exact actor ID without treating it as search text", async () => {
        const id = 'actor-"quoted"';
        const actor = { id, preferred_username: "anna", is_local: false, domain: "example.org" };
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => Response.json({ hits: [actor] }));
        const controller = new AbortController();
        expect(await searchActorById(id, request, controller.signal)).toEqual(actor);
        expect(request).toHaveBeenCalledOnce();
        expect(request.mock.calls[0][0]).toBe("/api/v1/search/actors");
        expect(request.mock.calls[0][1]?.signal).toBe(controller.signal);
        const body = JSON.parse(String(request.mock.calls[0][1]?.body));
        expect(body.q).toBe("");
        expect(body.options.limit).toBe(1);
        expect(JSON.parse(body.options.filter.slice(5))).toBe(id);
    });

    it("does not accept a different actor when a saved ID cannot be resolved", async () => {
        const request = vi.fn(async () => Response.json({ hits: [{ id: "other", preferred_username: "other" }] }));
        expect(await searchActorById("missing", request)).toBeNull();
    });

    it("leaves unavailable actors unresolved without replacing the active filter", async () => {
        expect(await searchActorById("missing", async () => Response.json({}, { status: 503 }))).toBeNull();
    });
});
