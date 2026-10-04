import { TagCreateSchema } from "$lib/models/api/tag_schema";
import type { Tag } from "$lib/models/tag";
import { afterEach, describe, expect, it, vi } from "vitest";
import { tags_create } from "./tag_store";

afterEach(() => vi.unstubAllGlobals());

describe("tag creation", () => {
    it.each([
        ["pasted tab", "Grü\tnezi", "Grünezi"],
        ["all C0 and DEL", "a" + String.fromCharCode(...Array.from({ length: 32 }, (_, i) => i), 127) + "b", "ab"],
        ["controls only", "\t\u0000\u007f", ""],
        ["literal Unicode and whitespace", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; e\u0301 \u0085\u00a0\u2028\u200d  ", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; e\u0301 \u0085\u00a0\u2028\u200d  "],
        ["length checked after cleaning", "🌍".repeat(5000) + "\t", "🌍".repeat(5000)],
    ])("sends an API-valid copy for %s", async (_label, name, expectedName) => {
        const tag: Tag = Object.freeze({ id: "tag000000000001", name });
        const request = vi.fn(async (_input: RequestInfo | URL, options?: RequestInit) => {
            const payload = TagCreateSchema.parse(JSON.parse(String(options?.body)));
            return Response.json(payload);
        });
        vi.stubGlobal("fetch", request);

        await expect(tags_create(tag)).resolves.toEqual({ ...tag, name: expectedName });
        expect(request).toHaveBeenCalledOnce();
        const [url, options] = request.mock.calls[0];
        expect(url).toBe("/api/v1/tag");
        expect(options?.method).toBe("PUT");
        expect(JSON.parse(String(options?.body))).toEqual({ ...tag, name: expectedName });
        expect(tag.name).toBe(name);
    });

    it("keeps the API error and does not truncate an overlong name", async () => {
        const name = "🌍".repeat(5001) + "\t";
        const request = vi.fn(async (_input: RequestInfo | URL, options?: RequestInit) => {
            const payload = JSON.parse(String(options?.body));
            expect(payload.name).toBe("🌍".repeat(5001));
            expect(TagCreateSchema.safeParse(payload).success).toBe(false);
            return Response.json({ message: "Invalid tag", detail: "too-long" }, { status: 400 });
        });
        vi.stubGlobal("fetch", request);

        await expect(tags_create({ name })).rejects.toMatchObject({ status: 400, message: "Invalid tag", detail: "too-long" });
    });
});
