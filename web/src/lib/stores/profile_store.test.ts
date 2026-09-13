import { describe, expect, it, vi } from "vitest";
import { profile_stats_index } from "./profile_store";

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
