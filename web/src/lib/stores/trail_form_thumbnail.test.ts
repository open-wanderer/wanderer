import { afterEach, expect, it, vi } from "vitest";
import { Trail } from "#lib/models/trail.ts";
import { trails_create, trails_update } from "./trail_store";

afterEach(() => vi.unstubAllGlobals());
it.each(["create", "update"])("selects a %s cover through the asset endpoint without serializing the legacy thumbnail index", async mode => {
    const original = new Trail("Outing", { id: "trail000000001", public: false, photos: [], thumbnail: 0 });
    original.author = "actor0000000001";
    const pending = { ...original, thumbnail: 1 };
    const user = { id: "user", actor: original.author, collectionId: "users", collectionName: "users" };
    const bodies: FormData[] = [];
    const thumbnailRequests: string[] = [];
    const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const path = new URL(String(input), "https://a.example").pathname;
        if (path.includes("/federation/")) return Response.json({ status: "not_federated" });
        if (path.endsWith("/thumbnail")) { thumbnailRequests.push(path); return Response.json({ acknowledged: true }); }
        if (path.startsWith("/api/v1/trail/form")) bodies.push(options!.body as FormData);
        return Response.json(original);
    });
    vi.stubGlobal("fetch", request);
    if (mode === "create") await trails_create(pending, [], null, request, user);
    else await trails_update(original, pending);
    expect(bodies).toHaveLength(1);
    expect(bodies[0].has("thumbnail")).toBe(false);
    expect(thumbnailRequests).toEqual([`/api/v1/trail/${original.id}/thumbnail`]);
});
