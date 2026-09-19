import type { RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";
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

    it("returns idle as a successful, uncached status response", async () => {
        const status = { trailId: "trail0000000001", status: "idle" };
        const send = vi.fn().mockResolvedValue(status);
        const event = { params: { id: status.trailId }, locals: { pb: { send } }, fetch: vi.fn() } as unknown as RequestEvent;
        const response = await GET(event);
        expect(response.status).toBe(200);
        expect(response.headers.get("Cache-Control")).toBe("private, no-store");
        expect(await response.json()).toEqual(status);
    });

    it.each([401, 403, 404])("preserves backend HTTP %s errors", async (status) => {
        const failure = { status, message: "Status request failed.", data: {} };
        const send = vi.fn().mockRejectedValue(new ClientResponseError({ status, response: failure, originalError: { data: failure } }));
        const event = { params: { id: "trail0000000001" }, locals: { pb: { send } }, fetch: vi.fn() } as unknown as RequestEvent;
        const response = await GET(event);
        expect(response.status).toBe(status);
        expect(await response.json()).toMatchObject(failure);
    });
});
