import { describe, expect, it, vi } from "vitest";
import { tags_index } from "./tag_store";

describe("tag search", () => {
    it.each([
        ["Grüezi 🌍", 'name ~ "Grüezi 🌍"'],
        ["O'Brien", 'name ~ "O\'Brien"'],
        ["x' || name != '' || name~'", 'name ~ "x\' || name != \'\' || name~\'"'],
        ['x" || name != "" || name~"', 'name ~ "x\\" || name != \\"\\" || name~\\""'],
        ["tag\\name", 'name ~ "tag\\\\name"'],
    ])("searches for %s as one quoted value", async (name, filter) => {
        const request = vi.fn(async (_input: RequestInfo | URL, _options?: RequestInit) => Response.json({ items: [{ id: "tag", name }] }));
        const result = await tags_index(name, request);
        const url = new URL(String(request.mock.calls[0][0]), "https://example.com");
        expect(url.pathname).toBe("/api/v1/tag");
        expect(url.searchParams.get("filter")).toBe(filter);
        expect(request.mock.calls[0][1]).toEqual({ method: "GET" });
        expect(result.items[0].name).toBe(name);
    });
    it("reports API failures", async () => {
        const request = vi.fn(async () => Response.json({ message: "Denied", detail: "Tag list is private" }, { status: 403 }));
        await expect(tags_index("test", request)).rejects.toMatchObject({ status: 403, message: "Denied" });
    });
});
