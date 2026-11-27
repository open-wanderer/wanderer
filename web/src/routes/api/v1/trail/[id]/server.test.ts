import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { describe, expect, it, vi } from "vitest";
import { GET } from "./+server";

function trailEvent(query: string, fetch: typeof globalThis.fetch): RequestEvent {
    const url = new URL(`https://a.example/api/v1/trail/trail0000000001${query}`);
    return {
        url, request: new Request(url), params: { id: "trail0000000001" },
        locals: { pb: new PocketBase("https://b.example") }, fetch,
    } as unknown as RequestEvent;
}

describe("backend trail child completeness", () => {
    it.each([
        ["?expand=waypoints_via_trail,summit_logs_via_trail", [], []],
        ["?expand=waypoints_via_trail", [], undefined],
        ["?expand=waypoints_via_trail.author,summit_logs_via_trail&share=shared-token", undefined, []],
        ["?expand=waypoints_via_trail,summit_logs_via_trail", undefined, undefined],
        ["", undefined, undefined],
        ["?expand=waypoints_via_trail&fields=id,name", undefined, undefined],
    ])("passes through backend emptiness/unknown results for '%s' without SDK list proofs", async (query, waypoints, logs) => {
        const expand = {
            ...(waypoints === undefined ? {} : { waypoints_via_trail: waypoints }),
            ...(logs === undefined ? {} : { summit_logs_via_trail: logs }),
        };
        const fetch = vi.fn(async () => Response.json({ id: "trail0000000001", name: "Trail", expand }));
        const event = trailEvent(query, fetch);
        const waypointsList = vi.spyOn(event.locals.pb.collection("waypoints"), "getList");
        const logsList = vi.spyOn(event.locals.pb.collection("summit_logs"), "getList");
        const response = await GET(event);
        expect(response.status).toBe(200);
        const trail = await response.json();
        expect(trail.expand?.waypoints_via_trail).toEqual(waypoints);
        expect(trail.expand?.summit_logs_via_trail).toEqual(logs);
        expect(fetch).toHaveBeenCalledOnce();
        expect(waypointsList).not.toHaveBeenCalled();
        expect(logsList).not.toHaveBeenCalled();
    });

    it("preserves auth/share/handle/projection forwarding, date enrichment and waypoint sorting", async () => {
        const fetch = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => Response.json({
            id: "storedtrail00001", date: "2026-10-01 12:00:00.000Z",
            expand: {
                waypoints_via_trail: [{ id: "far", distance_from_start: 3 }, { id: "start" }, { id: "near", distance_from_start: 1 }],
                summit_logs_via_trail: [{ id: "log", date: "2026-09-01 12:00:00.000Z" }],
            },
        }));
        const query = "?expand=waypoints_via_trail,summit_logs_via_trail&share=shared-token&handle=@alice@remote.example&fields=";
        const event = trailEvent(query, fetch);
        event.locals.pb.authStore.save("caller-token");
        const response = await GET(event);
        expect(response.status).toBe(200);
        const trail = await response.json();
        expect(trail.id).toBe("storedtrail00001");
        expect(trail.date).toBe("2026-10-01");
        expect(trail.expand.summit_logs_via_trail[0].date).toBe("2026-09-01");
        expect(trail.expand.waypoints_via_trail.map((point: { id: string }) => point.id)).toEqual(["start", "near", "far"]);
        expect(fetch).toHaveBeenCalledOnce();
        const [input, options] = fetch.mock.calls[0];
        expect(new URL(String(input)).pathname).toBe("/remote/trail/trail0000000001");
        expect(new URL(String(input)).searchParams.toString()).toBe(event.url.searchParams.toString());
        expect(new Headers(options?.headers).get("Authorization")).toBe("caller-token");
    });
});
