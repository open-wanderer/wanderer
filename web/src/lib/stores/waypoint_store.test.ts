import { afterEach, expect, it, vi } from "vitest";
import type { AuthRecord } from "pocketbase";
import { Waypoint } from "#lib/models/waypoint.ts";
import { currentUser } from "./user_store";
import { waypoints_create, waypoints_update } from "./waypoint_store";

const user: AuthRecord = { id: "user", actor: "actor", collectionId: "users", collectionName: "users" };
const waypointID = "waypoint0000001";
const assetID = "asset0000000001";
const linkedID = "linked000000001";
const photo = `/api/v1/files/assets/${assetID}/saved.jpg`;
function photoWaypoint() {
    const waypoint = new Waypoint(47, 8, { trail: "trail" });
    waypoint._assetLinks = [linkedID];
    waypoint.photos = [`/api/v1/assets/${linkedID}/file`];
    return waypoint;
}
function savedWaypoint() { return { id: waypointID, trail: "trail", lat: 47, lon: 8 }; }
const failed = () => Response.json({ message: "Unavailable" }, { status: 502 });
afterEach(() => { vi.unstubAllGlobals(); currentUser.set(null); });

it("counts saved links even when the asset record was not expanded", async () => {
    const request = vi.fn()
        .mockResolvedValueOnce(Response.json(savedWaypoint()))
        .mockResolvedValueOnce(failed())
        .mockResolvedValueOnce(Response.json({ ...savedWaypoint(), expand: { waypoint_assets_via_waypoint: [{ asset: assetID }] } }));
    await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({
        message: "asset_import_partial_failure", detail: { savedWaypoint: { id: waypointID } },
    });
    expect(request).toHaveBeenCalledTimes(3);
    expect(request.mock.calls.some(([, options]) => options?.method === "DELETE")).toBe(false);
});

it("does not delete a waypoint whose saved attachments cannot be checked", async () => {
    const request = vi.fn()
        .mockResolvedValueOnce(Response.json(savedWaypoint()))
        .mockResolvedValueOnce(failed())
        .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 503 }));
    await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({
        message: "asset_import_cleanup_failed", detail: { savedWaypoint: { id: waypointID }, importError: { status: 502 } },
    });
    expect(request).toHaveBeenCalledTimes(3);
    expect(request.mock.calls.some(([, options]) => options?.method === "DELETE")).toBe(false);
});

it("reports cleanup failure without losing the original import error", async () => {
    const request = vi.fn()
        .mockResolvedValueOnce(Response.json(savedWaypoint()))
        .mockResolvedValueOnce(failed())
        .mockResolvedValueOnce(Response.json(savedWaypoint()))
        .mockResolvedValueOnce(Response.json({ message: "Unavailable" }, { status: 503 }));
    await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({
        message: "asset_import_cleanup_failed",
        detail: { savedWaypoint: { id: waypointID }, importError: { status: 502 }, cleanupError: { status: 503 } },
    });
});

it("keeps an intentionally photo-free waypoint", async () => {
    const request = vi.fn().mockResolvedValueOnce(Response.json(savedWaypoint()));
    await expect(waypoints_create(new Waypoint(47, 8), request, user)).resolves.toMatchObject({ id: waypointID, photos: [] });
    expect(request).toHaveBeenCalledOnce();
});

it("does not attempt cleanup when waypoint creation itself fails", async () => {
    const request = vi.fn().mockResolvedValueOnce(Response.json({ message: "Conflict" }, { status: 400 }));
    await expect(waypoints_create(photoWaypoint(), request, user)).rejects.toMatchObject({ status: 400 });
    expect(request).toHaveBeenCalledOnce();
});

it("keeps the waypoint and its old photo when replacement linking fails", async () => {
    currentUser.set(user as never);
    const old = new Waypoint(47, 8, { id: waypointID, trail: "trail", photos: [photo] });
    const updated = photoWaypoint(); updated.id = waypointID;
    const request = vi.fn().mockResolvedValueOnce(Response.json(savedWaypoint())).mockResolvedValueOnce(failed());
    vi.stubGlobal("fetch", request);
    await expect(waypoints_update(old, updated)).rejects.toMatchObject({ message: "asset_import_failed", detail: { savedWaypoint: { photos: [photo] } } });
    expect(request).toHaveBeenCalledTimes(2);
    expect(request.mock.calls.some(([, options]) => options?.method === "DELETE")).toBe(false);
});

it("retries a transient link failure without uploading the saved photo again", async () => {
    const waypoint = photoWaypoint();
    waypoint._photos = [new File(["photo"], "saved.jpg", { type: "image/jpeg" })];
    const asset = { id: assetID, collectionId: "assets", type: "photo", file: "saved.jpg" };
    const request = vi.fn()
        .mockResolvedValueOnce(Response.json(savedWaypoint()))
        .mockResolvedValueOnce(Response.json([asset]))
        .mockResolvedValueOnce(failed())
        .mockResolvedValueOnce(Response.json({ ...savedWaypoint(), expand: { waypoint_assets_via_waypoint: [{ asset: assetID, expand: { asset } }] } }))
        .mockResolvedValueOnce(Response.json({ ...savedWaypoint(), photos: [photo] }))
        .mockResolvedValueOnce(Response.json([{ ...asset, id: linkedID, file: undefined }]));
    const failure = await waypoints_create(waypoint, request, user).catch(error => error);
    expect(failure.detail.savedWaypoint.id).toBe(waypointID);
    expect(failure.detail.remainingAttachments).toEqual({ _photos: undefined, _assetLinks: [linkedID] });
    currentUser.set(user as never); vi.stubGlobal("fetch", request);
    const retry = await waypoints_update(failure.detail.savedWaypoint, { ...failure.detail.savedWaypoint, ...failure.detail.remainingAttachments, photos: [...failure.detail.savedWaypoint.photos, ...waypoint.photos!] });
    expect(retry.photos).toHaveLength(2);
    expect(request.mock.calls.filter(([url]) => String(url) === "/api/v1/waypoint")).toHaveLength(1);
    expect(request.mock.calls.filter(([, options]) => (options?.body as FormData)?.has?.("files"))).toHaveLength(1);
});
