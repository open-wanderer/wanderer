import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import type { AssetLink } from "#lib/models/asset.ts";
import type { Trail } from "#lib/models/trail.ts";
import { describe, expect, it, vi } from "vitest";

vi.mock("#lib/server/category_preference_filter.ts", () => ({
    withTrailPreferencePocketBaseFilter: async (_event: RequestEvent, filter?: string) => filter,
}));

import { GET as getTrails } from "./trail/+server";
import { GET as getList } from "./list/[id]/+server";

const waypointPhotos = "waypoints_via_trail.waypoint_assets_via_waypoint.asset";
const summitPhotos = "summit_logs_via_trail.summit_log_assets_via_summit_log.asset";

function photoLink(id: string): AssetLink {
    return {
        asset: id,
        expand: { asset: {
            id, collectionId: "assets", collectionName: "assets", author: "author000000001",
            type: "photo", storage_mode: "link_private",
        } },
    };
}

// Simulate PocketBase's requested relation expansion at the transport boundary.
function expandedTrail(expand = ""): Trail {
    const paths = new Set(expand.split(","));
    const relations: NonNullable<Trail["expand"]> = {};
    if (paths.has("trail_assets_via_trail.asset")) {
        relations.trail_assets_via_trail = [photoLink("trailphoto00001")];
    }
    if (paths.has(waypointPhotos)) {
        relations.waypoints_via_trail = [{
            id: "waypoint0000001", author: "author000000001", lat: 46, lon: 7, photos: [],
            expand: { waypoint_assets_via_waypoint: [photoLink("waypointphoto01")] },
        }];
    }
    if (paths.has(summitPhotos)) {
        relations.summit_logs_via_trail = [{
            id: "summitlog000001", author: "author000000001", date: "2026-08-01 00:00:00.000Z", photos: [],
            expand: { summit_log_assets_via_summit_log: [photoLink("summitphoto0001")] },
        }];
    }
    return {
        id: "trail0000000001", name: "Trail", author: "author000000001", public: false,
        completed: false, tags: [], like_count: 0, photos: [], expand: relations,
    };
}

function endpointEvent(path: string): RequestEvent {
    return {
        params: { id: "list00000000001" },
        url: new URL(`http://a.example${path}`),
        fetch: vi.fn(),
        locals: { pb: {
            collection: vi.fn(() => ({
                getList: vi.fn(async (_page, _perPage, options) => ({ items: [expandedTrail(options.expand)] })),
                getFullList: vi.fn(async (options) => [expandedTrail(options.expand)]),
            })),
            send: vi.fn(async (path: string) => {
                const expands = new URL(path, "http://a.example").searchParams.get("expand") ?? "";
                return {
                    id: "list00000000001",
                    expand: { trails: [expandedTrail(expands.split(",").map((p) => p.replace(/^trails\./, "")).join(","))] },
                };
            }),
        } },
    } as unknown as RequestEvent;
}

describe("trail and list photo expansions", () => {
    it.each([undefined, -1])("returns direct photos without expanding child records on trail listing (perPage %s)", async (perPage) => {
        const response = await responseOverHttp(await getTrails(endpointEvent(`/api/v1/trail${perPage ? `?perPage=${perPage}` : ""}`)));
        expect(response.status).toBe(200);
        const [trail] = (await response.json()).items;
        expect(trail.photos).toEqual(["/api/v1/assets/trailphoto00001/file"]);
        expect(trail.expand.waypoints_via_trail).toBeUndefined();
        expect(trail.expand.summit_logs_via_trail).toBeUndefined();
    });

    it("returns only direct trail photos for a list unless child records were requested", async () => {
        const response = await responseOverHttp(await getList(endpointEvent("/api/v1/list/list00000000001?expand=trails")));
        expect(response.status).toBe(200);
        const [trail] = (await response.json()).expand.trails;
        expect(trail.photos).toEqual(["/api/v1/assets/trailphoto00001/file"]);
        expect(trail.expand.waypoints_via_trail).toBeUndefined();
        expect(trail.expand.summit_logs_via_trail).toBeUndefined();
    });

    it.each(["trails", "list"])("includes child photos when only their parent relations are requested on %s", async (endpoint) => {
        const prefix = endpoint === "list" ? "trails." : "";
        const path = endpoint === "list" ? "/api/v1/list/list00000000001" : "/api/v1/trail";
        const event = endpointEvent(`${path}?${new URLSearchParams({ expand: `${prefix}waypoints_via_trail,${prefix}summit_logs_via_trail` })}`);
        const response = await responseOverHttp(await (endpoint === "list" ? getList(event) : getTrails(event)));
        expect(response.status).toBe(200);
        const body = await response.json();
        const [trail] = endpoint === "list" ? body.expand.trails : body.items;
        expect(trail.expand.waypoints_via_trail[0].photos).toEqual(["/api/v1/assets/waypointphoto01/file"]);
        expect(trail.expand.summit_logs_via_trail[0].photos).toEqual(["/api/v1/assets/summitphoto0001/file"]);
    });
});
