import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { describe, expect, it, vi } from "vitest";
import { GET, POST } from "./+server";

const assets = [
    { id: "asset0000000001", collectionId: "pbc_assets", collectionName: "assets", type: "photo", file: "first.jpg", storage_mode: "copy" },
    { id: "asset0000000002", collectionId: "pbc_assets", collectionName: "assets", type: "photo", file: "second.jpg", storage_mode: "copy" },
];
const links = assets.map((asset, i) => ({
    id: `link00000000000${i + 1}`,
    trail: "trail0000000001",
    asset: asset.id,
    is_thumbnail: i === 1,
    expand: { asset },
}));

// Like PocketBase, the update response only carries the requested expands.
function pocketBaseFetch() {
    return vi.fn(async (url: RequestInfo | URL, _options?: RequestInit) => {
        const expand = new URL(String(url)).searchParams.get("expand")?.split(",") ?? [];
        return Response.json({
            id: "trail0000000001", name: "Renamed", date: "2026-10-01 00:00:00.000Z", thumbnail: 0,
            ...(expand.includes("trail_assets_via_trail.asset") ? { expand: { trail_assets_via_trail: links } } : {}),
        });
    });
}

function trailEvent(query: string, fetch: typeof globalThis.fetch, update = false): RequestEvent {
    const url = new URL(`https://a.example/api/v1/trail/trail0000000001${query}`);
    const pb = new PocketBase("https://b.example");
    pb.beforeSend = (url, options) => ({ url, options: { ...options, fetch } });
    return {
        url,
        request: new Request(url, update ? { method: "POST", body: JSON.stringify({ name: "Renamed" }) } : undefined),
        params: { id: "trail0000000001" }, locals: { pb }, fetch,
    } as unknown as RequestEvent;
}

describe("trail update endpoint", () => {
    it.each([
        ["", "trail_assets_via_trail.asset"],
        ["?expand=tags", "tags,trail_assets_via_trail.asset"],
    ])("answers with the trail photos for the query '%s'", async (query, expand) => {
        const fetch = pocketBaseFetch();
        const response = await responseOverHttp(await POST(trailEvent(query, fetch, true)));
        expect(response.status).toBe(200);
        expect(fetch).toHaveBeenCalledOnce();
        expect(new URL(String(fetch.mock.calls[0][0])).searchParams.get("expand")).toBe(expand);
        const trail = await response.json();
        expect(trail.photos).toEqual([
            "/api/v1/files/pbc_assets/asset0000000001/first.jpg",
            "/api/v1/files/pbc_assets/asset0000000002/second.jpg",
        ]);
        expect(trail.thumbnail).toBe(1);
    });
});

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
            trail_assets_via_trail: [],
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

    it.each([200, 403, 500])("keeps native child completeness independent of an asset-link proof returning %s", async (status) => {
        const fetch = vi.fn(async (input: RequestInfo | URL) => {
            if (new URL(String(input)).pathname === "/api/collections/trail_assets/records") {
                return Response.json(status === 200 ? { page: 1, perPage: 500, items: [] } : { message: "Unavailable" }, { status });
            }
            return Response.json({ id: "trail0000000001", expand: { summit_logs_via_trail: [] } });
        });
        const event = trailEvent("?expand=waypoints_via_trail,summit_logs_via_trail&share=shared-token", fetch);
        const response = await GET(event);
        expect(response.status).toBe(200);
        const trail = await response.json();
        expect(trail.expand).not.toHaveProperty("waypoints_via_trail");
        expect(trail.expand.summit_logs_via_trail).toEqual([]);
        expect(fetch.mock.calls.some(([input]) => /\/collections\/(waypoints|summit_logs)\/records/.test(String(input)))).toBe(false);
        const proofURL = fetch.mock.calls.map(([input]) => new URL(String(input))).find(url => url.pathname === "/api/collections/trail_assets/records")!;
        expect(proofURL.searchParams.get("share")).toBe("shared-token");
        expect(proofURL.searchParams.get("filter")).toContain("trail0000000001");
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
        const forwarded = new URL(String(input));
        expect(forwarded.pathname).toBe("/remote/trail/trail0000000001");
        for (const key of ["share", "handle", "fields"]) {
            expect(forwarded.searchParams.get(key)).toBe(event.url.searchParams.get(key));
        }
        expect(forwarded.searchParams.get("expand")?.split(",")).toEqual([
            "waypoints_via_trail", "summit_logs_via_trail", "trail_assets_via_trail.asset",
            "waypoints_via_trail.waypoint_assets_via_waypoint.asset", "summit_logs_via_trail.summit_log_assets_via_summit_log.asset",
        ]);
        expect(event.url.search).toBe(query);
        expect(new Headers(options?.headers).get("Authorization")).toBe("caller-token");
    });
});

describe("trail access error responses", () => {
    const marker = { code: "federated_share_denied", message: "Share access denied." };

    it.each([
        ["", 403, { share: marker }],
        ["&fields=id", 403, { share: marker }],
        ["", 403, {}],
        ["", 500, {}],
    ])("preserves the origin access result for '%s' with status %s", async (fields, status, data) => {
        const fetch = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) =>
            Response.json({ status, message: "Trail access failed.", data }, { status }));
        const token = "s".repeat(32);
        const event = trailEvent(`?share=${token}${fields}`, fetch);

        const response = await responseOverHttp(await GET({ ...event, request: new Request(event.url) }));

        expect(response.status).toBe(status);
        expect((await response.json()).data).toEqual(data);
        expect(fetch).toHaveBeenCalledOnce();
        const requested = new URL(String(fetch.mock.calls[0][0]));
        expect(requested.pathname).toBe("/remote/trail/trail0000000001");
        expect(requested.searchParams.get("share")).toBe(token);
        if (fields) expect(requested.searchParams.get("fields")).toBe("id");
    });
});
