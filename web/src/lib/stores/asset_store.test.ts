import { describe, expect, it, vi } from "vitest";

import { assets_attach_to_target, assets_import_plugin_links } from "./asset_store";
describe("assets_import_plugin_links", () => {
    it("keeps successful imports and reports omissions to the caller", async () => {
        const onOmitted = vi.fn();
        const request = vi.fn(async () => new Response(JSON.stringify({
            imported: [{ asset: { id: "asset-record" } }],
            omitted: [{ assetId: "missing", reason: "not_found" }],
        }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
        }));

        const assets = await assets_import_plugin_links([
            { pluginId: "immich", assetIds: ["asset-record", "missing"] },
        ], { trail: "trail" }, request, onOmitted);

        expect(assets).toEqual([{ id: "asset-record" }]);
        expect(request).toHaveBeenCalledOnce();
        expect(onOmitted).toHaveBeenCalledOnce();
        expect(onOmitted).toHaveBeenCalledWith([
            { pluginId: "immich", assetId: "missing", reason: "not_found" },
        ]);
    });

    it.each([
        { imported: [], omitted: [{ assetId: "missing", reason: "download_failed" }] },
        { imported: [], omitted: [] },
    ])("rejects a requested import that saved no photos: %j", async (response) => {
        const request = vi.fn(async () => Response.json(response));

        await expect(assets_import_plugin_links([
            { pluginId: "immich", assetIds: ["missing"] },
        ], { trail: "trail" }, request)).rejects.toMatchObject({ message: "asset_import_failed" });
    });

    it("does not report omissions when every selected photo was imported", async () => {
        const onOmitted = vi.fn();
        const request = vi.fn(async () => Response.json({
            imported: [{ asset: { id: "photo" } }], omitted: [],
        }));
        await expect(assets_import_plugin_links([
            { pluginId: "immich", assetIds: ["photo"] },
        ], { trail: "trail" }, request, onOmitted)).resolves.toEqual([{ id: "photo" }]);
        expect(onOmitted).not.toHaveBeenCalled();
    });

    it("reports completed uploads and plugin groups when a later group fails", async () => {
        const first = { pluginId: "first", assetIds: ["first-photo"] };
        const second = { pluginId: "second", assetIds: ["second-photo"] };
        const uploaded = { id: "uploaded", collectionId: "assets", file: "photo.jpg" };
        const request = vi.fn()
            .mockResolvedValueOnce(Response.json([uploaded]))
            .mockResolvedValueOnce(Response.json({ imported: [{ asset: { id: "remote" } }], omitted: [] }))
            .mockResolvedValueOnce(Response.json({ imported: [], omitted: [{ assetId: "second-photo", reason: "download failed" }] }));

        await expect(assets_attach_to_target({
            files: [new File(["photo"], "photo.jpg", { type: "image/jpeg" })],
            pluginLinks: [first, second],
            target: { trail: "trail", waypoint: "waypoint" },
            f: request,
        })).rejects.toMatchObject({
            message: "asset_import_failed",
            detail: {
                completedAttachments: { files: true, assetIds: true, pluginLinks: [first] },
                attachedPhotos: ["/api/v1/files/assets/uploaded/photo.jpg", "/api/v1/assets/remote/file"],
            },
        });
    });
});
