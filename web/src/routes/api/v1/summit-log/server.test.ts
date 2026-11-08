import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { GET } from "./+server";

describe("summit-log photo projection", () => {
    it.each([undefined, {}, { summit_log_assets_via_summit_log: [{ asset: "photo" }] }])("keeps missing photo expansions absent (%j)", async (expand) => {
        const log = { id: "log", date: "2026-10-14 12:00:00", expand };
        const getList = vi.fn().mockResolvedValue({ items: [log] });
        const url = new URL("http://a.example/api/v1/summit-log");
        const response = await responseOverHttp(await GET({ url, locals: { pb: { collection: () => ({ getList }) } } } as unknown as RequestEvent));
        expect(response.status).toBe(200);
        const returned = (await response.json()).items[0];
        expect(returned.date).toBe("2026-10-14");
        expect(returned).not.toHaveProperty("photos");
    });

    it("builds photos when link assets are expanded", async () => {
        const log = { id: "log", date: "2026-10-14", expand: { summit_log_assets_via_summit_log: [{ asset: "photo", expand: { asset: { id: "photo", type: "photo", file: "photo.jpg", collectionId: "assets" } } }] } };
        const url = new URL("http://a.example/api/v1/summit-log?expand=summit_log_assets_via_summit_log.asset");
        const response = await responseOverHttp(await GET({ url, locals: { pb: { collection: () => ({ getList: vi.fn().mockResolvedValue({ items: [log] }) }) } } } as unknown as RequestEvent));
        expect((await response.json()).items[0].photos).toEqual(["/api/v1/files/assets/photo/photo.jpg"]);
    });
});

it.each([undefined, {}])("proves the requested empty photo relation instead of omitting photos (%j)", async (expand) => {
    const getFullList = vi.fn().mockResolvedValue([]);
    const getList = vi.fn().mockResolvedValue({ items: [{ id: "empty", date: "2026-10-14 12:00:00", expand }] });
    const url = new URL("http://a.example/api/v1/summit-log?expand=summit_log_assets_via_summit_log.asset&share=reader-share");
    const response = await responseOverHttp(await GET({ url, locals: { pb: { filter: vi.fn(() => "summit_log = 'empty'"), collection: () => ({ getList, getFullList }) } } } as unknown as RequestEvent));
    const log = (await response.json()).items[0];
    expect(log.photos).toEqual([]);
    expect(log.expand.summit_log_assets_via_summit_log).toEqual([]);
    expect(getFullList).toHaveBeenCalledWith(expect.objectContaining({ expand: "asset", query: { share: "reader-share" } }));
});

it("leaves an unsuccessful requested relation unproven", async () => {
    const url = new URL("http://a.example/api/v1/summit-log?expand=summit_log_assets_via_summit_log.asset");
    const response = await responseOverHttp(await GET({ url, locals: { pb: { filter: vi.fn(() => "summit_log = 'log'"), collection: () => ({
        getList: vi.fn().mockResolvedValue({ items: [{ id: "log", date: "2026-10-14", expand: {} }] }),
        getFullList: vi.fn().mockRejectedValue(new Error("expand unavailable")),
    }) } } } as unknown as RequestEvent));
    const log = (await response.json()).items[0];
    expect(log).not.toHaveProperty("photos");
    expect(log.expand).not.toHaveProperty("summit_log_assets_via_summit_log");
});
