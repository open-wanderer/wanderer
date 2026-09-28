import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$lib/util/activitypub_server_util", () => ({
    getActorResponseForHandle: vi.fn(async () => ({
        actor: {
            id: "actor0000000001",
            is_local: true,
        },
    })),
}));

import { GET } from "./+server";

describe("profile statistics endpoint", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("prevents caching of an authentication-dependent response", async () => {
        const getFullList = vi.fn(async () => []);
        const event = {
            params: { handle: "@hiker" },
            url: new URL(
                "http://localhost/api/v1/profile/@hiker/stats?startDate=2026-08-01&endDate=2026-08-31",
            ),
            locals: {
                pb: {
                    collection: vi.fn(() => ({ getFullList })),
                },
            },
            fetch: vi.fn(),
        } as unknown as RequestEvent;

        const response = await GET(event);

        expect(response.status).toBe(200);
        expect(response.headers.get("Cache-Control")).toBe(
            "private, no-store",
        );
        expect(response.headers.get("Vary")).toBe("Cookie, Authorization");
    });
});
