import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { describe, expect, it, vi } from "vitest";
import { GET } from "./+server";

// PocketBase leaves an empty reverse relation out of the record response.
function pocketBaseFetch() {
    return vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => {
        if (/\/api\/collections\/(waypoints|summit_logs)\/records$/.test(new URL(String(_url)).pathname)) {
            return Response.json({ page: 1, perPage: 1, totalItems: 0, totalPages: 0, items: [] });
        }
        return Response.json({ id: "trail0000000001", name: "Trail", date: "2026-10-01 00:00:00.000Z" });
    });
}

function trailEvent(query: string, fetch: typeof globalThis.fetch): RequestEvent {
    const url = new URL(`https://a.example/api/v1/trail/trail0000000001${query}`);
    return {
        url, request: new Request(url), params: { id: "trail0000000001" },
        locals: { pb: new PocketBase("https://b.example") }, fetch,
    } as unknown as RequestEvent;
}

describe("trail child completeness", () => {
    it.each([
        ["?expand=waypoints_via_trail,summit_logs_via_trail", [], []],
        ["?expand=waypoints_via_trail", [], undefined],
        ["", undefined, undefined],
        ["?expand=waypoints_via_trail&fields=id,name", undefined, undefined],
    ])("marks requested empty children for '%s'", async (query, waypoints, logs) => {
        const event = trailEvent(query, pocketBaseFetch());
        const response = await GET(event);
        expect(response.status).toBe(200);
        const trail = await response.json();
        expect(trail.expand?.waypoints_via_trail).toEqual(waypoints);
        expect(trail.expand?.summit_logs_via_trail).toEqual(logs);
    });

    it.each([403, 404, 500, "nonempty"])("preserves an unknown relation when the completeness proof is %s", async (proof) => {
        const original = pocketBaseFetch();
        const fetch = vi.fn(async (url: RequestInfo | URL, options?: RequestInit) => {
            if (new URL(String(url)).pathname === "/api/collections/waypoints/records") {
                if (proof === "nonempty") return Response.json({ page: 1, perPage: 1, totalItems: 1, totalPages: 1, items: [{ id: "existing-child" }] });
                return Response.json({ message: "Relation unavailable" }, { status: Number(proof) });
            }
            return original(url, options);
        });
        const event = trailEvent("?expand=waypoints_via_trail.author,summit_logs_via_trail&share=shared-token", fetch);
        const response = await GET(event);
        expect(response.status).toBe(200);
        const trail = await response.json();
        expect(trail.expand?.waypoints_via_trail).toBeUndefined();
        expect(trail.expand?.summit_logs_via_trail).toEqual([]);
        const proofURL = fetch.mock.calls.map(([url]) => new URL(String(url))).find((url) => url.pathname === "/api/collections/waypoints/records")!;
        expect(proofURL.searchParams.get("share")).toBe("shared-token");
        expect(proofURL.searchParams.get("perPage")).toBe("1");
        expect(proofURL.searchParams.get("fields")).toBe("id");
        expect(proofURL.searchParams.get("filter")).toContain("trail0000000001");
    });
});

it("leaves existing child expands untouched without additional relation queries", async () => {
    const fetch = vi.fn(async () => Response.json({
        id: "trail0000000001", name: "Trail",
        expand: { waypoints_via_trail: [{ id: "waypoint0000001", distance_from_start: 1 }], summit_logs_via_trail: [] },
    }));
    const response = await GET(trailEvent("?expand=waypoints_via_trail,summit_logs_via_trail", fetch));
    expect(response.status).toBe(200);
    expect((await response.json()).expand.waypoints_via_trail).toHaveLength(1);
    expect(fetch).toHaveBeenCalledOnce();
});
