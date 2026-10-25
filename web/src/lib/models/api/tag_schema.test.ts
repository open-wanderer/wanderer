import { describe, expect, it } from "vitest";
import { TagCreateSchema, TagUpdateSchema } from "./tag_schema";

describe("tag names", () => {
    it.each(["Grüezi", "日本語", "مرحبا", "e\u0301", "👩‍👩‍👧‍👦", "[route].* (test)", "O'Brien", "100%", "tag\\", "<img src=x>", "&lt;img&gt;", "  name  ", "\u0080\u009f\u00a0\u200d\u2028", ""])(
        "preserves legitimate plain text %s", name => {
            expect(TagCreateSchema.parse({ name }).name).toBe(name);
            expect(TagUpdateSchema.parse({ name }).name).toBe(name);
        },
    );
    it.each(["null\u0000byte", "line\nfeed", "tab\tname", "unit\u001fseparator", "del\u007f", "trailing\n", "trailing\r", "\n"])("rejects control characters in %s", name => {
        expect(TagCreateSchema.safeParse({ name }).success).toBe(false);
        expect(TagUpdateSchema.safeParse({ name }).success).toBe(false);
    });
    it.each([
        ["ASCII", "a".repeat(5000)],
        ["supplementary characters", "🌍".repeat(5000)],
        ["combining marks", "e\u0301".repeat(2500)],
    ])("counts %s as Unicode code points consistently with PocketBase", (_label, name) => {
        for (const schema of [TagCreateSchema, TagUpdateSchema]) {
            expect(schema.parse({ name }).name).toBe(name);
            expect(schema.safeParse({ name: name + "a" }).success).toBe(false);
        }
    });

    it("keeps omitted updates distinct from an explicit empty name", () => {
        expect(TagUpdateSchema.parse({})).toEqual({});
        expect(TagUpdateSchema.parse({ name: "" })).toEqual({ name: "" });
        expect(TagCreateSchema.safeParse({}).success).toBe(false);
        expect(TagCreateSchema.safeParse({ name: null }).success).toBe(false);
        expect(TagUpdateSchema.safeParse({ name: null }).success).toBe(false);
    });
});
