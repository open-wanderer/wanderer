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

import { POST } from "./+server";

function profileTrailsEvent(
    body: unknown,
    search: ReturnType<typeof vi.fn>,
    user?: { id: string },
) {
    return {
        params: { handle: "@hiker" },
        url: new URL("http://localhost/api/v1/profile/@hiker/trails"),
        request: new Request("http://localhost/api/v1/profile/@hiker/trails", {
            method: "POST",
            body: JSON.stringify(body),
        }),
        locals: {
            user,
            ms: { index: vi.fn(() => ({ search })) },
            pb: {
                filter: vi.fn(() => ""),
                collection: vi.fn(() => ({
                    getFullList: vi.fn(async () => []),
                })),
            },
        },
        fetch: vi.fn(),
    } as unknown as RequestEvent;
}

describe("profile trails endpoint", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("sends no null filter entries for an anonymous visitor", async () => {
        const search = vi.fn(async () => ({ hits: [] }));
        const event = profileTrailsEvent(
            { q: "", options: { hitsPerPage: 12, page: 1 } },
            search,
        );

        const response = await POST(event);

        expect(response.status).toBe(200);
        expect(search).toHaveBeenCalledWith("", {
            hitsPerPage: 12,
            page: 1,
            filter: ["author = actor0000000001"],
        });
    });

    it("keeps a client filter alongside the author filter", async () => {
        const search = vi.fn(async () => ({ hits: [] }));
        const event = profileTrailsEvent(
            { q: "", options: { filter: "difficulty = 1" } },
            search,
        );

        await POST(event);

        expect(search).toHaveBeenCalledWith("", {
            filter: ["difficulty = 1", "author = actor0000000001"],
        });
    });

    it("sends no null filter entries for a signed-in visitor", async () => {
        const search = vi.fn(async () => ({ hits: [] }));
        const event = profileTrailsEvent(
            { q: "", options: {} },
            search,
            { id: "user00000000001" },
        );

        await POST(event);

        expect(search).toHaveBeenCalledWith("", {
            filter: ["author = actor0000000001"],
        });
    });
});
