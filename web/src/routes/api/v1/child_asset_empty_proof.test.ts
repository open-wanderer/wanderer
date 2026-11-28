import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { afterEach, expect, it, vi } from "vitest";
import { GET as getWaypoint } from "./waypoint/[id]/+server";
import { GET as getLog } from "./summit-log/[id]/+server";

afterEach(() => vi.unstubAllGlobals());
it.each([["waypoint", "empty"], ["waypoint", "failed"], ["waypoint", "nonempty"], ["summit_log", "empty"], ["summit_log", "failed"], ["summit_log", "nonempty"]] as const)("confirms %s photos from the requested collection (%s)", async (kind, state) => {
    const collection = kind === "waypoint" ? "waypoint_assets" : "summit_log_assets";
    const relation = `${collection}_via_${kind}`;
    const share = "a".repeat(32);
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input));
        if (url.pathname.includes(`/${collection}/records`)) {
            expect(url.searchParams.get("share")).toBe(share);
            expect(new Headers(init?.headers).get("Authorization")).toBe("reader-token");
            if (state === "failed") return Response.json({ message: "Read unavailable" }, { status: 503 });
            const items = state === "empty" ? [] : [{ id: "link", [kind]: "target000000001", asset: "photo", expand: { asset: { id: "photo", type: "photo", storage_mode: "copy", file: "photo.jpg" } } }];
            return Response.json({ page: 1, perPage: 500, totalItems: items.length, totalPages: items.length ? 1 : 0, items });
        }
        // A real empty PB backrelation has no expand object or photo keys.
        return Response.json({ id: "target000000001", date: "2026-10-01", author: "author", lat: 46, lon: 8 });
    });
    vi.stubGlobal("fetch", fetch);
    const pb = new PocketBase("https://a.example"); pb.authStore.save("reader-token");
    const url = new URL(`https://a.example/child/child?expand=${relation}.asset&share=${share}`);
    const event = { url, params: { id: "target000000001" }, fetch, locals: { pb }, request: new Request(url) } as unknown as RequestEvent;
    const response = await responseOverHttp(await (kind === "waypoint" ? getWaypoint : getLog)(event));
    expect(response.status).toBe(200);
    const record = await response.json();
    if (state === "failed") {
        expect(record.expand).not.toHaveProperty(relation);
        expect(record).not.toHaveProperty("photos");
    } else {
        expect(record.expand[relation]).toHaveLength(state === "empty" ? 0 : 1);
        expect(record.photos).toHaveLength(state === "empty" ? 0 : 1);
    }
    expect(fetch).toHaveBeenCalledTimes(2);
});
