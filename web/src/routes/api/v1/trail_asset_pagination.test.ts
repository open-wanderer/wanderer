import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import type { AssetLink } from "#lib/models/asset.ts";
import type { Trail } from "#lib/models/trail.ts";
import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("#lib/server/category_preference_filter.ts", () => ({
    withTrailPreferencePocketBaseFilter: async (_event: RequestEvent, filter?: string) => filter,
}));

import { PUT as createTrailForm } from "./trail/form/+server";
import { GET as getTrail, POST as updateTrail } from "./trail/[id]/+server";
import { GET as getTrails } from "./trail/+server";
import { GET as getLists } from "./list/+server";
import { GET as getList } from "./list/[id]/+server";
import { completeTrailAssetExpands } from "#lib/server/trail_asset_pagination.ts";

const trailId = "trail0000000001";
const waypointId = "waypoint0000001";
const summitLogId = "summitlog000001";
const share = "a".repeat(32);

function photoLinks(count: number): AssetLink[] {
    return Array.from({ length: count }, (_, i) => {
        const id = `asset${String(i).padStart(10, "0")}`;
        return {
            id: `links${String(i).padStart(10, "0")}`,
            asset: id, is_thumbnail: i === 1075,
            expand: { asset: {
                id, collectionId: "assets", collectionName: "assets", type: "photo",
                author: "author000000001", storage_mode: "link_private",
            } },
        };
    });
}

interface TransportOptions {
    count: number;
    children?: boolean;
    rejectPage?: number;
}

// Use the real SDK pagination and its actual query/header serialization.
function pocketBaseTransport(options: TransportOptions) {
    const links = photoLinks(options.count);
    const trail: Trail = {
        id: trailId, name: "Trail", public: false, completed: false,
        tags: [], like_count: 0, author: "author000000001", photos: [], thumbnail: 0,
        expand: {
            trail_assets_via_trail: links.slice(0, 1000),
            ...(options.children ? {
                waypoints_via_trail: [{
                    id: waypointId, author: "author000000001", lat: 46, lon: 7, photos: [],
                    expand: { waypoint_assets_via_waypoint: links.slice(0, 1000) },
                }],
                summit_logs_via_trail: [{
                    id: summitLogId, author: "author000000001", date: "2026-10-01", photos: [],
                    expand: { summit_log_assets_via_summit_log: links.slice(0, 1000) },
                }],
            } : {}),
        },
    };
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input));
        const send = (body: unknown, status = 200) => new Response(JSON.stringify(body), {
            status, headers: { "Content-Type": "application/json" },
        });
        const match = url.pathname.match(/^\/api\/collections\/(trail_assets|waypoint_assets|summit_log_assets)\/records$/);
        if (match) {
            const expectedTargets = {
                trail_assets: ["trail", trailId],
                waypoint_assets: ["waypoint", waypointId],
                summit_log_assets: ["summit_log", summitLogId],
            };
            const [field, id] = expectedTargets[match[1] as keyof typeof expectedTargets];
            expect(url.searchParams.get("filter")).toBe(`${field} = "${id}"`);
            expect(url.searchParams.get("expand")).toBe("asset");
            expect(url.searchParams.get("sort")).toBe("+created,+id");
            const authorized = new Headers(init?.headers).get("Authorization") === "owner-token" || url.searchParams.get("share") === share;
            if (!authorized) return send({ message: "Unauthorized" }, 401);
            const page = Number(url.searchParams.get("page"));
            const perPage = Number(url.searchParams.get("perPage"));
            expect(perPage).toBe(500);
            if (page === options.rejectPage) return send({ message: "Read denied" }, 403);
            return send({
                page, perPage, totalItems: links.length, totalPages: Math.ceil(links.length / perPage),
                items: links.slice((page - 1) * perPage, page * perPage),
            });
        }
        if (url.pathname.startsWith("/remote/list/")) return send({ expand: { trails: [structuredClone(trail)] } });
        if (url.pathname === "/api/collections/lists/records") {
            expect(url.searchParams.get("expand")).toContain("trails.trail_assets_via_trail.asset");
            return send({ items: [{ expand: { trails: [structuredClone(trail)] } }] });
        }
        if (url.pathname === "/api/collections/trails/records") return send({ items: [structuredClone(trail)] });
        return send(structuredClone(trail));
    });
    vi.stubGlobal("fetch", fetch);
    return fetch;
}

function event(path: string, fetch: typeof globalThis.fetch, token = "owner-token", method = "GET"): RequestEvent {
    const pb = new PocketBase("https://a.example");
    pb.authStore.save(token);
    const url = new URL(`https://a.example${path}`);
    return {
        url, params: { id: trailId }, fetch, locals: { pb, ...(token ? { user: { id: "owner0000000001", actor: "author000000001" } } : {}) },
        request: new Request(url, { method, ...(method === "POST" ? { body: JSON.stringify({ name: "Renamed" }) } : {}) }),
    } as unknown as RequestEvent;
}

afterEach(() => vi.unstubAllGlobals());

describe("complete trail asset responses", () => {
    it("sorts an unsaturated expand by creation time and link ID too", async () => {
        const links = photoLinks(3).map((link, i) => ({ ...link, created: i === 0 ? "2026-10-02" : "2026-10-01" })).reverse();
        const trail = { expand: { trail_assets_via_trail: links } } as unknown as Trail;
        await completeTrailAssetExpands({} as RequestEvent, [trail]);
        expect(trail.expand!.trail_assets_via_trail!.map((link) => link.id)).toEqual([links[1].id, links[0].id, links[2].id]);
    });
    it.each(["owner-token", ""])("loads all 1110 photos on every requested target, preserving access (%s)", async (token) => {
        const fetch = pocketBaseTransport({ count: 1110, children: true });
        const query = new URLSearchParams({
            share,
            expand: "waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail.summit_log_assets_via_summit_log.asset",
        });
        const response = await responseOverHttp(await getTrail(event(`/api/v1/trail/${trailId}?${query}`, fetch, token)));
        expect(response.status, await response.clone().text()).toBe(200);
        const trail = await response.json();
        expect(trail.photos).toHaveLength(1110);
        expect(trail.photos[1109]).toBe(`/api/v1/assets/asset0000001109/file?share=${share}`);
        expect(trail.thumbnail).toBe(1075);
        expect(trail.expand.waypoints_via_trail[0].photos).toHaveLength(1110);
        expect(trail.expand.summit_logs_via_trail[0].photos).toHaveLength(1110);
        const pages = fetch.mock.calls.filter(([url]) => String(url).includes("_assets/records"));
        expect(pages).toHaveLength(9);
        expect(pages.map(([url]) => new URL(String(url)).searchParams.get("share"))).toEqual(Array(9).fill(share));
    });

    it.each([999, 0])("makes no extra queries for smaller or empty relations (%s links)", async (count) => {
        const fetch = pocketBaseTransport({ count, children: true });
        const response = await responseOverHttp(await getTrail(event(`/api/v1/trail/${trailId}`, fetch)));
        expect(response.status).toBe(200);
        expect((await response.json()).photos).toHaveLength(count);
        expect(fetch).toHaveBeenCalledOnce();
    });

    it.each(["trail listing", "list", "list listing"])("completes direct photos and the marked thumbnail in %s", async (endpoint) => {
        const fetch = pocketBaseTransport({ count: 1110 });
        const response = await responseOverHttp(await (endpoint === "list"
            ? getList(event(`/api/v1/list/${trailId}?expand=trails`, fetch))
            : endpoint === "list listing" ? getLists(event("/api/v1/list?expand=trails", fetch))
            : getTrails(event("/api/v1/trail", fetch))));
        expect(response.status).toBe(200);
        const body = await response.json();
        const [trail] = endpoint === "list" ? body.expand.trails : endpoint === "list listing" ? body.items[0].expand.trails : body.items;
        expect(trail.photos).toHaveLength(1110);
        expect(trail.thumbnail).toBe(1075);
        expect(fetch).toHaveBeenCalledTimes(4);
    });

    it("keeps all existing photos when an update response becomes the editor's saved baseline", async () => {
        const fetch = pocketBaseTransport({ count: 1110 });
        const response = await responseOverHttp(await updateTrail(event(`/api/v1/trail/${trailId}`, fetch, "owner-token", "POST")));
        expect(response.status).toBe(200);
        const trail = await response.json();
        expect(trail.photos).toHaveLength(1110);
        expect(trail.thumbnail).toBe(1075);
        expect(fetch).toHaveBeenCalledTimes(4);
    });

    it("surfaces a later page's access failure instead of returning a truncated photo list", async () => {
        const fetch = pocketBaseTransport({ count: 1110, rejectPage: 2 });
        const response = await responseOverHttp(await getTrail(event(`/api/v1/trail/${trailId}`, fetch)));
        expect(response.status).toBe(403);
        expect(await response.json()).toMatchObject({ message: "Read denied" });
    });
});

it("preserves the created trail identity when the post-create record refresh fails", async () => {
    const fetch = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => init?.method === "POST"
        ? Response.json({ id: trailId, name: "Saved", author: "author000000001", photos: [] })
        : Response.json({ message: "Refresh unavailable" }, { status: 503 }));
    vi.stubGlobal("fetch", fetch);
    const form = new FormData(); form.append("name", "Saved");
    const requestEvent = event("/api/v1/trail/form", fetch);
    const response = await responseOverHttp(await createTrailForm({ ...requestEvent, request: new Request(requestEvent.url, { method: "PUT", body: form }) }));
    expect(response.status).toBe(503);
    expect(await response.json()).toMatchObject({ detail: { recordCommitted: true, savedRecord: { id: trailId, name: "Saved" } } });
    expect(fetch).toHaveBeenCalledTimes(2);
});
