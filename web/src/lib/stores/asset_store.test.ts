import { describe, expect, it, vi } from "vitest";

import { assets_import_plugin_links } from "./asset_store";

describe("assets_import_plugin_links", () => {
    it("reads imported assets from the import envelope", async () => {
        const request = vi.fn(async () => new Response(JSON.stringify({
            imported: [{ asset: { id: "asset-record" } }],
            omitted: [{ assetId: "missing", reason: "not_found" }],
        }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
        }));

        const assets = await assets_import_plugin_links([
            { pluginId: "immich", assetIds: ["asset-record", "missing"] },
        ], { trail: "trail" }, request);

        expect(assets).toEqual([{ id: "asset-record" }]);
        expect(request).toHaveBeenCalledOnce();
    });
});
