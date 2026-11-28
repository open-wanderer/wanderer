import { describe, expect, it, vi } from "vitest";
import { PUT } from "./+server.ts";
import type { RequestEvent } from "@sveltejs/kit";

describe("summit log creation save lease", () => {
    const event = (token?: string) => {
        const create = vi.fn(async (_data: FormData, _options: Record<string, unknown>) => ({ id: "entry", date: "2026-11-09 00:00:00.000Z" }));
        const url = new URL("https://a.example/api/v1/summit-log/form");
        if (token !== undefined) url.searchParams.set("federationBatch", token);
        const data = new FormData();
        data.set("text", "Entry");
        return { create, request: { url, request: new Request(url, { method: "PUT", body: data }), locals: { user: { id: "user" }, pb: { collection: () => ({ create }) } } } as unknown as RequestEvent };
    };
    it("forwards the token before the entry is created", async () => {
        const { create, request } = event("entry-save-token-1234");
        const response = await PUT(request);
        expect(response.status).toBe(200);
        expect(create).toHaveBeenCalledWith(expect.any(FormData), expect.objectContaining({ federationBatch: "entry-save-token-1234" }));
    });
    it.each(["", "short", "entry save token with spaces", "x".repeat(129)])("rejects invalid token %s before creation", async (token) => {
        const { create, request } = event(token);
        expect((await PUT(request)).status).toBe(400);
        expect(create).not.toHaveBeenCalled();
    });
    it("keeps callers without a token working", async () => {
        const { create, request } = event();
        expect((await PUT(request)).status).toBe(200);
        expect(create).toHaveBeenCalledOnce();
        expect(create.mock.calls[0][1]).not.toHaveProperty("federationBatch");
    });
});
