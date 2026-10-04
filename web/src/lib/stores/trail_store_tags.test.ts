import { TagCreateSchema } from "$lib/models/api/tag_schema";
import { Trail } from "$lib/models/trail";
import type { AuthRecord } from "pocketbase";
import { afterEach, describe, expect, it, vi } from "vitest";
import { trails_create, trails_update } from "./trail_store";

afterEach(() => vi.unstubAllGlobals());

describe("saving trails with pasted tag names", () => {
    it.each(["create", "update"])("continues to the trail %s after creating a cleaned tag", async mode => {
        const trail = new Trail("Trail with pasted tag", {
            id: mode === "update" ? "createdtrail001" : undefined,
            tags: [{ name: "Grü\tnezi\u007f" }],
        });
        const tagID = "createdtag00001";
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            const path = new URL(String(input), "https://wanderer.test").pathname;
            if (path === "/api/v1/tag") {
                const payload = TagCreateSchema.safeParse(JSON.parse(String(options?.body)));
                if (!payload.success) {
                    return Response.json({ message: "Invalid tag", detail: payload.error.message }, { status: 400 });
                }
                return Response.json({ ...payload.data, id: tagID });
            }
            expect(path).toBe(mode === "create" ? "/api/v1/trail/form" : "/api/v1/trail/form/createdtrail001");
            expect(options?.method).toBe(mode === "create" ? "PUT" : "POST");
            expect(options?.body).toBeInstanceOf(FormData);
            expect((options!.body as FormData).getAll("tags")).toEqual([tagID]);
            return Response.json({ ...trail, id: "createdtrail001", expand: { tags: [{ id: tagID, name: "Grünezi" }] } });
        });
        vi.stubGlobal("fetch", request);
        const user: AuthRecord = {
            id: "user00000000001", actor: "actor0000000001",
            collectionId: "users0000000001", collectionName: "users",
        };

        const result = mode === "create"
            ? await trails_create(trail, [], null, request, user)
            : await trails_update(new Trail("Old trail", { id: trail.id }), trail);

        expect(request).toHaveBeenCalledTimes(2);
        expect(result.tags).toEqual([tagID]);
        expect(result.expand?.tags?.[0].name).toBe("Grünezi");
    });
});
