import { describe, expect, it } from "vitest";
import { TagCreateSchema, TagUpdateSchema } from "./tag_schema";

describe("tag names", () => {
    it.each(["Grüezi", "日本語", "مرحبا", "e\u0301", "👩‍👩‍👧‍👦", "[route].* (test)", "O'Brien", "<img src=x>", "&lt;img&gt;", ""])(
        "preserves legitimate plain text %s", name => {
            expect(TagCreateSchema.parse({ name }).name).toBe(name);
            expect(TagUpdateSchema.parse({ name }).name).toBe(name);
        },
    );
    it.each(["null\u0000byte", "line\nfeed", "tab\tname", "del\u007f", "trailing\n", "trailing\r", "\n"])("rejects control characters in %s", name => {
        expect(TagCreateSchema.safeParse({ name }).success).toBe(false);
        expect(TagUpdateSchema.safeParse({ name }).success).toBe(false);
    });
    it("counts Unicode code points consistently with PocketBase", () => {
        expect(TagCreateSchema.safeParse({ name: "🌍".repeat(5000) }).success).toBe(true);
        expect(TagCreateSchema.safeParse({ name: "🌍".repeat(5001) }).success).toBe(false);
        expect(TagUpdateSchema.safeParse({ name: "a".repeat(5001) }).success).toBe(false);
        expect(TagUpdateSchema.parse({})).toEqual({});
    });
});
