import { TagCreateSchema } from "./api/tag_schema";
import { Tag } from "./tag";
import { describe, expect, it } from "vitest";

describe("new UI tags before trail-form validation", () => {
    it.each([
        ["pasted tab", "Grü\tnezi", "Grünezi"],
        ["DEL and C0", "a\u007f\u001fb", "ab"],
        ["control-only name", "\t\u007f", ""],
        ["unmodified plain text", "  e\u0301 👩‍👩‍👧‍👦 &lt;img&gt; \u0085\u2028  ", "  e\u0301 👩‍👩‍👧‍👦 &lt;img&gt; \u0085\u2028  "],
        ["length checked after cleaning", "🌍".repeat(5000) + "\t", "🌍".repeat(5000)],
    ])("passes the editor's strict tag validation for %s", (_label, name, expectedName) => {
        const tag = new Tag(name);
        expect(TagCreateSchema.parse(tag).name).toBe(expectedName);
        expect(tag.name).toBe(expectedName);
    });

    it("still rejects control characters in direct API inputs", () => {
        expect(TagCreateSchema.safeParse({ name: "Grü\tnezi" }).success).toBe(false);
    });
});
