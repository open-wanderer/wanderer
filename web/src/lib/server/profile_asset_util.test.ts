import type { FeedItem } from "#lib/models/feed.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import { enrichFeedItemAssetPhotos, enrichSummitLogAssetPhotos } from "./profile_asset_util.ts";
import { describe, expect, it } from "vitest";

describe("profile photo fallbacks", () => {
    it.each([undefined, "https://b.example"])("distinguishes omitted, confirmed empty and existing summit-log photos (%s)", (origin) => {
        const log = { id: "log000000000001", date: "2026-10-01", author: "actor0000000001" } as unknown as SummitLog;
        enrichSummitLogAssetPhotos([log], origin);
        expect(log).not.toHaveProperty("photos");
        const empty = {
            id: "log000000000002", date: "2026-10-01", author: "actor0000000001",
            expand: { summit_log_assets_via_summit_log: [] },
        } as unknown as SummitLog;
        enrichSummitLogAssetPhotos([empty], origin);
        expect(empty.photos).toEqual([]);
        const legacy = {
            id: "log000000000003", date: "2026-10-01", author: "actor0000000001", photos: ["photo.jpg"],
        } as unknown as SummitLog;
        enrichSummitLogAssetPhotos([legacy], origin);
        expect(legacy.photos).toEqual([origin ? "https://b.example/api/v1/files/summit_logs/log000000000003/photo.jpg" : "photo.jpg"]);
    });

    it.each(["trail", "summit_log"] as const)("keeps omitted %s feed photos unknown", (type) => {
        const item = { id: "record000000001", expand: {} };
        enrichFeedItemAssetPhotos([{ item: item.id, type, expand: { item } } as unknown as FeedItem], "https://b.example");
        expect(item).not.toHaveProperty("photos");
    });

});
