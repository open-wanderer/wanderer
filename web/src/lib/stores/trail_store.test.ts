import type { Trail, TrailFilter } from "#lib/models/trail.ts";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { trails_create, trails_search_filter, trails_update } from "./trail_store";
import type { Waypoint } from "#lib/models/waypoint.ts";
import { APIError } from "#lib/util/api_util.ts";
import { assets_attach_to_target } from "./asset_store";
import { waypoints_create, waypoints_update } from "./waypoint_store";

vi.mock("./waypoint_store", () => ({ waypoints_create: vi.fn(), waypoints_update: vi.fn(), waypoints_delete: vi.fn() }));
vi.mock("./asset_store", () => ({ assets_attach_to_target: vi.fn(), assets_delete_removed: vi.fn(), assets_set_trail_thumbnail: vi.fn(), has_asset_attachments: vi.fn() }));


async function requestedFilter(near: TrailFilter["near"]): Promise<string> {
    const filter: TrailFilter = {
        q: "",
        category: [],
        subcategory: [],
        tags: [],
        difficulty: [],
        near,
        distanceMin: 0,
        distanceMax: 100000,
        distanceLimit: 100000,
        elevationGainMin: 0,
        elevationGainMax: 10000,
        elevationGainLimit: 10000,
        elevationLossMin: 0,
        elevationLossMax: 10000,
        elevationLossLimit: 10000,
        sort: "created",
        sortOrder: "-",
    };
    const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) =>
        Response.json({ hits: [], page: 1, totalPages: 0 }),
    );

    await trails_search_filter(filter, 1, 20, request);

    expect(request).toHaveBeenCalledOnce();
    const [url, options] = request.mock.calls[0];
    expect(url).toBe("/api/v1/search/trails");
    expect(options?.method).toBe("POST");
    return JSON.parse(String(options?.body)).options.filter;
}

describe("trail search radius", () => {
    it.each([
        { lat: 47, lon: 8, radius: 2000 },
        { lat: 0, lon: 8, radius: 2000 },
        { lat: 47, lon: 0, radius: 2000 },
        { lat: 0, lon: 0, radius: 2000 },
        { lat: -47, lon: -8, radius: 5000 },
        { lat: -90, lon: -180, radius: 2000 },
        { lat: 90, lon: 180, radius: 1 },
    ])("applies exactly one radius at ($lat, $lon) with $radius metres", async (near) => {
        const filter = await requestedFilter(near);

        expect(filter.match(/_geoRadius\([^)]*\)/g)).toEqual([
            `_geoRadius(${near.lat}, ${near.lon}, ${near.radius})`,
        ]);
    });

    it.each([undefined, NaN, Infinity, -Infinity, -90.01, 90.01])(
        "omits the radius for invalid latitude %s",
        async (lat) => {
            const filter = await requestedFilter({ lat, lon: 8, radius: 2000 });

            expect(filter).not.toContain("_geoRadius");
            expect(filter).toContain("distance >= 0");
        },
    );

    it.each([undefined, NaN, Infinity, -Infinity, -180.01, 180.01])(
        "omits the radius for invalid longitude %s",
        async (lon) => {
            const filter = await requestedFilter({ lat: 47, lon, radius: 2000 });

            expect(filter).not.toContain("_geoRadius");
            expect(filter).toContain("distance >= 0");
        },
    );

    it.each([0, -1, NaN, Infinity, -Infinity])(
        "omits invalid radius %s",
        async (radius) => {
            const filter = await requestedFilter({ lat: 47, lon: 8, radius });

            expect(filter).not.toContain("_geoRadius");
            expect(filter).toContain("distance >= 0");
        },
    );
});

describe("searchResultToTrailList", () => {
    it("keeps the moving time, which cards and lists prefer over the total duration", async () => {
        const { searchResultToTrailList } = await import("./trail_store");
        const { trailDisplayDuration } = await import("#lib/util/format_util.ts");
        const hit = {
            id: "trail000000001", name: "Ridge", author: "actor", author_name: "alice", author_avatar: "",
            created: 0, date: 0, public: true, completed: false, difficulty: 1, distance: 5000,
            duration: 3432, moving_duration: 1800, elevation_gain: 300, elevation_loss: 300,
            _geo: { lat: 47, lng: 11 }, tags: [],
        };

        const [trail] = await searchResultToTrailList([hit] as any);

        expect(trail.moving_duration).toBe(1800);
        expect(trailDisplayDuration(trail)).toBe(1800);
    });
});

const retryUser = { id: "test-user", collectionId: "users", collectionName: "users", actor: "actor" };
function retryWaypoint(id?: string): Waypoint {
    return { id, name: "Photo waypoint", lat: 47, lon: 8, author: "actor", photos: [], _assetLinks: ["selected0000001"] };
}
function retryTrail(waypoints: Waypoint[] = []): Trail {
    return { id: "trail-id", name: "Trail", author: "actor", public: false, completed: true, like_count: 0, tags: [], photos: [], expand: { waypoints_via_trail: waypoints } };
}
function savedRetryWaypoint(input: Waypoint, id = input.id!): Waypoint {
    const { _assetLinks, ...saved } = input;
    return { ...saved, id, photos: ["/api/v1/assets/photo000000001/file"] };
}
beforeEach(() => {
    vi.resetAllMocks();
    vi.stubGlobal("fetch", vi.fn(async () => Response.json({ message: "later-save-failed" }, { status: 400 })));
    vi.mocked(assets_attach_to_target).mockResolvedValue([]);
});
afterEach(() => vi.unstubAllGlobals());


describe("trail save retries after waypoint attachment failures", () => {
    it("does not recreate successful earlier waypoints when a later waypoint fails", async () => {
        const baseline = retryTrail();
        const first = retryWaypoint("first");
        const second = retryWaypoint("second");
        const pending = retryTrail([first, second]);
        const failure = new APIError(400, "asset_import_failed");
        vi.mocked(waypoints_create).mockResolvedValueOnce(savedRetryWaypoint(first)).mockRejectedValueOnce(failure);

        await expect(trails_update(baseline, pending)).rejects.toBe(failure);
        expect(baseline.expand?.waypoints_via_trail?.map((wp) => wp.id)).toEqual(["first"]);
        expect(first._assetLinks).toBeUndefined();
        expect(second._assetLinks).toHaveLength(1);

        vi.mocked(waypoints_create).mockResolvedValueOnce(savedRetryWaypoint(second));
        await expect(trails_update(baseline, pending)).rejects.toMatchObject({ message: "later-save-failed" });
        expect(vi.mocked(waypoints_create).mock.calls.map(([wp]) => wp.id)).toEqual(["first", "second", "second"]);
        expect(waypoints_update).not.toHaveBeenCalled();
    });

    it("retries a partly saved waypoint as an update and preserves its saved photos", async () => {
        const baseline = retryTrail();
        const photoWaypoint = retryWaypoint();
        const pending = retryTrail([photoWaypoint]);
        const saved = savedRetryWaypoint(photoWaypoint, "server-waypoint");
        photoWaypoint._photos = [new File(["saved"], "saved.jpg")];
        photoWaypoint._assetLinks = ["already-linked"];
        const remainingLinks = ["missing-photo"];
        const failure = new APIError(400, "asset_import_partial_failure", {
            savedWaypoint: saved,
            remainingAttachments: {
                _photos: undefined,
                _assetLinks: remainingLinks,
            },
        });
        vi.mocked(waypoints_create).mockRejectedValueOnce(failure);

        await expect(trails_update(baseline, pending)).rejects.toBe(failure);
        expect(photoWaypoint.id).toBe("server-waypoint");
        expect(photoWaypoint.photos).toEqual(saved.photos);
        expect(photoWaypoint._assetLinks).toEqual(remainingLinks);
        expect(photoWaypoint._photos).toBeUndefined();
        expect(baseline.expand?.waypoints_via_trail?.[0]._assetLinks).toBeUndefined();

        vi.mocked(waypoints_update).mockResolvedValueOnce(saved);
        await expect(trails_update(baseline, pending)).rejects.toMatchObject({ message: "later-save-failed" });
        expect(waypoints_create).toHaveBeenCalledOnce();
        expect(waypoints_update).toHaveBeenCalledOnce();
        expect(vi.mocked(waypoints_update).mock.calls[0][0].id).toBe(saved.id);
        expect(vi.mocked(waypoints_update).mock.calls[0][1].photos).toEqual(saved.photos);
    });

    it("consumes completed root attachments while retaining the failed selection", async () => {
        const pending = retryTrail();
        pending._assetLinks = ["failed000000001"];
        const files = [new File(["photo"], "photo.jpg")];
        const attachedPhotos = ["/api/v1/assets/local0000000001/file"];
        const failure = new APIError(400, "asset_import_failed", {
            completedAttachments: { files: true, assetIds: false }, attachedPhotos,
        });
        vi.mocked(assets_attach_to_target).mockRejectedValueOnce(failure);
        const request = vi.fn(async () => Response.json(retryTrail()));
        await expect(trails_create(pending, files, null, request, retryUser)).rejects.toBe(failure);
        expect(files).toEqual([]);
        expect(pending._assetLinks).toEqual(["failed000000001"]);
        expect(pending.photos).toEqual(attachedPhotos);
        expect(failure.detail.savedTrail.photos).toEqual(attachedPhotos);
    });
});
