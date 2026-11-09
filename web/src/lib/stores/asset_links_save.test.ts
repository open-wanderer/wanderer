import { afterEach, describe, expect, it, vi } from "vitest";
import type { Asset, AssetLink } from "#lib/models/asset.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import { Trail } from "#lib/models/trail.ts";
import type { Waypoint } from "#lib/models/waypoint.ts";
import { summit_logs_update } from "./summit_log_store";
import { shownTrailPhotos, trails_create, trails_update } from "./trail_store";
import { currentUser } from "./user_store";
import { waypoints_update } from "./waypoint_store";

const user = {
    id: "test-user", collectionId: "users", collectionName: "users", actor: "actor", password: "unused",
};
const id = (name: string) => name.padEnd(15, "0");
const photo = (name: string) => `/api/v1/assets/${id(name)}/file`;

function asset(assetId: string): Asset {
    return {
        id: assetId, collectionId: "assets", collectionName: "assets",
        type: "photo", author: "actor", storage_mode: "link_private",
    };
}

function links(...names: string[]): AssetLink[] {
    return names.map((name) => ({ asset: id(name), expand: { asset: asset(id(name)) } }));
}

function savedTrail(photos: string[] = []): Trail {
    return {
        id: "trail-id", name: "Trail", author: "actor", public: false,
        completed: true, like_count: 0, tags: [], photos, expand: {},
    };
}

// A trail whose trail, waypoint and summit log have two photos each.
function original(): Trail {
    return {
        ...savedTrail([photo("trail1"), photo("trail2")]), id: "original",
        expand: {
            trail_assets_via_trail: links("trail1", "trail2"),
            waypoints_via_trail: [{
                id: "waypoint", lat: 47, lon: 8, author: "actor", photos: [photo("waypoint1"), photo("waypoint2")],
                expand: { waypoint_assets_via_waypoint: links("waypoint1", "waypoint2") },
            }],
            summit_logs_via_trail: [{
                id: "summit-log", date: "2026-09-13", author: "actor", photos: [photo("log1"), photo("log2")],
                expand: { summit_log_assets_via_summit_log: links("log1", "log2") },
            }],
        },
    };
}

function savedWaypoint(): Waypoint {
    return { id: "waypoint", trail: "trail-id", lat: 47, lon: 8, author: "actor", photos: [] };
}

function savedSummitLog(): SummitLog {
    return { id: "summit-log", trail: "trail-id", date: "2026-09-13", author: "actor", photos: [] };
}

// Records which assets are linked to or removed from the trail, a waypoint or
// a summit log. The first `failedLinks` link requests fail.
function fakeServer(failedLinks = 0, copiedPhotos: string[] = []) {
    const linked: Record<string, string[]> = {};
    const removed: string[] = [];
    const request = vi.fn(async (url: RequestInfo | URL, config?: RequestInit) => {
        const path = String(url);
        if (path.startsWith("/api/v1/federation/")) return Response.json({ status: "ok" });
        const method = config?.method ?? "GET";
        if (path.startsWith("/api/v1/trail/form")) return Response.json(savedTrail());
        // Waypoint and summit log records have no photos field; their photos
        // are asset links.
        if (path.startsWith("/api/v1/summit-log/form")) {
            const form = config!.body as FormData;
            return Response.json({ id: form.get("id"), date: form.get("date"), author: "actor", trail: "trail-id" });
        }
        if (path.startsWith("/api/v1/waypoint")) {
            const { photos: _, ...waypoint } = JSON.parse(String(config!.body));
            return Response.json(waypoint);
        }
        if (path === "/api/v1/assets" && method === "PUT") {
            if (failedLinks > 0) {
                failedLinks--;
                return Response.json({ message: "link-failed" }, { status: 500 });
            }
            const form = config!.body as FormData;
            const target = ["summit_log", "waypoint", "trail"].find((key) => form.has(key))!;
            const assetIds = form.getAll("assetIds").map(String);
            const assets = assetIds.map((assetId) => copiedPhotos.includes(assetId)
                ? { ...asset(id("copy" + assetId.replace(/0+$/, ""))), sourceAssetId: assetId }
                : asset(assetId));
            linked[target] = [...(linked[target] ?? []), ...assets.map((asset) => asset.id)];
            return Response.json(assets);
        }
        if (path.startsWith("/api/v1/assets/") && method === "DELETE") {
            removed.push(path);
            return Response.json({ acknowledged: true });
        }
        if (path.endsWith("/thumbnail")) return Response.json({ acknowledged: true });
        if (path.startsWith("/api/v1/trail/trail-id?")) return Response.json(savedTrail());
        throw new Error(`Unexpected request: ${method} ${path}`);
    });
    vi.stubGlobal("fetch", request);
    currentUser.set(user);
    return { request, linked, removed };
}

afterEach(() => {
    vi.unstubAllGlobals();
    currentUser.set(null);
});

describe("photos of a duplicate", () => {
    it("saves copied photos from other authors on the trail, waypoint and summit log", async () => {
        const server = fakeServer(0, [id("trail1"), id("waypoint1"), id("log1")]);
        const copy = Trail.from(original(), "actor", {
            trailPhotos: true, waypointPhotos: true, summitLogs: true, summitLogPhotos: true,
        });
        copy.photos = [photo("trail1")];
        const [waypoint] = copy.expand!.waypoints_via_trail!;
        const [log] = copy.expand!.summit_logs_via_trail!;
        waypoint.photos = [photo("waypoint1")];
        log.photos = [photo("log1")];

        await trails_create(copy, [], null, server.request, user);

        expect(server.linked).toEqual({ trail: [id("copytrail1")], waypoint: [id("copywaypoint1")], summit_log: [id("copylog1")] });
        expect(copy.photos).toEqual([photo("copytrail1")]);
        expect(copy.expand!.waypoints_via_trail![0].photos).toEqual([photo("copywaypoint1")]);
        expect(copy.expand!.summit_logs_via_trail![0].photos).toEqual([photo("copylog1")]);
    });

    it("links only the copied photos the editor still shows, and photos linked by ID (duplicated photos)", async () => {
        const server = fakeServer();
        const copy = Trail.from(original(), "actor", {
            trailPhotos: true, waypointPhotos: true, summitLogs: true, summitLogPhotos: true,
        });
        const [waypoint] = copy.expand!.waypoints_via_trail!;
        const [log] = copy.expand!.summit_logs_via_trail!;
        // Removing a photo in the editor or a dialog only takes it out of `photos`.
        copy.photos = [photo("trail1")];
        waypoint.photos = [photo("waypoint1")];
        log.photos = [photo("log1")];
        // A duplicated photo linked by ID, as the dialogs add it on save.
        log._assetLinks = [...log._assetLinks!, id("linked")];
        log.photos = [...(log.photos ?? []), photo("linked")];

        await trails_create(copy, [], null, server.request, user);

        expect(server.linked).toEqual({
            trail: [id("trail1")],
            summit_log: [id("log1"), id("linked")],
            waypoint: [id("waypoint1")],
        });
    });

    it.each([
        ["waypoint", "waypoint"],
        ["summit log", "summit_log"],
    ])("links only the photos still shown when a copied %s is saved again", async (_, target) => {
        const server = fakeServer();
        // Saved before its photos were linked, with the second copied photo removed.
        const copied = { photos: [photo("copied1")], _assetLinks: [id("copied1"), id("copied2")] };

        if (target === "waypoint") {
            await waypoints_update(savedWaypoint(), { ...savedWaypoint(), ...copied });
        } else {
            await summit_logs_update(savedSummitLog(), { ...savedSummitLog(), ...copied });
        }

        expect(server.linked).toEqual({ [target]: [id("copied1")] });
    });
});

describe("photo linked by ID (duplicated photo) dropped after a failed save", () => {
    it("is not removed from the trail, which never had it", async () => {
        const server = fakeServer(1);
        const baseline = savedTrail([photo("keep")]);
        // What the editor adds on save for a duplicated photo linked by ID.
        const pending = { ...savedTrail([photo("keep"), photo("linked")]), _assetLinks: [id("linked")] };

        await expect(trails_update(baseline, pending)).rejects.toMatchObject({ message: "link-failed" });
        // The editor shows the photo as pending card again, and the user drops it.
        pending.photos = shownTrailPhotos(pending);
        await trails_update(baseline, pending);

        expect(server.linked).toEqual({});
        expect(server.removed).toEqual([]);
    });

    it.each(["waypoint", "summit log"])("is not removed from a %s, which never had it", async (kind) => {
        const server = fakeServer(1);
        const picked = { photos: [photo("linked")], _assetLinks: [id("linked")] };
        // The user removes the photo in the dialog; its ID stays in `_assetLinks`.
        const dropped = { photos: [], _assetLinks: [id("linked")] };

        if (kind === "waypoint") {
            const failure = await waypoints_update(savedWaypoint(), { ...savedWaypoint(), ...picked }).catch((error) => error);
            await waypoints_update(failure.detail.savedWaypoint, { ...savedWaypoint(), ...dropped });
        } else {
            const failure = await summit_logs_update(savedSummitLog(), { ...savedSummitLog(), ...picked }).catch((error) => error);
            await summit_logs_update(failure.detail.savedSummitLog, { ...savedSummitLog(), ...dropped });
        }

        expect(server.linked).toEqual({});
        expect(server.removed).toEqual([]);
    });
});
