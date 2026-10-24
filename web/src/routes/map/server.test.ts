import { isRedirect } from "@sveltejs/kit";
import { describe, expect, it } from "vitest";
import { GET } from "./+server";

describe("map shortcut", () => {
    it.each(["", "?lat=47&lon=8", "?view=table&tl_lat=49&tl_lon=11&br_lat=45&br_lon=5&category=hiking"])("keeps existing parameters when opening map view: %s", (query) => {
        try {
            GET({ url: new URL(`http://localhost/map${query}`) } as Parameters<typeof GET>[0]);
            throw new Error("Expected a redirect");
        } catch (result) {
            expect(isRedirect(result)).toBe(true);
            if (!isRedirect(result)) throw result;
            expect(result.status).toBe(307);
            const target = new URL(result.location, "http://localhost");
            expect(target.pathname).toBe("/trails");
            expect(target.searchParams.get("view")).toBe("map");
            for (const [key, value] of new URLSearchParams(query)) {
                if (key !== "view") expect(target.searchParams.get(key)).toBe(value);
            }
        }
    });
});
