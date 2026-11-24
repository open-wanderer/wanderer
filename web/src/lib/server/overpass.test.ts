import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";

vi.mock("$app/env", () => ({ version: "test" }));
vi.mock("#lib/server/url.ts", () => ({ resolveBaseUrl: () => "https://overpass-api.de" }));

import { fetchOverpass } from "./overpass";

describe("fetchOverpass", () => {
    it("sends a descriptive User-Agent", async () => {
        const fetch = vi.fn(async (_url: string, _init?: RequestInit) => new Response("{}", { status: 200 }));
        const event = { fetch } as unknown as RequestEvent;

        await fetchOverpass(event, new URLSearchParams({ data: "[out:json];node(1);out;" }));

        const init = fetch.mock.calls[0][1]!;
        expect(new Headers(init.headers).get("User-Agent")).toBe(
            "wanderer/test (+https://github.com/open-wanderer/wanderer)",
        );
    });

    it("does not retry a failed request", async () => {
        const fetch = vi.fn(async () => {
            throw new TypeError("fetch failed");
        });
        const event = { fetch } as unknown as RequestEvent;

        await expect(fetchOverpass(event, new URLSearchParams({ data: "x" }))).rejects.toThrow("fetch failed");
        expect(fetch).toHaveBeenCalledTimes(1);
    });
});
