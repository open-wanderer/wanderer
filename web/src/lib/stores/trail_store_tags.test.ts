import { TagCreateSchema } from "$lib/models/api/tag_schema";
import { Trail } from "$lib/models/trail";
import type { AuthRecord } from "pocketbase";
import { afterEach, describe, expect, it, vi } from "vitest";
import { trails_create, trails_update } from "./trail_store";

afterEach(() => vi.unstubAllGlobals());

const user: AuthRecord = {
    id: "user00000000001", actor: "actor0000000001",
    collectionId: "users0000000001", collectionName: "users",
};

function taggedTrail(mode: string) {
    return new Trail("Trail with pasted tag", {
        id: mode === "update" ? "createdtrail001" : undefined,
        tags: [{ name: "Grü\tnezi\u007f" }],
    });
}

async function saveTrail(mode: string, trail: Trail, request: typeof fetch) {
    return mode === "create"
        ? await trails_create(trail, [], null, request, user)
        : await trails_update(new Trail("Old trail", { id: trail.id }), trail);
}

function expectLookup(input: RequestInfo | URL, options?: RequestInit) {
    const url = new URL(String(input), "https://wanderer.test");
    expect(url.pathname).toBe("/api/v1/tag/lookup");
    expect(url.search).toBe("");
    expect(options?.method).toBe("POST");
    expect(JSON.parse(String(options?.body))).toEqual({ name: "Grü nezi" });
}

describe("saving trails with pasted tag names", () => {
    it.each(["create", "update"])("continues to the trail %s after an empty lookup and tag creation", async mode => {
        const trail = taggedTrail(mode);
        const tagID = "createdtag00001";
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            const path = new URL(String(input), "https://wanderer.test").pathname;
            if (path === "/api/v1/tag/lookup") {
                expectLookup(input, options);
                return Response.json({ items: [] });
            }
            if (path === "/api/v1/tag") {
                expect(options?.method).toBe("PUT");
                const payload = TagCreateSchema.parse(JSON.parse(String(options?.body)));
                expect(payload.name).toBe("Grü nezi");
                return Response.json({ ...payload, id: tagID });
            }
            expect(path).toBe(mode === "create" ? "/api/v1/trail/form" : "/api/v1/trail/form/createdtrail001");
            expect(options?.method).toBe(mode === "create" ? "PUT" : "POST");
            expect(options?.body).toBeInstanceOf(FormData);
            expect((options!.body as FormData).getAll("tags")).toEqual([tagID]);
            return Response.json({ ...trail, id: "createdtrail001", expand: { tags: [{ id: tagID, name: "Grü nezi" }] } });
        });
        vi.stubGlobal("fetch", request);

        const result = await saveTrail(mode, trail, request);

        expect(request).toHaveBeenCalledTimes(3);
        expectLookup(...request.mock.calls[0]);
        expect(result.tags).toEqual([tagID]);
        expect(result.expand?.tags?.[0].name).toBe("Grü nezi");
    });

    it.each(["create", "update"])("uses the canonical ID in trail %s without a tag PUT", async mode => {
        const trail = taggedTrail(mode);
        const canonical = { id: "000000000000001", name: "Grü nezi" };
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            const path = new URL(String(input), "https://wanderer.test").pathname;
            if (path === "/api/v1/tag/lookup") {
                expectLookup(input, options);
                return Response.json({ items: [canonical] });
            }
            expect(path).toBe(mode === "create" ? "/api/v1/trail/form" : "/api/v1/trail/form/createdtrail001");
            expect(options?.method).toBe(mode === "create" ? "PUT" : "POST");
            expect((options!.body as FormData).getAll("tags")).toEqual([canonical.id]);
            return Response.json({ ...trail, id: "createdtrail001", expand: { tags: [canonical] } });
        });
        vi.stubGlobal("fetch", request);

        const result = await saveTrail(mode, trail, request);

        expect(request).toHaveBeenCalledTimes(2);
        expect(result.tags).toEqual([canonical.id]);
        expect(result.expand?.tags).toEqual([canonical]);
    });

    it.each(["create", "update"])("aborts trail %s when lookup fails instead of creating a tag or saving the trail", async mode => {
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            expectLookup(input, options);
            return Response.json({ message: "Lookup unavailable", detail: "Tag lookup failed" }, { status: 503 });
        });
        vi.stubGlobal("fetch", request);

        await expect(saveTrail(mode, taggedTrail(mode), request)).rejects.toMatchObject({ status: 503, message: "Lookup unavailable" });
        expect(request).toHaveBeenCalledOnce();
    });
});
