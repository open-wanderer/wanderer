import { describe, expect, it, vi } from "vitest";
import { profile_follows_index, profile_stats_index } from "./profile_store";

describe("profile statistics requests", () => {
    it("bypasses browser caches because the response depends on authentication", async () => {
        const request = vi.fn(
            async (
                _url: RequestInfo | URL,
                _config?: RequestInit,
            ): Promise<Response> =>
                new Response(JSON.stringify([]), {
                    status: 200,
                    headers: { "Content-Type": "application/json" },
                }),
        );

        await profile_stats_index(
            "@hiker",
            {
                startDate: "2026-08-01",
                endDate: "2026-08-31",
                category: [],
                subcategory: [],
            },
            request,
        );

        expect(request).toHaveBeenCalledOnce();
        expect(request.mock.calls[0][1]).toMatchObject({
            method: "GET",
            cache: "no-store",
        });
    });
});

describe("profile follows requests", () => {
    function respondWith(items: string[], next = "") {
        return async (): Promise<Response> =>
            new Response(
                JSON.stringify({
                    page: 1,
                    perPage: items.length,
                    totalItems: 10,
                    totalPages: 5,
                    items: items.map((id) => ({ id })),
                    next,
                }),
                { status: 200, headers: { "Content-Type": "application/json" } },
            );
    }

    it("returns only the requested page, whatever else was loaded before", async () => {
        await profile_follows_index("@hiker", "following", 1, respondWith(["following-1"]));
        const followers = await profile_follows_index(
            "@hiker",
            "followers",
            2,
            respondWith(["followers-2"], "https://remote.example/followers?page=3"),
            "https://remote.example/followers?page=2",
        );

        expect(followers.items.map((a) => a.id)).toEqual(["followers-2"]);
        expect(followers.next).toBe("https://remote.example/followers?page=3");
    });
});
