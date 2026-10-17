import { describe, expect, it, vi } from "vitest";
import { searchLocations } from "./search_store";

describe("location search query interpretation", () => {
    it("passes the requested suggestion limit to the real geocoding route", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => Response.json({ features: [] }));
        expect(await searchLocations("Bern", 1, request)).toEqual([]);
        const url = new URL(String(request.mock.calls[0][0]), "http://localhost");
        expect(url.pathname).toBe("/api/v1/geocoding/search");
        expect(url.searchParams.get("q")).toBe("Bern");
        expect(url.searchParams.get("limit")).toBe("1");
    });
});
