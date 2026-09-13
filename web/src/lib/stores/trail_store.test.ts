import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AuthRecord } from "pocketbase";
import type { Trail } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";
import type { SummitLog } from "$lib/models/summit_log";
import { APIError } from "$lib/util/api_util";
import { assets_attach_to_target } from "./asset_store";
import { waypoints_create, waypoints_update } from "./waypoint_store";
import { trails_create, trails_update, trailSaveErrorKey } from "./trail_store";

vi.mock("./waypoint_store", () => ({ waypoints_create: vi.fn(), waypoints_update: vi.fn(), waypoints_delete: vi.fn() }));
vi.mock("./asset_store", () => ({
    assets_attach_to_target: vi.fn(), assets_delete_removed: vi.fn(),
    assets_set_trail_thumbnail: vi.fn(), has_asset_attachments: vi.fn(),
}));

const user: AuthRecord = {
    id: "test-user", collectionId: "users", collectionName: "users", actor: "actor",
};

function waypoint(id?: string): Waypoint {
    return { id, name: "Photo waypoint", lat: 47, lon: 8, author: "actor", photos: [],
        _assetPluginLinks: [{ pluginId: "immich", assetIds: ["photo"] }] };
}
function trail(waypoints: Waypoint[] = []): Trail {
    return { id: "trail-id", name: "Trail", author: "actor", public: false,
        completed: true, like_count: 0, tags: [], photos: [], expand: { waypoints_via_trail: waypoints } };
}
function savedWaypoint(input: Waypoint, id = input.id!): Waypoint {
    const { _assetPluginLinks, ...saved } = input;
    return { ...saved, id, photos: ["/api/v1/assets/photo/file"] };
}

beforeEach(() => {
    vi.resetAllMocks();
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ message: "later-save-failed" }), { status: 400 })));
    vi.mocked(assets_attach_to_target).mockResolvedValue([]);
});

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("trail save retries after waypoint attachment failures", () => {
    it("does not recreate successful earlier waypoints when a later waypoint fails", async () => {
        const baseline = trail();
        const first = waypoint("first");
        const second = waypoint("second");
        const pending = trail([first, second]);
        const failure = new APIError(400, "asset_import_failed");
        vi.mocked(waypoints_create).mockResolvedValueOnce(savedWaypoint(first)).mockRejectedValueOnce(failure);

        await expect(trails_update(baseline, pending)).rejects.toBe(failure);
        expect(baseline.expand?.waypoints_via_trail?.map((wp) => wp.id)).toEqual(["first"]);
        expect(first._assetPluginLinks).toBeUndefined();
        expect(second._assetPluginLinks).toHaveLength(1);

        vi.mocked(waypoints_create).mockResolvedValueOnce(savedWaypoint(second));
        await expect(trails_update(baseline, pending)).rejects.toMatchObject({ message: "later-save-failed" });
        expect(vi.mocked(waypoints_create).mock.calls.map(([wp]) => wp.id)).toEqual(["first", "second", "second"]);
        expect(waypoints_update).not.toHaveBeenCalled();
    });

    it("retries a partly saved waypoint as an update and preserves its saved photos", async () => {
        const baseline = trail();
        const photoWaypoint = waypoint();
        const pending = trail([photoWaypoint]);
        const saved = savedWaypoint(photoWaypoint, "server-waypoint");
        photoWaypoint._photos = [new File(["saved"], "saved.jpg")];
        photoWaypoint._assetLinks = ["already-linked"];
        const remainingPluginLinks = [{ pluginId: "immich", assetIds: ["missing-photo"] }];
        const failure = new APIError(400, "asset_import_partial_failure", {
            savedWaypoint: saved,
            remainingAttachments: {
                _photos: undefined,
                _assetLinks: undefined,
                _assetPluginLinks: remainingPluginLinks,
            },
        });
        vi.mocked(waypoints_create).mockRejectedValueOnce(failure);

        await expect(trails_update(baseline, pending)).rejects.toBe(failure);
        expect(photoWaypoint.id).toBe("server-waypoint");
        expect(photoWaypoint.photos).toEqual(saved.photos);
        expect(photoWaypoint._assetPluginLinks).toEqual(remainingPluginLinks);
        expect(photoWaypoint._photos).toBeUndefined();
        expect(photoWaypoint._assetLinks).toBeUndefined();
        expect(baseline.expand?.waypoints_via_trail?.[0]._assetPluginLinks).toBeUndefined();

        vi.mocked(waypoints_update).mockResolvedValueOnce(saved);
        await expect(trails_update(baseline, pending)).rejects.toMatchObject({ message: "later-save-failed" });
        expect(waypoints_create).toHaveBeenCalledOnce();
        expect(waypoints_update).toHaveBeenCalledOnce();
        expect(vi.mocked(waypoints_update).mock.calls[0][0].id).toBe(saved.id);
        expect(vi.mocked(waypoints_update).mock.calls[0][1].photos).toEqual(saved.photos);
    });

    it("reports the existing new trail and completed waypoints so retry can update it", async () => {
        const first = waypoint("first");
        const second = waypoint("second");
        const pending = trail([first, second]);
        const files = [new File(["photo"], "photo.jpg")];
        const failure = new APIError(400, "asset_import_failed");
        const request = vi.fn(async () => new Response(JSON.stringify(trail()), { status: 200 }));
        vi.mocked(assets_attach_to_target).mockResolvedValueOnce(["/api/v1/assets/local/file"]);
        vi.mocked(waypoints_create).mockResolvedValueOnce(savedWaypoint(first)).mockRejectedValueOnce(failure);

        await expect(trails_create(pending, files, null, request, user)).rejects.toBe(failure);
        const saved = failure.detail.savedTrail as Trail;
        expect(saved.id).toBe("trail-id");
        expect(saved.expand?.waypoints_via_trail?.map((wp) => wp.id)).toEqual(["first"]);
        expect(saved.photos).toEqual(["/api/v1/assets/local/file"]);
        expect(pending.photos).toEqual(saved.photos);
        expect(files).toEqual([]);
        expect(request).toHaveBeenCalledOnce();

        vi.mocked(waypoints_create).mockResolvedValueOnce(savedWaypoint(second));
        await expect(trails_update(saved, pending)).rejects.toMatchObject({ message: "later-save-failed" });
        expect(vi.mocked(waypoints_create).mock.calls.map(([wp]) => wp.id)).toEqual(["first", "second", "second"]);
    });

    it("keeps partly attached summit-log photos out of the root trail", async () => {
        const pending = trail();
        const log: SummitLog = {
            id: "summit-log", date: "2026-09-13", author: "actor", photos: [],
            _assetPluginLinks: [{ pluginId: "immich", assetIds: ["log-photo"] }],
        };
        pending.expand!.summit_logs_via_trail = [log];
        const rootPhoto = "/api/v1/assets/root-photo/file";
        const logPhoto = "/api/v1/assets/summit-photo/file";
        const failure = new APIError(400, "asset_import_failed", {
            completedAttachments: { files: true, assetIds: true, pluginLinks: [] },
            attachedPhotos: [logPhoto],
        });
        vi.mocked(assets_attach_to_target)
            .mockResolvedValueOnce([rootPhoto])
            .mockRejectedValueOnce(failure);
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(trail()))
            .mockResolvedValueOnce(Response.json({ ...log, trail: "trail-id" }));

        await expect(trails_create(pending, [], null, request, user)).rejects.toBe(failure);
        expect(failure.detail.savedTrail.photos).toEqual([rootPhoto]);
        expect(pending.photos).toEqual([rootPhoto]);
        expect(vi.mocked(assets_attach_to_target).mock.calls[1][0].target).toEqual({
            trail: "trail-id", summit_log: "summit-log",
        });
    });

    it("consumes completed root attachments while retaining the failed selection", async () => {
        const pending = trail();
        pending._assetLinks = ["existing"];
        pending._assetPluginLinks = [
            { pluginId: "first-plugin", assetIds: ["imported"] },
            { pluginId: "second-plugin", assetIds: ["failed"] },
        ];
        const files = [new File(["photo"], "photo.jpg")];
        const attachedPhotos = ["/api/v1/assets/local/file", "/api/v1/assets/imported/file"];
        const failure = new APIError(400, "asset_import_failed", {
            completedAttachments: {
                files: true, assetIds: true,
                pluginLinks: [{ pluginId: "first-plugin", assetIds: ["imported"] }],
            },
            attachedPhotos,
        });
        vi.mocked(assets_attach_to_target).mockRejectedValueOnce(failure);
        const request = vi.fn(async () => new Response(JSON.stringify(trail()), { status: 200 }));

        await expect(trails_create(pending, files, null, request, user)).rejects.toBe(failure);
        expect(files).toEqual([]);
        expect(pending._assetLinks).toBeUndefined();
        expect(pending._assetPluginLinks).toEqual([{ pluginId: "second-plugin", assetIds: ["failed"] }]);
        expect(pending.photos).toEqual(attachedPhotos);
        expect(failure.detail.savedTrail.photos).toEqual(attachedPhotos);
    });

    it("retains the created trail ID when its first photo import fails", async () => {
        const pending = trail();
        pending.id = undefined;
        const failure = new APIError(400, "asset_import_failed");
        vi.mocked(assets_attach_to_target).mockRejectedValueOnce(failure);
        const request = vi.fn(async () => new Response(JSON.stringify(trail()), { status: 200 }));

        await expect(trails_create(pending, [], null, request, user)).rejects.toBe(failure);
        expect(failure.detail.savedTrail.id).toBe("trail-id");
        expect(waypoints_create).not.toHaveBeenCalled();
    });
});

describe("trail save error messages", () => {
    it.each(["asset_import_failed", "asset_import_partial_failure", "asset_import_cleanup_failed", "asset_publish_failed", "asset_publish_limit_reached", "asset_publish_photo_too_large"])("translates %s", (code) => {
        expect(trailSaveErrorKey(new APIError(400, code))).toBe(code);
    });
    it("uses the generic message for other errors", () => {
        expect(trailSaveErrorKey(new Error("network failed"))).toBe("error-saving-trail");
    });
});
