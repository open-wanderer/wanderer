import { afterEach, describe, expect, it, vi } from "vitest";
import type { AuthRecord } from "pocketbase";
import { Waypoint } from "$lib/models/waypoint";
import { currentUser } from "./user_store";
import { waypoints_create, waypoints_update } from "./waypoint_store";

vi.mock("./toast_store.svelte", () => ({ show_toast: vi.fn() }));
vi.mock("./user_store", async () => {
    const { writable } = await import("svelte/store");
    return { currentUser: writable(null) };
});

const user: AuthRecord = { id: "user", actor: "actor", collectionId: "users", collectionName: "users" };
const waypointID = "waypoint0000001";
const assetID = "asset0000000001";
const pluginLinks = [{ pluginId: "immich", assetIds: ["remote-photo"] }];

function photoWaypoint() {
    const waypoint = new Waypoint(47, 8, { trail: "trail" });
    waypoint._assetPluginLinks = pluginLinks;
    return waypoint;
}

function savedWaypoint() {
    return { id: waypointID, trail: "trail", lat: 47, lon: 8 };
}

afterEach(() => vi.unstubAllGlobals());

describe("photo waypoint creation", () => {
    it.each([
        () => Response.json({ imported: [], omitted: [] }),
        () => Response.json({ message: "Provider unavailable" }, { status: 502 }),
    ])("removes a newly created empty waypoint after import failure", async (failedImport) => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(failedImport())
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(Response.json({ acknowledged: true }));

        await expect(waypoints_create(photoWaypoint(), request, user))
            .rejects.toMatchObject({ message: "asset_import_failed" });

        expect(request.mock.calls.map(([url, options]) => [url, options?.method ?? "GET"]))
            .toEqual([
                ["/api/v1/waypoint", "PUT"],
                ["/api/v1/plugins/assets/immich/import-to-target", "POST"],
                [`/api/v1/waypoint/${waypointID}?expand=waypoint_assets_via_waypoint.asset`, "GET"],
                [`/api/v1/waypoint/${waypointID}`, "DELETE"],
            ]);
    });

    it("keeps photos saved before a later plugin request fails", async () => {
        const waypoint = photoWaypoint();
        waypoint._photos = [new File(["photo"], "photo.jpg", { type: "image/jpeg" })];
        const asset = { id: assetID, collectionId: "assets", file: "photo.jpg", type: "photo" };
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(Response.json([asset]))
            .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 502 }))
            .mockResolvedValueOnce(Response.json({
                ...savedWaypoint(),
                expand: { waypoint_assets_via_waypoint: [{ asset: assetID, expand: { asset } }] },
            }));

        await expect(waypoints_create(waypoint, request, user)).rejects.toMatchObject({
            message: "asset_import_partial_failure",
            detail: {
                savedWaypoint: { id: waypointID, photos: [`/api/v1/files/assets/${assetID}/photo.jpg`] },
                remainingAttachments: { _photos: undefined, _assetLinks: undefined, _assetPluginLinks: pluginLinks },
            },
        });
        expect(request.mock.calls.some(([, options]) => options?.method === "DELETE")).toBe(false);
    });

    it("counts saved links even when the asset record was not expanded", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 502 }))
            .mockResolvedValueOnce(Response.json({
                ...savedWaypoint(), expand: { waypoint_assets_via_waypoint: [{ asset: assetID }] },
            }));

        await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({
            message: "asset_import_partial_failure", detail: { savedWaypoint: { id: waypointID } },
        });
        expect(request).toHaveBeenCalledTimes(3);
    });

    it("does not delete a waypoint whose saved attachments cannot be checked", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 502 }))
            .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 503 }));

        await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({
            message: "asset_import_cleanup_failed",
            detail: { savedWaypoint: { id: waypointID }, importError: { status: 502 } },
        });
        expect(request).toHaveBeenCalledTimes(3);
    });

    it("reports cleanup failure without losing the original import error", async () => {
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 502 }))
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 503 }));

        await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({
            message: "asset_import_cleanup_failed",
            detail: { savedWaypoint: { id: waypointID }, importError: { status: 502 }, cleanupError: { status: 503 } },
        });
    });

    it("keeps an intentionally photo-free waypoint", async () => {
        const request = vi.fn().mockResolvedValueOnce(Response.json(savedWaypoint()));
        await expect(waypoints_create(new Waypoint(47, 8), request, user))
            .resolves.toMatchObject({ id: waypointID, photos: [] });
        expect(request).toHaveBeenCalledOnce();
    });

    it("does not attempt cleanup when waypoint creation itself fails", async () => {
        const request = vi.fn().mockResolvedValueOnce(Response.json({ message: "Conflict" }, { status: 400 }));
        await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({ status: 400 });
        expect(request).toHaveBeenCalledOnce();
    });
});

describe("existing photo waypoint update", () => {
    it("keeps the waypoint and its old photo when replacement import fails", async () => {
        currentUser.set(user as never);
        const old = new Waypoint(47, 8, {
            id: waypointID, trail: "trail", photos: [`/api/v1/assets/${assetID}/file`],
        });
        const updated = photoWaypoint();
        updated.id = waypointID;
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json(savedWaypoint()))
            .mockResolvedValueOnce(Response.json({ imported: [], omitted: [{ assetId: "remote-photo", reason: "download_failed" }] }));
        vi.stubGlobal("fetch", request);

        await expect(waypoints_update(old, updated)).rejects.toMatchObject({ message: "asset_import_failed" });
        expect(request).toHaveBeenCalledTimes(2);
        expect(request.mock.calls.some(([, options]) => options?.method === "DELETE")).toBe(false);
    });
});
