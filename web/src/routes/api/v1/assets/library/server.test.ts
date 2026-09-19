import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { POST } from "./+server";

function requestEvent(body: unknown) {
    const send = vi.fn().mockResolvedValue({ candidates: [], hasMore: false });
    return {
        request: new Request("http://localhost/api/v1/assets/library", {
            method: "POST",
            body: JSON.stringify(body),
        }),
        locals: { pb: { send } },
        fetch: vi.fn(),
    } as unknown as RequestEvent;
}

describe("photo library viewport proxy", () => {
    it.each([
        { west: 7, south: 46, east: 9, north: 48 },
        { west: 170, south: -10, east: -170, north: 10 },
        { west: -180, south: -90, east: 180, north: 90 },
    ])("preserves viewport bounds $west to $east through validation", async (bounds) => {
        const body = { bounds, page: 2, perPage: 100, takenAfter: "2026-01-01T00:00:00Z" };
        const event = requestEvent(body);
        expect((await POST(event)).status).toBe(200);
        expect(event.locals.pb.send).toHaveBeenCalledWith("/assets/library", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: expect.any(String),
            fetch: event.fetch,
        });
        const forwarded = vi.mocked(event.locals.pb.send).mock.calls[0][1]?.body;
        expect(JSON.parse(String(forwarded))).toEqual(body);
    });

    it.each([
        { west: 7, south: 48, east: 9, north: 46 },
        { west: -181, south: 46, east: 9, north: 48 },
        { west: 7, south: 46, east: 9 },
    ])("rejects invalid bounds before proxying", async (bounds) => {
        const event = requestEvent({ bounds });
        expect((await POST(event)).status).toBe(400);
        expect(event.locals.pb.send).not.toHaveBeenCalled();
    });
});
