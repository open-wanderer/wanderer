import { afterEach, expect, it, vi } from "vitest";
import { assets_attach_to_target, assets_delete_removed } from "./asset_store";

const asset = { id: "photo0000000001", collectionId: "assets", collectionName: "assets", type: "photo", author: "actor", file: "photo.jpg" };
const photo = `/api/v1/files/assets/${asset.id}/photo.jpg`;
afterEach(() => vi.unstubAllGlobals());

it("keeps omitted photos and removes an explicitly cleared selection", async () => {
    const request = vi.fn(async () => Response.json({ acknowledged: true }));
    vi.stubGlobal("fetch", request);
    await assets_delete_removed([photo], undefined, { trail: "trail" });
    expect(request).not.toHaveBeenCalled();
    await assets_delete_removed([photo], [], { trail: "trail" });
    expect(request).toHaveBeenCalledWith(`/api/v1/assets/${asset.id}?trail=trail`, { method: "DELETE" });
});

it("retains uploaded photos and completion information after linking fails", async () => {
    const request = vi.fn(async (_url: RequestInfo | URL, init?: RequestInit) => (init!.body as FormData).has("files")
        ? Response.json([asset])
        : Response.json({ message: "link unavailable" }, { status: 503 }));
    const error = await assets_attach_to_target({ files: [new File(["photo"], "photo.jpg")], assetIds: ["source000000001"], target: { trail: "trail" }, f: request }).catch((error) => error);
    expect(error.detail).toMatchObject({ attachedPhotos: [photo], completedAttachments: { files: true, assetIds: false } });
});

it("replaces a copied source identity before exposing linked photos", async () => {
    const linked = { ...asset, sourceAssetId: "source000000001" };
    const onLinked = vi.fn();
    const request = vi.fn(async () => Response.json([linked]));
    expect(await assets_attach_to_target({ assetIds: [linked.sourceAssetId], target: { trail: "trail" }, existingPhotos: [`/api/v1/assets/${linked.sourceAssetId}/file`], onLinked, f: request })).toEqual([photo]);
    expect(onLinked).toHaveBeenCalledWith([linked]);
});

it.each(["trail", "waypoint", "summit-log"] as const)("preserves existing photos when a %s edit omitted its photo list", async (kind) => {
    const { currentUser } = await import("./user_store");
    currentUser.set({ id: "user", actor: "actor", collectionId: "users", collectionName: "users", password: "unused" });
    const old = { id: "target000000001", name: "Trail", lat: 47, lon: 8, date: "2026-11-09", author: "actor", public: false, completed: true, like_count: 0, tags: [], photos: [photo], expand: { waypoints_via_trail: [], summit_logs_via_trail: [] } };
    const request = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
        if (String(url).includes("/federation/")) return Response.json({ status: "not_federated" });
        expect(init?.method).not.toBe("DELETE");
        return Response.json({ ...old, photos: undefined });
    });
    vi.stubGlobal("fetch", request);
    const pending = { ...old, name: "Edited", photos: undefined };
    try {
        if (kind === "trail") await (await import("./trail_store")).trails_update(old, pending);
        if (kind === "waypoint") await (await import("./waypoint_store")).waypoints_update({ ...old, expand: undefined }, { ...pending, expand: undefined });
        if (kind === "summit-log") await (await import("./summit_log_store")).summit_logs_update({ ...old, expand: undefined }, { ...pending, expand: undefined });
        expect(request.mock.calls.filter(([, init]) => init?.method === "DELETE")).toEqual([]);
    } finally { currentUser.set(null); }
});
