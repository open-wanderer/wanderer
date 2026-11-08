import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import type { Trail } from "#lib/models/trail.ts";
import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { afterEach, expect, it, vi } from "vitest";
import { GET as getTrail } from "./trail/[id]/+server";

const share = "a".repeat(32);
afterEach(() => vi.unstubAllGlobals());

// These are the actual PocketBase response shapes: an empty backrelation is
// omitted completely, including its enclosing expand when nothing else exists.
function originRecord(): Trail {
    return {
        id: "trail", name: "Trail", author: "author", public: true, completed: false,
        expand: {
            waypoints_via_trail: [{ id: "point", lat: 46, lon: 8, author: "author" }],
            summit_logs_via_trail: [{ id: "log", author: "author", date: "2026-10-01" }],
        },
    } as Trail;
}

function event(fetch: typeof globalThis.fetch): RequestEvent {
    const url = new URL(`https://a.example/api/v1/trail/trail?share=${share}&expand=waypoints_via_trail,summit_logs_via_trail`);
    const pb = new PocketBase("https://a.example");
    pb.authStore.save("reader-token");
    return { url, params: { id: "trail" }, fetch, locals: { pb }, request: new Request(url) } as unknown as RequestEvent;
}

it.each(["empty", "failed", "nonempty"])("returns a confirmed photo relation only after the real SDK query (%s)", async (state) => {
    const requests: string[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input));
        if (url.pathname === "/remote/trail/trail") return Response.json(originRecord());
        const match = url.pathname.match(/^\/api\/collections\/(trail_assets|waypoint_assets|summit_log_assets)\/records$/);
        expect(match, `unexpected query ${url}`).toBeTruthy();
        const collection = match![1] as "trail_assets" | "waypoint_assets" | "summit_log_assets";
        requests.push(collection);
        expect(new Headers(init?.headers).get("Authorization")).toBe("reader-token");
        expect(url.searchParams.get("share")).toBe(share);
        expect(url.searchParams.get("expand")).toBe("asset");
        expect(url.searchParams.get("filter")).toBe({ trail_assets: 'trail = "trail"', waypoint_assets: 'waypoint = "point"', summit_log_assets: 'summit_log = "log"' }[collection]);
        if (state === "failed") return Response.json({ message: "Relation read failed" }, { status: 503 });
        const target = { trail_assets: ["trail", "trail"], waypoint_assets: ["waypoint", "point"], summit_log_assets: ["summit_log", "log"] }[collection]!;
        const items = state === "empty" ? [] : [{ id: "link", [target[0]]: target[1], asset: "photo", expand: { asset: { id: "photo", type: "photo", storage_mode: "copy", file: "summit.jpg" } } }];
        return Response.json({ page: 1, perPage: 500, totalItems: items.length, totalPages: items.length ? 1 : 0, items });
    });
    vi.stubGlobal("fetch", fetch);
    const response = await responseOverHttp(await getTrail(event(fetch)));
    expect(response.status).toBe(200);
    const body = await response.json();
    if (state === "empty" && process.env.WANDERER_WIRE_CAPTURE) {
        const { writeFileSync } = await import("node:fs");
        writeFileSync(process.env.WANDERER_WIRE_CAPTURE, JSON.stringify(body));
    }
    expect(requests.sort()).toEqual(["summit_log_assets", "trail_assets", "waypoint_assets"]);
    for (const [record, key] of [[body, "trail_assets_via_trail"], [body.expand.waypoints_via_trail[0], "waypoint_assets_via_waypoint"], [body.expand.summit_logs_via_trail[0], "summit_log_assets_via_summit_log"]] as const) {
        if (state === "failed") expect(record.expand).not.toHaveProperty(key);
        else {
            expect(record.expand[key]).toHaveLength(state === "empty" ? 0 : 1);
            expect(record.photos).toHaveLength(state === "empty" ? 0 : 1);
        }
    }
});

it("confirms empty photos for 150 waypoints with bounded real SDK filters", async () => {
    const record = originRecord();
    record.expand!.trail_assets_via_trail = [];
    record.expand!.summit_logs_via_trail = [];
    record.expand!.waypoints_via_trail = Array.from({ length: 150 }, (_, i) => ({
        id: `point${String(i).padStart(10, "0")}`, author: "author", lat: 46, lon: 8,
    })) as unknown as NonNullable<Trail["expand"]>["waypoints_via_trail"];
    const filters: string[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input));
        if (url.pathname === "/remote/trail/trail") return Response.json(record);
        expect(url.pathname).toBe("/api/collections/waypoint_assets/records");
        const filter = url.searchParams.get("filter")!;
        filters.push(filter);
        // Reject too-long filters as the backend does, rather than pretending
        // that a failed proof establishes an empty relation.
        if (filter.length > 3500) return Response.json({ message: "Filter too long" }, { status: 400 });
        return Response.json({ page: 1, perPage: 500, totalItems: 0, totalPages: 0, items: [] });
    });
    vi.stubGlobal("fetch", fetch);
    // getTrail calls completeTrailAssetExpands before deriving wire photos.
    const response = await responseOverHttp(await getTrail(event(fetch)));
    expect(response.status).toBe(200);
    const body = await response.json();
    expect(filters.length).toBeGreaterThan(1);
    expect(filters.every((filter) => filter.length <= 3500)).toBe(true);
    expect(body.expand.waypoints_via_trail).toHaveLength(150);
    expect(body.expand.waypoints_via_trail.every((point: { photos?: string[] }) => Array.isArray(point.photos) && point.photos.length === 0)).toBe(true);
});
