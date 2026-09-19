import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { GET, POST } from "./+server";

describe("trail publication proxy", () => {
    it.each([["GET", GET, 200], ["POST", POST, 202]] as const)("forwards %s with the authenticated client", async (method, handler, status) => {
        const job = { id: "job", status: "running" };
        const send = vi.fn().mockResolvedValue(job);
        const event = { params: { id: "trail0000000001" }, locals: { pb: { send } }, fetch: vi.fn() } as unknown as RequestEvent;
        const response = await handler(event);
        expect(response.status).toBe(status);
        expect(response.headers.get("Cache-Control")).toBe("private, no-store");
        expect(await response.json()).toEqual(job);
        expect(send).toHaveBeenCalledWith("/trails/trail0000000001/publication", { method, fetch: event.fetch, requestKey: null });
    });

    it("rejects invalid IDs before proxying", async () => {
        const send = vi.fn();
        const event = { params: { id: "../other" }, locals: { pb: { send } } } as unknown as RequestEvent;
        expect((await POST(event)).status).toBe(400);
        expect(send).not.toHaveBeenCalled();
    });
});
