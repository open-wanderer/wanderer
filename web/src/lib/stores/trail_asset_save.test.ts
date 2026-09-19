import { afterEach, describe, expect, it, vi } from "vitest";
import type { AuthRecord } from "pocketbase";
import type { Trail } from "$lib/models/trail";
import type { Waypoint } from "$lib/models/waypoint";
import { trails_create, trails_update } from "./trail_store";
import { currentUser } from "./user_store";

const user: AuthRecord = {
    id: "test-user", collectionId: "users", collectionName: "users", actor: "actor",
};
const assetId = (id: string) => id.padEnd(15, "0");
const photo = (id: string) => `/api/v1/assets/${assetId(id)}/file`;
function trail(photos: string[] = []): Trail {
    return {
        id: "trail-id", name: "Trail", author: "actor", public: false,
        completed: true, like_count: 0, tags: [], photos,
        expand: { waypoints_via_trail: [] },
    };
}

afterEach(() => {
    vi.unstubAllGlobals();
    currentUser.set(null);
});

describe("trail photo saves", () => {
    it("retries a partial update without uploading saved local files or relinking successful groups", async () => {
        const baseline = trail([photo("keep"), photo("remove")]);
        const pending = trail([photo("keep")]);
        pending._assetLinks = ["existing"];
        pending._assetPluginLinks = [
            { pluginId: "first", assetIds: ["first-photo"] },
            { pluginId: "second", assetIds: ["second-photo"] },
        ];
        const files = [new File(["photo"], "photo.jpg", { type: "image/jpeg" })];
        const pluginRequests: string[] = [];
        const deleted: string[] = [];
        let uploads = 0;
        let assetLinkRequests = 0;
        const storedPhotos = [...baseline.photos];
        const request = vi.fn(async (url: RequestInfo | URL, config?: RequestInit) => {
            const path = String(url);
            if (path.startsWith("/api/v1/trail/form/")) {
                return Response.json(trail([...storedPhotos]));
            }
            if (path === "/api/v1/assets" && config?.method === "PUT") {
                const form = config.body as FormData;
                const id = form.has("files") ? "local" : "existing";
                if (form.has("files")) uploads++;
                else assetLinkRequests++;
                storedPhotos.push(photo(id));
                return Response.json([{ id: assetId(id) }]);
            }
            if (path.startsWith("/api/v1/plugins/assets/")) {
                const plugin = path.includes("/first/") ? "first" : "second";
                pluginRequests.push(plugin);
                if (plugin === "second" && pluginRequests.filter((p) => p === "second").length === 1) {
                    return Response.json({ imported: [], omitted: [{ assetId: "second-photo", reason: "download_failed" }] });
                }
                storedPhotos.push(photo(plugin));
                return Response.json({ imported: [{ asset: { id: assetId(plugin) } }], omitted: [] });
            }
            if (config?.method === "DELETE") {
                deleted.push(path);
                storedPhotos.splice(storedPhotos.indexOf(photo("remove")), 1);
                return Response.json({ acknowledged: true });
            }
            if (path.endsWith("/thumbnail")) return Response.json({ acknowledged: true });
            if (path.startsWith("/api/v1/trail/trail-id?")) return Response.json(trail([...storedPhotos]));
            throw new Error(`Unexpected request: ${path}`);
        });
        vi.stubGlobal("fetch", request);

        await expect(trails_update(baseline, pending, files)).rejects.toMatchObject({ message: "asset_import_failed" });
        expect(uploads).toBe(1);
        expect(files).toEqual([]);
        expect(pending._assetLinks).toBeUndefined();
        expect(pending._assetPluginLinks).toEqual([{ pluginId: "second", assetIds: ["second-photo"] }]);
        expect(baseline.photos).toEqual([photo("keep"), photo("remove"), photo("local"), photo("existing"), photo("first")]);
        expect(pending.photos).toEqual([photo("keep"), photo("local"), photo("existing"), photo("first")]);
        expect(deleted).toEqual([]);

        const result = await trails_update(baseline, pending, files);
        expect(uploads).toBe(1);
        expect(assetLinkRequests).toBe(1);
        expect(pluginRequests).toEqual(["first", "second", "second"]);
        expect(result.photos).toEqual([photo("keep"), photo("local"), photo("existing"), photo("first"), photo("second")]);
        expect(deleted).toEqual([`/api/v1/assets/${assetId("remove")}?trail=trail-id`]);
    });

    it("does not reupload waypoint photos when removing an old photo fails after attachment", async () => {
        currentUser.set({ id: "test-user", collectionId: "users", collectionName: "users", actor: "actor", password: "unused" });
        const oldWaypoint: Waypoint = {
            id: "waypoint-id", trail: "trail-id", author: "actor", lat: 47, lon: 8,
            photos: [photo("remove")],
        };
        const pendingWaypoint: Waypoint = {
            ...oldWaypoint, photos: [],
            _photos: [new File(["photo"], "photo.jpg", { type: "image/jpeg" })],
            _assetLinks: ["existing"],
            _assetPluginLinks: [{ pluginId: "immich", assetIds: ["remote"] }],
        };
        const baseline = trail();
        baseline.expand!.waypoints_via_trail = [oldWaypoint];
        const pending = trail();
        pending.expand!.waypoints_via_trail = [pendingWaypoint];
        const storedPhotos = [...oldWaypoint.photos];
        let uploads = 0;
        let linkRequests = 0;
        let pluginRequests = 0;
        let deleteAttempts = 0;
        const request = vi.fn(async (url: RequestInfo | URL, config?: RequestInit) => {
            const path = String(url);
            if (path === "/api/v1/waypoint/waypoint-id") {
                return Response.json({ id: oldWaypoint.id, trail: oldWaypoint.trail, author: "actor", lat: 47, lon: 8 });
            }
            if (path === "/api/v1/assets" && config?.method === "PUT") {
                const isFile = (config.body as FormData).has("files");
                if (isFile) uploads++;
                else linkRequests++;
                const id = isFile ? "local" : "existing";
                storedPhotos.push(photo(id));
                return Response.json([{ id: assetId(id) }]);
            }
            if (path.includes("/import-to-target")) {
                pluginRequests++;
                storedPhotos.push(photo("remote"));
                return Response.json({ imported: [{ asset: { id: assetId("remote") } }], omitted: [] });
            }
            if (path === `/api/v1/assets/${assetId("remove")}?waypoint=waypoint-id` && config?.method === "DELETE") {
                deleteAttempts++;
                if (deleteAttempts === 1) return Response.json({ message: "delete-failed" }, { status: 503 });
                storedPhotos.splice(storedPhotos.indexOf(photo("remove")), 1);
                return Response.json({ acknowledged: true });
            }
            if (path.startsWith("/api/v1/trail/")) {
                return Response.json({ ...trail(), expand: { waypoints_via_trail: [{ ...oldWaypoint, photos: [...storedPhotos] }] } });
            }
            throw new Error(`Unexpected request: ${path}`);
        });
        vi.stubGlobal("fetch", request);

        await expect(trails_update(baseline, pending)).rejects.toMatchObject({ message: "delete-failed" });
        expect(pendingWaypoint._photos).toBeUndefined();
        expect(pendingWaypoint._assetLinks).toBeUndefined();
        expect(pendingWaypoint._assetPluginLinks).toBeUndefined();
        expect(pendingWaypoint.photos).toEqual([photo("local"), photo("existing"), photo("remote")]);
        expect(baseline.expand!.waypoints_via_trail![0].photos).toContain(photo("remove"));

        const result = await trails_update(baseline, pending);
        expect(uploads).toBe(1);
        expect(linkRequests).toBe(1);
        expect(pluginRequests).toBe(1);
        expect(deleteAttempts).toBe(2);
        expect(result.expand!.waypoints_via_trail![0].photos).toEqual([photo("local"), photo("existing"), photo("remote")]);
    });

    it("reports omissions from two photo waypoints once for the whole save", async () => {
        const pending = trail();
        pending.expand!.waypoints_via_trail = ["first", "second"].map((id): Waypoint => ({
            id, name: id, lat: 47, lon: 8, author: "actor", photos: [],
            _assetPluginLinks: [{ pluginId: "immich", assetIds: [`${id}-good`, `${id}-missing`] }],
        }));
        const savedWaypoints: Waypoint[] = [];
        const onOmitted = vi.fn();
        const request = vi.fn(async (url: RequestInfo | URL, config?: RequestInit) => {
            const path = String(url);
            if (path.startsWith("/api/v1/trail/form?")) return Response.json(trail());
            if (path === "/api/v1/waypoint") {
                const saved = JSON.parse(String(config?.body)) as Waypoint;
                savedWaypoints.push(saved);
                return Response.json(saved);
            }
            if (path.includes("/import-to-target")) {
                expect(onOmitted).not.toHaveBeenCalled();
                const { waypointId } = JSON.parse(String(config?.body));
                return Response.json({
                    imported: [{ asset: { id: `${waypointId}-good` } }],
                    omitted: [{ assetId: `${waypointId}-missing`, reason: "not_found" }],
                });
            }
            if (path.endsWith("/thumbnail")) return Response.json({ acknowledged: true });
            if (path.startsWith("/api/v1/trail/trail-id?")) {
                return Response.json({ ...trail(), expand: { waypoints_via_trail: savedWaypoints } });
            }
            throw new Error(`Unexpected request: ${path}`);
        });

        await trails_create(pending, [], null, request, user, onOmitted);
        expect(onOmitted).toHaveBeenCalledOnce();
        expect(onOmitted).toHaveBeenCalledWith([
            { pluginId: "immich", assetId: "first-missing", reason: "not_found" },
            { pluginId: "immich", assetId: "second-missing", reason: "not_found" },
        ]);
        expect(savedWaypoints).toHaveLength(2);
    });
});
