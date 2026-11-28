import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import { afterEach, expect, it, vi } from "vitest";
import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import type { Waypoint } from "#lib/models/waypoint.ts";
import { completeChildAssetResponse } from "#lib/server/trail_asset_pagination.ts";
import { GET as getWaypoint, POST as updateWaypoint } from "./waypoint/[id]/+server";
import { GET as getLog, POST as updateLog } from "./summit-log/[id]/+server";
import { POST as updateLogForm } from "./summit-log/form/[id]/+server";

afterEach(() => vi.unstubAllGlobals());
it.each([
    ["waypoint", getWaypoint, "GET", 0], ["waypoint", updateWaypoint, "POST", 0],
    ["summit_log", getLog, "GET", 0], ["summit_log", updateLog, "POST", 0], ["summit_log", updateLogForm, "FORM", 0],
    ["waypoint", updateWaypoint, "POST", 2], ["summit_log", updateLog, "POST", 2], ["summit_log", updateLogForm, "FORM", 2],
] as const)("completes %s photos on %s or reports the committed record (failedPage=%s)", async (kind, endpoint, method, failedPage) => {
    const id = "target000000001";
    const collection = kind === "waypoint" ? "waypoint_assets" : "summit_log_assets";
    const relation = `${collection}_via_${kind}`;
    const links = Array.from({ length: 1110 }, (_, i) => ({
        id: `links${String(i).padStart(10, "0")}`, asset: `asset${String(i).padStart(10, "0")}`,
        expand: { asset: { id: `asset${String(i).padStart(10, "0")}`, type: "photo", storage_mode: "link_private" } },
    }));
    const share = "a".repeat(32);
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input));
        if (url.pathname.includes(`/${collection}/records`)) {
            expect(url.searchParams.get("share")).toBe(share);
            const page = Number(url.searchParams.get("page"));
            if (page === failedPage) return Response.json({ message: "Read denied" }, { status: 403 });
            return Response.json({ page, perPage: 500, totalItems: links.length, totalPages: 3, items: links.slice((page - 1) * 500, page * 500) });
        }
        return Response.json({ id, date: "2026-10-03", lat: 47, lon: 8, expand: { [relation]: links.slice(0, 1000) } });
    });
    vi.stubGlobal("fetch", fetch);
    const pb = new PocketBase("https://a.example");
    pb.authStore.save("owner-token");
    const url = new URL(`https://a.example/child/${id}?expand=${relation}.asset&share=${share}`);
    const form = new FormData();
    form.append("name", "Updated");
    const event = { url, params: { id }, fetch, locals: { pb, user: { id: "owner" } },
        request: new Request(url, { method: method === "GET" ? "GET" : "POST", ...(method !== "GET" ? { body: method === "FORM" ? form : JSON.stringify({ name: "Updated" }) } : {}) }),
    } as unknown as RequestEvent;
    const response = await responseOverHttp(await endpoint(event));
    const body = await response.json();
    if (failedPage) {
        expect(response.status).toBe(403);
        expect(body.detail).toMatchObject({ recordCommitted: true, savedRecord: { id } });
        expect(body.detail.savedRecord).not.toHaveProperty("photos");
        expect(body.detail.savedRecord).not.toHaveProperty("expand");
        expect(fetch).toHaveBeenCalledTimes(3);
    } else {
        expect(response.status).toBe(200);
        expect(body.photos).toHaveLength(1110);
        expect(fetch).toHaveBeenCalledTimes(4);
    }
});

it("does not fabricate an empty photo list for a links-only expand", async () => {
    const query = vi.fn().mockResolvedValue([]);
    const record = { id: "waypoint0000001", lat: 0, lon: 0, expand: { waypoint_assets_via_waypoint: [{ asset: "asset0000000001" }] } } as Waypoint;
    const event = { url: new URL("http://a.example/waypoint?expand=waypoint_assets_via_waypoint"), locals: { pb: { collection: () => ({ getFullList: query }) } } } as unknown as RequestEvent;
    await completeChildAssetResponse(event, record, "waypoint");
    expect(record).not.toHaveProperty("photos");
    expect(record.expand!.waypoint_assets_via_waypoint).toHaveLength(1);
    expect(query).not.toHaveBeenCalled();
});
