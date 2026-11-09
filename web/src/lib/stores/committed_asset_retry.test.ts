import { afterEach, expect, it, vi } from "vitest";
import type { Trail } from "#lib/models/trail.ts";
import { trails_create, trails_update } from "./trail_store";
import { currentUser } from "./user_store";

const user = { id: "user00000000001", actor: "actor0000000001", collectionId: "users", collectionName: "users" };
afterEach(() => { vi.unstubAllGlobals(); currentUser.set(null); });
it("retries an already committed trail as update while preserving its pending upload", async () => {
    currentUser.set(user as never);
    const id = "target000000001";
    const stored = { id, name: "Saved", author: user.actor, trail: "trail0000000001", date: "2026-10-01", public: false, completed: true, like_count: 0, tags: [], photos: [] as string[], lat: 0, lon: 0, expand: { waypoints_via_trail: [] } };
    let metadataCreates = 0;
    let uploads = 0;
    const request = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
        const path = String(url);
        if (path.includes("/federation/")) return Response.json({ status: "ok" });
        if (init?.method === "PUT" && path !== "/api/v1/assets") {
            metadataCreates++;
            return Response.json({ message: "Refresh unavailable", detail: { recordCommitted: true, savedRecord: { ...stored, photos: undefined, expand: undefined } } }, { status: 503 });
        }
        if (path === "/api/v1/assets") {
            uploads++;
            stored.photos = ["/api/v1/assets/upload000000001/file"];
            return Response.json([{ id: "upload000000001", type: "photo", storage_mode: "link_private" }]);
        }
        return Response.json(stored);
    });
    vi.stubGlobal("fetch", request);
    const files = [new File(["photo"], "pending.jpg", { type: "image/jpeg" })];
    const pending: Trail = { ...stored, id: undefined };
    const error = await trails_create(pending, files, null, request, user).catch((error) => error);
    expect(error.detail.savedTrail.id).toBe(id);
    expect(pending.id).toBe(id);
    await trails_update(error.detail.savedTrail, pending, files);
    expect(metadataCreates).toBe(1);
    expect(uploads).toBe(1);
});
