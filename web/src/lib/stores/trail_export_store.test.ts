import { expect, it, vi } from "vitest";
import { ownedTrailExportPages } from "./trail_export_store";

it.each([false, true])("pages only the owner's trails with optional photo expansion (photos=%s)", async photos => {
    const request = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://a.example");
        expect(url.searchParams.get("filter")).toBe('author = "ownactor0000001"');
        expect(url.searchParams.get("perPage")).toBe("100");
        expect(url.searchParams.get("sort")).toBe("id");
        expect(url.searchParams.get("photos")).toBe(String(photos));
        expect(url.searchParams.get("expand")?.split(",")).toContain("waypoints_via_trail");
        return Response.json({ items: Array.from({ length: url.searchParams.get("page") === "1" ? 100 : 1 }, (_, i) => ({ id: `${url.searchParams.get("page")}-${i}` })), totalPages: 2 });
    });
    const sizes: number[] = [];
    for await (const page of ownedTrailExportPages("ownactor0000001", photos, request)) sizes.push(page.length);
    expect(sizes).toEqual([100, 1]);
    expect(request).toHaveBeenCalledTimes(2);
});
