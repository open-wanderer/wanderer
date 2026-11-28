import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { PATCH } from "./+server.ts";

const trail = "trail0000000001";
const actor = "actor0000000001";
const photo = "photo0000000001";

function thumbnailEvent(owner = actor) {
    const links = [{ id: "selected", asset: photo, is_thumbnail: false }, { id: "previous", asset: "other0000000001", is_thumbnail: true }];
    const update = vi.fn().mockResolvedValue({});
    const shares = vi.fn().mockResolvedValue([]);
    const photoLinks = vi.fn((options) => Promise.resolve(options.filter.includes("asset") ? [links[0]] : links));
    const collection = vi.fn((name: string) => name === "trails" ? { getOne: vi.fn().mockResolvedValue({ author: owner }) }
        : name === "trail_share" ? { getFullList: shares } : { getFullList: photoLinks, update });
    const request = new Request(`https://a.example/api/v1/trail/${trail}/thumbnail`, { method: "PATCH", body: JSON.stringify({ asset: photo }) });
    const event = { params: { id: trail }, request, locals: { user: { actor }, pb: { collection, filter: (value: string) => value } } } as unknown as RequestEvent;
    return { event, update, shares, photoLinks };
}

describe("trail thumbnail selection", () => {
    it("marks the selected linked photo and clears the former thumbnail", async () => {
        const { event, update } = thumbnailEvent();
        expect((await PATCH(event)).status).toBe(200);
        expect(update.mock.calls).toEqual([
            ["selected", { is_thumbnail: true }, { requestKey: null }],
            ["previous", { is_thumbnail: false }, { requestKey: null }],
        ]);
    });

    it("requires trail ownership or a current edit share before writing links", async () => {
        const { event, update, shares } = thumbnailEvent("other0000000001");
        const retry = event.request.clone();
        expect((await PATCH(event)).status).toBe(403);
        expect(update).not.toHaveBeenCalled();
        shares.mockResolvedValue([{ id: "edit-share" }]);
        expect((await PATCH({ ...event, request: retry })).status).toBe(200);
        expect(update).toHaveBeenCalled();
    });

    it("rejects a photo outside the trail without changing its thumbnail", async () => {
        const { event, photoLinks, update } = thumbnailEvent();
        photoLinks.mockResolvedValue([]);
        expect((await PATCH(event)).status).toBe(400);
        expect(update).not.toHaveBeenCalled();
    });
});
