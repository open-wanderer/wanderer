import { TagCreateSchema } from "$lib/models/api/tag_schema";
import type { Tag } from "$lib/models/tag";
import { afterEach, describe, expect, it, vi } from "vitest";
import { tags_create } from "./tag_store";

afterEach(() => vi.unstubAllGlobals());

function expectLookup(input: RequestInfo | URL, options: RequestInit | undefined, name: string) {
    const url = new URL(String(input), "https://wanderer.test");
    expect(url.pathname).toBe("/api/v1/tag/lookup");
    expect(url.search).toBe("");
    expect(options?.method).toBe("POST");
    expect(new Headers(options?.headers).get("Content-Type")).toBe("application/json");
    expect(JSON.parse(String(options?.body))).toEqual({ name });
}

describe("tag creation", () => {
    it.each([
        ["pasted tab", "Grü\tnezi", "Grü nezi"],
        ["all C0 and DEL", "a" + String.fromCharCode(...Array.from({ length: 32 }, (_, i) => i), 127) + "b", "a     b"],
        ["controls only", "\t\u0000\u007f", " "],
        ["empty API name", "\u0000\u007f", ""],
        ["CRLF keeps two boundaries", "word\r\nword", "word  word"],
        ["literal Unicode and whitespace", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; e\u0301 \u0085\u00a0\u2028\u200d  ", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; e\u0301 \u0085\u00a0\u2028\u200d  "],
        ["length checked after cleaning", "🌍".repeat(5000) + "\u0000", "🌍".repeat(5000)],
        ["whitespace at the length boundary", "🌍".repeat(4999) + "\t", "🌍".repeat(4999) + " "],
    ])("creates an API-valid copy after an empty lookup for %s", async (_label, name, expectedName) => {
        const tag: Tag = Object.freeze({ id: "tag000000000001", name });
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            if (new URL(String(input), "https://wanderer.test").pathname === "/api/v1/tag/lookup") {
                expectLookup(input, options, expectedName);
                expect(TagCreateSchema.parse(JSON.parse(String(options?.body))).name).toBe(expectedName);
                return Response.json({ items: [] });
            }
            expect(String(input)).toBe("/api/v1/tag");
            expect(options?.method).toBe("PUT");
            const payload = TagCreateSchema.parse(JSON.parse(String(options?.body)));
            return Response.json(payload);
        });
        vi.stubGlobal("fetch", request);

        await expect(tags_create(tag)).resolves.toEqual({ ...tag, name: expectedName });
        expect(request).toHaveBeenCalledTimes(2);
        const [lookupInput, lookupOptions] = request.mock.calls[0];
        expectLookup(lookupInput, lookupOptions, expectedName);
        expect(JSON.parse(String(request.mock.calls[1][1]?.body))).toEqual({ ...tag, name: expectedName });
        expect(tag.name).toBe(name);
    });

    it.each([
        ["normalized words", "Grü\tnezi\u007f", "Grü nezi"],
        ["apostrophe", "O'Brien", "O'Brien"],
        ["double quotes", 'He said "Hike"', 'He said "Hike"'],
        ["mixed quotes", '"O\'Brien"', '"O\'Brien"'],
        ["embedded backslashes", "C:\\trails\\name", "C:\\trails\\name"],
        ["trailing backslash", "trail\\", "trail\\"],
        ["literal wildcard characters", "100%_done", "100%_done"],
        ["Unicode and spacing", "  e\u0301 日本語 🌍  ", "  e\u0301 日本語 🌍  "],
        ["empty API name", "\u0000\u007f", ""],
        ["space-only API name", "  ", "  "],
    ])("reuses an existing tag for %s without creating a duplicate", async (_label, name, expectedName) => {
        const tag: Tag = Object.freeze({ name });
        // The lookup endpoint sorts by ID and returns the smallest existing ID.
        const canonical = { id: "000000000000001", name: expectedName, created: "2026-10-04T00:00:00Z" };
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            expectLookup(input, options, expectedName);
            return Response.json({ items: [canonical] });
        });
        vi.stubGlobal("fetch", request);

        await expect(tags_create(tag)).resolves.toEqual(canonical);
        expect(request).toHaveBeenCalledOnce();
        expect(tag.name).toBe(name);
    });

    it.each([403, 503])("propagates lookup HTTP %s without attempting creation", async status => {
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            expectLookup(input, options, "Word Word");
            return Response.json({ message: "Lookup unavailable", detail: "Tag lookup failed" }, { status });
        });
        vi.stubGlobal("fetch", request);

        await expect(tags_create({ name: "Word\tWord" })).rejects.toMatchObject({
            status, message: "Lookup unavailable", detail: "Tag lookup failed",
        });
        expect(request).toHaveBeenCalledOnce();
    });

    it("propagates a failed lookup request without attempting creation", async () => {
        const failure = new Error("Network unavailable");
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            expectLookup(input, options, "Word Word");
            throw failure;
        });
        vi.stubGlobal("fetch", request);

        await expect(tags_create({ name: "Word\tWord" })).rejects.toBe(failure);
        expect(request).toHaveBeenCalledOnce();
    });

    it("propagates overlong lookup rejection without truncation or creation", async () => {
        const name = "🌍".repeat(5000) + "\t";
        const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            expectLookup(input, options, "🌍".repeat(5000) + " ");
            expect(TagCreateSchema.safeParse(JSON.parse(String(options?.body))).success).toBe(false);
            return Response.json({ message: "Invalid tag", detail: "too-long" }, { status: 400 });
        });
        vi.stubGlobal("fetch", request);

        await expect(tags_create({ name })).rejects.toMatchObject({ status: 400, message: "Invalid tag", detail: "too-long" });
        expect(request).toHaveBeenCalledOnce();
    });
});
