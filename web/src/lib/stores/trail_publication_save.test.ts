import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AuthRecord } from "pocketbase";
import type { Trail } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";
import { assets_attach_to_target, assets_delete_removed, assets_set_trail_thumbnail } from "./asset_store";
import { waypoints_create, waypoints_delete } from "./waypoint_store";
import { summit_logs_create, summit_logs_delete } from "./summit_log_store";
import { trails_create, trails_update } from "./trail_store";

vi.mock("./waypoint_store", () => ({ waypoints_create: vi.fn(), waypoints_update: vi.fn(), waypoints_delete: vi.fn() }));
vi.mock("./summit_log_store", () => ({ summit_logs_create: vi.fn(), summit_logs_update: vi.fn(), summit_logs_delete: vi.fn() }));
vi.mock("./asset_store", () => ({ assets_attach_to_target: vi.fn(), assets_delete_removed: vi.fn(), assets_set_trail_thumbnail: vi.fn(), has_asset_attachments: vi.fn() }));

const user: AuthRecord = { id: "user", actor: "actor", collectionId: "users", collectionName: "users" };
const makeTrail = (isPublic = false): Trail => ({ id: "trail", name: "Trail", author: "actor", public: isPublic, completed: false, like_count: 0, tags: [], photos: [], expand: { waypoints_via_trail: [], summit_logs_via_trail: [] } });
let events: string[];
let published: boolean;
let failPublication: boolean;
let failRefreshAfterPublication: boolean;

const request = vi.fn(async (url: RequestInfo | URL, options?: RequestInit) => {
    const path = String(url);
    if (path.includes("/form")) {
        events.push("save-private");
        expect((options!.body as FormData).get("public")).toBe("false");
        return Response.json(makeTrail());
    }
    if (path.endsWith("/publication")) {
        events.push("publish");
        published = !failPublication;
        return Response.json({ id: "job", trailId: "trail", status: failPublication ? "failed" : "completed", error: failPublication ? "asset_publish_failed" : undefined, total: 0, processed: 0, failed: 0 });
    }
    if (path.startsWith("/api/v1/trail/trail?")) {
        if (published && failRefreshAfterPublication) return Response.json({ message: "refresh-failed" }, { status: 503 });
        return Response.json(makeTrail(published));
    }
    throw new Error(`Unexpected request: ${path}`);
});

beforeEach(() => {
    vi.resetAllMocks();
    events = [];
    published = false;
    failPublication = false;
    failRefreshAfterPublication = false;
    vi.stubGlobal("fetch", request);
    vi.mocked(assets_attach_to_target).mockImplementation(async (options) => { events.push("attach-root"); return options.existingPhotos ?? []; });
    vi.mocked(assets_delete_removed).mockImplementation(async () => { events.push("delete-photos"); });
    vi.mocked(assets_set_trail_thumbnail).mockImplementation(async () => { events.push("thumbnail"); });
    vi.mocked(waypoints_create).mockImplementation(async (waypoint) => { events.push("waypoint-photos"); return { ...waypoint, id: "waypoint" }; });
    vi.mocked(waypoints_delete).mockImplementation(async () => { events.push("delete-waypoint"); return { acknowledged: true }; });
    vi.mocked(summit_logs_create).mockImplementation(async (log) => { events.push("summit-photos"); return { ...log, id: "log" }; });
    vi.mocked(summit_logs_delete).mockImplementation(async () => { events.push("delete-log"); return { acknowledged: true }; });
});
afterEach(() => vi.unstubAllGlobals());

describe("publication after complete trail save", () => {
    it.each(["create", "update"])("preserves routing provenance while saving a private draft before publication (%s)", async (mode) => {
        const pending = makeTrail(true);
        pending.routing_provenance = [
            { pluginId: "brouter", routingMode: "segment", profileKey: "trekking" },
            null,
            { routeTopology: "closed_loop", roundTripTargetMeters: 10000 },
        ];

        const saved = mode === "create"
            ? await trails_create(pending, [], null, request, user)
            : await trails_update(makeTrail(), pending);

        const formRequest = request.mock.calls.find(([url]) => String(url).includes("/form"));
        const body = formRequest?.[1]?.body as FormData;
        expect(JSON.parse(body.get("routing_provenance") as string)).toEqual(pending.routing_provenance);
        expect(body.has("routing_provenance[0][pluginId]")).toBe(false);
        expect(saved.public).toBe(true);
        expect(events.at(-1)).toBe("publish");
    });

    it("creates a private draft and finishes root, waypoint, summit-log photos and thumbnail before publishing", async () => {
        const pending = makeTrail(true);
        pending.expand!.waypoints_via_trail = [{ author: "actor", lat: 47, lon: 8, photos: [] }];
        pending.expand!.summit_logs_via_trail = [{ author: "actor", date: "2026-09-16", photos: [] }];
        const saved = await trails_create(pending, [], null, request, user);
        expect(saved.public).toBe(true);
        expect(events).toEqual(["save-private", "attach-root", "summit-photos", "waypoint-photos", "thumbnail", "publish"]);
    });

    it("finishes removed children/photos and thumbnail changes before publishing an existing trail", async () => {
        const baseline = makeTrail();
        baseline.photos = ["old-photo"];
        baseline.expand!.waypoints_via_trail = [{ id: "waypoint", author: "actor", lat: 47, lon: 8, photos: [] }];
        baseline.expand!.summit_logs_via_trail = [{ id: "log", author: "actor", date: "2026-09-16", photos: [] }];
        const saved = await trails_update(baseline, makeTrail(true));
        expect(saved.public).toBe(true);
        expect(events).toEqual(["delete-waypoint", "delete-log", "save-private", "attach-root", "delete-photos", "thumbnail", "publish"]);
    });

    it("also publishes a trail with no linked photos through the background endpoint", async () => {
        const saved = await trails_update(makeTrail(), makeTrail(true));
        expect(saved.public).toBe(true);
        expect(events.at(-1)).toBe("publish");
    });

    it("keeps the created private draft and consumed attachments when publication fails, so retry updates its ID", async () => {
        failPublication = true;
        const pending = makeTrail(true);
        const files = [new File(["photo"], "photo.jpg")];
        const failure = await trails_create(pending, files, null, request, user).catch((error) => error as APIError);
        expect(failure).toMatchObject({ message: "asset_publish_failed", detail: { savedTrail: { id: "trail", public: false } } });
        if (!(failure instanceof APIError)) throw new Error("Expected publication to fail");
        expect(files).toEqual([]);
        expect(pending.public).toBe(true);
        failPublication = false;
        const saved = await trails_update(failure.detail.savedTrail, pending, files);
        expect(saved.public).toBe(true);
        expect(request.mock.calls.filter(([url]) => String(url).startsWith("/api/v1/trail/form?"))).toHaveLength(1);
    });

    it("keeps a saved private baseline when an existing trail's publication fails", async () => {
        failPublication = true;
        await expect(trails_update(makeTrail(), makeTrail(true))).rejects.toMatchObject({ message: "asset_publish_failed", detail: { savedTrail: { id: "trail", public: false } } });
    });

    it.each(["create", "update"])("keeps a public baseline if the refresh after successful publication fails (%s)", async (mode) => {
        failRefreshAfterPublication = true;
        const pending = makeTrail(true);
        const operation = mode === "create"
            ? trails_create(pending, [], null, request, user)
            : trails_update(makeTrail(), pending);
        await expect(operation).rejects.toMatchObject({ message: "refresh-failed", detail: { savedTrail: { id: "trail", public: true } } });
        expect(events.filter((event) => event === "publish")).toHaveLength(1);
    });

    it("does not start another publication when editing an already-public trail", async () => {
        published = true;
        request.mockImplementationOnce(async (_url, options) => {
            expect((options!.body as FormData).get("public")).toBe("true");
            return Response.json(makeTrail(true));
        });
        const saved = await trails_update(makeTrail(true), makeTrail(true));
        expect(saved.public).toBe(true);
        expect(events).not.toContain("publish");
    });
});
