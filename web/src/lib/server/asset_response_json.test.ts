import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import type { Trail } from "#lib/models/trail.ts";
import { expect, it, vi } from "vitest";
import { assetResponseJson } from "./asset_response_json";
import { enrichTrailResponse } from "./trail_response_util";
import { GET as getList } from "../../routes/api/v1/list/[id]/+server";

it("serializes each linked asset once while preserving photos, thumbnails and thumbnails", async () => {
    const asset = { id: "asset0000000001", collectionId: "assets", type: "photo", file: "photo.jpg", metadata: { generated: { kind: "route-preview" } } };
    const trail = { id: "trail", name: "Hike", expand: { trail_assets_via_trail: [{ asset: asset.id, is_thumbnail: true, expand: { asset } }] } } as unknown as Trail;
    enrichTrailResponse(trail);
    expect(trail.expand?.assets_via_trail).toHaveLength(1);
    const response = await responseOverHttp(assetResponseJson(trail, { status: 201, headers: { "X-Test": "kept" } }));
    const bytes = await response.text();
    const record = JSON.parse(bytes);
    expect(response.status).toBe(201); expect(response.headers.get("X-Test")).toBe("kept");
    expect(record.expand.assets_via_trail).toBeUndefined();
    expect(record.photos).toEqual(["/api/v1/files/assets/asset0000000001/photo.jpg"]);
    expect(record.thumbnail).toBe(0);
    expect(bytes.match(/"file":"photo.jpg"/g)).toHaveLength(1);
});

it("requests only bounded photo previews for list cards and retains the share context", async () => {
    const send = vi.fn(async (path: string, options?: { query?: unknown }) => path.startsWith("/remote/list/")
        ? { id: "list", expand: { trails: [{ id: "trail", name: "Hike" }] } }
        : { previews: { trail: { photos: ["one", "two", "three"], thumbnail: 0 } } });
    const request = new Request("http://a.example/api/v1/list/list?expand=trails,trails.trail_assets_via_trail.asset&photoPreview=true&share=secret");
    const event = { request, url: new URL(request.url), params: { id: "list" }, locals: { pb: { send } }, fetch: vi.fn() } as unknown as RequestEvent;
    const response = await responseOverHttp(await getList(event));
    expect(response.status).toBe(200);
    expect((await response.json()).expand.trails[0]).toMatchObject({ photos: ["one", "two", "three"], thumbnail: 0 });
    expect(send.mock.calls[0][0]).not.toContain("_assets_via_");
    expect(send.mock.calls[1]).toEqual(["/assets/trail-previews", { method: "GET", query: { ids: "trail", share: "secret" }, signal: request.signal }]);
    expect(send).toHaveBeenCalledTimes(2);
});


it.each(["read", "update"])('serializes single summit-log photos once after %s', async (operation) => {
    const { GET, POST } = await import("../../routes/api/v1/summit-log/[id]/+server");
    const asset = { id: "asset0000000001", collectionId: "assets", type: "photo", file: "summit.jpg" };
    const record = { id: "summitlog000001", date: "2026-10-01", expand: { summit_log_assets_via_summit_log: [{ asset: asset.id, expand: { asset } }] } };
    const url = new URL("https://a.example/api/v1/summit-log/summitlog000001?expand=summit_log_assets_via_summit_log.asset");
    const pb = { collection: () => ({ getOne: vi.fn().mockResolvedValue(record), update: vi.fn().mockResolvedValue(record) }) };
    const event = { url, params: { id: record.id }, locals: { pb }, request: new Request(url, operation === "update" ? { method: "POST", body: JSON.stringify({ text: "Updated" }) } : {}) } as unknown as RequestEvent;
    const response = await responseOverHttp(await (operation === "read" ? GET : POST)(event));
    expect(response.status).toBe(200);
    const body = await response.text();
    expect(body.match(/"file":"summit.jpg"/g)).toHaveLength(1);
    const log = JSON.parse(body);
    expect(log.photos).toEqual(["/api/v1/files/assets/asset0000000001/summit.jpg"]);
    expect(log.expand.assets_via_summit_log).toBeUndefined();
    expect(log.expand.summit_log_assets_via_summit_log).toHaveLength(1);
});

it("retains link access on full photo URLs from a shared list", async () => {
    const share = "a".repeat(32);
    const asset = { id: "asset0000000001", collectionId: "assets", type: "photo", file: "summit.jpg" };
    const trail = { id: "trail0000000001", name: "Shared hike", expand: { trail_assets_via_trail: [{ asset: asset.id, expand: { asset } }] } };
    const url = new URL("https://a.example/api/v1/list/list00000000001?expand=trails&share=" + share);
    const event = { url, params: { id: "list00000000001" }, locals: { pb: { send: vi.fn().mockResolvedValue({ id: "list00000000001", expand: { trails: [trail] } }) } }, request: new Request(url) } as unknown as RequestEvent;
    const response = await responseOverHttp(await getList(event));
    expect(response.status).toBe(200);
    expect((await response.json()).expand.trails[0].photos).toEqual(["/api/v1/files/assets/asset0000000001/summit.jpg?share=" + share]);
});
