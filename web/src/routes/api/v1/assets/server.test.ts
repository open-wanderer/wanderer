import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { ClientResponseError } from "pocketbase";
import type { Asset } from "#lib/models/asset.ts";
import { PUT } from "./+server.ts";
import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";

function uploadEvent(fileMetadata: unknown[], metadata?: Record<string, unknown>) {
    const created: FormData[] = [];
    const collection = vi.fn((name: string) => ({
        getOne: vi.fn().mockResolvedValue({ id: "trail0000000001", author: "actor0000000001" }),
        getFullList: vi.fn().mockResolvedValue([]),
        create: vi.fn(async (data: FormData) => {
            if (name === "assets") {
                created.push(data);
            }
            return { id: `asset000000000${created.length}` };
        }),
    }));
    const cleanup = vi.fn().mockResolvedValue({ acknowledged: true, assetDeleted: true });
    const body = new FormData();
    body.append("trail", "trail0000000001");
    for (const [index] of fileMetadata.entries()) {
        body.append("files", new File(["no exif"], `photo${index}.jpg`, { type: "image/jpeg" }));
    }
    body.append("fileMetadata", JSON.stringify(fileMetadata));
    if (metadata) {
        body.append("metadata", JSON.stringify(metadata));
    }
    const event = {
        request: new Request("http://a.example/api/v1/assets", { method: "PUT", body }),
        locals: { user: { actor: "actor0000000001" }, pb: { collection, send: cleanup, filter: vi.fn(() => "") } },
    } as unknown as RequestEvent;
    return { event, created, collection, cleanup };
}

describe("photo upload", () => {
    it("reports successfully attached files when a later upload fails", async () => {
        const { event, collection } = uploadEvent([{}, {}]);
        const create = vi.fn()
            .mockResolvedValueOnce({ id: "asset0000000001" })
            .mockRejectedValueOnce(new ClientResponseError({ status: 400, response: { message: "Unsupported file" } }));
        collection.mockImplementation((name: string) => ({
            getOne: vi.fn().mockResolvedValue({ id: "trail0000000001", author: "actor0000000001" }),
            getFullList: vi.fn().mockResolvedValue([]),
            create: name === "assets" ? create : vi.fn().mockResolvedValue({}),
        }));
        const response = await responseOverHttp(await PUT(event));

        expect(response.status).toBe(400);
        expect(await response.json()).toMatchObject({ detail: { uploadedAssets: [{ id: "asset0000000001" }], completedFileCount: 1 } });
        expect(create).toHaveBeenCalledTimes(2);
    });

    it("removes an uploaded asset when its target link is rejected", async () => {
        const { event, collection, cleanup } = uploadEvent([{}]);
        collection.mockImplementation((name: string) => ({
            getOne: vi.fn().mockResolvedValue({ id: "trail0000000001", author: "actor0000000001" }),
            getFullList: vi.fn().mockResolvedValue([]),
            create: name === "assets"
                ? vi.fn().mockResolvedValue({ id: "asset0000000001" })
                : vi.fn().mockRejectedValue(new ClientResponseError({ status: 400, response: { message: "Link rejected" } })),
        }));
        expect((await PUT(event)).status).toBe(400);
        expect(cleanup).toHaveBeenCalledWith("/assets/asset0000000001?orphan=1", { method: "DELETE", fetch: event.fetch, requestKey: null });
    });


});

function linkEvent(source: Partial<Asset> = {}) {
    const actor = "actor0000000001";
    const original: Asset = {
        id: "source000000001", collectionId: "assetcollect001", collectionName: "assets",
        author: "other0000000001", type: "photo", storage_mode: "copy", file: "original.jpg",
        ...source,
    };
    const created: FormData[] = [];
    const assets = {
        getOne: vi.fn().mockResolvedValue(original),
        ownedCopy: vi.fn().mockRejectedValue(new ClientResponseError({ status: 404 })),
        create: vi.fn(async (data: FormData) => {
            created.push(data);
            return { ...original, id: "copy00000000001", author: actor, storage_mode: "copy", file: "copy.jpg" };
        }),
        delete: vi.fn().mockResolvedValue(true),
    };
    const links = { getFullList: vi.fn().mockResolvedValue([]), create: vi.fn().mockResolvedValue({}) };
    const collection = vi.fn((name: string) => name === "assets" ? assets
        : name.endsWith("_assets") ? links : { getOne: vi.fn().mockResolvedValue({ author: actor }) });
    const body = new FormData();
    body.append("trail", "trail0000000001");
    body.append("assetIds", original.id);
    const fetch = vi.fn().mockResolvedValue(new Response("original bytes", { headers: { "Content-Type": "image/jpeg" } }));
    const cleanup = vi.fn().mockResolvedValue({ acknowledged: true, assetDeleted: true });
    const send = vi.fn((path: string) => path.endsWith("/owned-copy") ? assets.ownedCopy() : cleanup(path));
    const event = {
        request: new Request("http://a.example/api/v1/assets", { method: "PUT", body }),
        locals: { user: { actor }, pb: { collection, send, filter: vi.fn(() => ""), authStore: { token: "reader-token" }, buildURL: (path: string) => `https://b.example/${path}` } },
        fetch,
    } as unknown as RequestEvent;
    return { event, original, assets, links, fetch, created, cleanup };
}

describe("readable photos attached from other authors", () => {
    it("uploads a canonical copy name instead of another stored-file suffix", async () => {
        const { event, created, fetch } = linkEvent({ file: "trail_a_473eqqcflx.jpg" });
        fetch.mockResolvedValue(new Response("webp bytes", { headers: { "Content-Type": "image/webp" } }));
        expect((await PUT(event)).status).toBe(200);
        expect((created[0].get("file") as File).name).toBe("trail_a.webp");
        expect(JSON.parse(String(created[0].get("metadata"))).original_filename).toBe("trail_a.webp");
    });
    it("reuses the own copy after a lost response without downloading or creating it again", async () => {
        const { event, original, assets, links, fetch } = linkEvent();
        const retry = event.request.clone();
        expect((await PUT(event)).status).toBe(200);
        const copied = { ...original, author: "actor0000000001", id: "copy00000000001", metadata: { copied_from: original.id } };
        assets.ownedCopy.mockResolvedValue(copied);
        links.getFullList.mockResolvedValue([{ id: "already-linked" }]);
        expect((await PUT({ ...event, request: retry })).status).toBe(200);
        expect(fetch).toHaveBeenCalledOnce();
        expect(assets.create).toHaveBeenCalledOnce();
        expect(links.create).toHaveBeenCalledOnce();
    });

    it("keeps a reused copy when linking another target fails", async () => {
        const { event, original, assets, links, fetch } = linkEvent();
        assets.ownedCopy.mockResolvedValue({ ...original, author: "actor0000000001", id: "copy00000000001" });
        links.create.mockRejectedValue(new ClientResponseError({ status: 400 }));
        expect((await PUT(event)).status).toBe(400);
        expect(assets.delete).not.toHaveBeenCalled();
        expect(fetch).not.toHaveBeenCalled();
    });
    it("continues to link the user's own assets without downloading them", async () => {
        const { event, original, created, links, fetch } = linkEvent({ author: "actor0000000001" });
        expect((await PUT(event)).status).toBe(200);
        expect(fetch).not.toHaveBeenCalled();
        expect(created).toEqual([]);
        expect(links.create).toHaveBeenCalledWith({ asset: original.id, trail: "trail0000000001" }, { requestKey: null });
    });

    it("does not download or create an asset when read permission has been revoked", async () => {
        const { event, assets, fetch } = linkEvent();
        assets.getOne.mockRejectedValue(new ClientResponseError({ status: 404, response: { message: "Not found" } }));
        expect((await PUT(event)).status).toBe(404);
        expect(fetch).not.toHaveBeenCalled();
        expect(assets.create).not.toHaveBeenCalled();
        expect(assets.ownedCopy).not.toHaveBeenCalled();
    });

    it("preserves storage failures after a successful download", async () => {
        const { event, assets } = linkEvent();
        assets.create.mockRejectedValue(new ClientResponseError({ status: 400, response: { message: "Invalid photo" } }));
        expect((await PUT(event)).status).toBe(400);
    });

    it("removes a newly copied file if creating the target link fails", async () => {
        const { event, assets, links, cleanup } = linkEvent();
        links.create.mockRejectedValue(new ClientResponseError({ status: 400, response: { message: "Failed to create record" } }));
        expect((await PUT(event)).status).toBe(400);
        expect(cleanup).toHaveBeenCalledWith("/assets/copy00000000001?orphan=1");
        expect(assets.delete).not.toHaveBeenCalled();
    });

    it("reports a saved copy when copying a later photo fails", async () => {
        const { event, original, assets } = linkEvent();
        assets.getOne.mockResolvedValueOnce(original)
            .mockRejectedValueOnce(new ClientResponseError({ status: 404, response: { message: "Not found" } }));
        const body = await event.request.formData();
        body.append("assetIds", "missing00000001");
        const response = await PUT({ ...event, request: new Request(event.request.url, { method: "PUT", body }) });

        expect(response.status).toBe(404);
        expect(await response.json()).toMatchObject({ detail: { attachedAssets: [{ id: "copy00000000001", sourceAssetId: original.id }] } });
        expect(assets.create).toHaveBeenCalledOnce();
        expect(assets.delete).not.toHaveBeenCalled();
    });
});


it("keeps a saved copy reused by another target while the first link fails", async () => {
    const { event, original, assets, links, cleanup } = linkEvent();
    const persisted = { ...original, id: "copy00000000001", author: "actor0000000001", storage_mode: "copy" as const, file: "copy.jpg" };
    let saved = false;
    assets.create.mockImplementation(async () => { saved = true; return persisted; });
    assets.ownedCopy.mockImplementation(async () => {
        if (!saved) throw new ClientResponseError({ status: 404 });
        return persisted;
    });
    const linkedTargets = new Set<string>();
    let started!: () => void;
    let failFirst!: (error: Error) => void;
    const firstLinkStarted = new Promise<void>((resolve) => { started = resolve; });
    const firstLink = new Promise<never>((_resolve, reject) => { failFirst = reject; });
    links.create.mockImplementation(async (link: { trail: string }) => {
        if (link.trail === "trail0000000001") { started(); return firstLink; }
        linkedTargets.add(link.trail);
        return {};
    });
    assets.delete.mockImplementation(async () => { linkedTargets.clear(); return true; });
    cleanup.mockImplementation(async () => {
        if (linkedTargets.size) return { acknowledged: true, assetDeleted: false };
        await assets.delete(persisted.id);
        return { acknowledged: true, assetDeleted: true };
    });
    const first = PUT(event);
    await firstLinkStarted;
    const body = new FormData();
    body.append("trail", "trail0000000002");
    body.append("assetIds", original.id);
    const second = await PUT({ ...event, request: new Request(event.request.url, { method: "PUT", body }) });
    expect(second.status).toBe(200);
    expect(linkedTargets.has("trail0000000002")).toBe(true);
    failFirst(new ClientResponseError({ status: 400, response: { message: "First link rejected" } }));
    expect((await first).status).toBe(400);
    expect(linkedTargets.has("trail0000000002")).toBe(true);
    expect(assets.delete).not.toHaveBeenCalled();
    expect(cleanup).toHaveBeenCalledWith("/assets/copy00000000001?orphan=1");
    expect(assets.create).toHaveBeenCalledOnce();
});
