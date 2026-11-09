import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("#lib/util/activitypub_server_util.ts", () => ({
    getActorResponseForHandle: vi.fn(async () => ({
        actor: {
            id: "actor0000000001",
            is_local: true,
        },
    })),
}));

vi.mock("#lib/util/secure_fetch_util.ts", () => ({
    DownloadError: class extends Error {},
    fetchExternalJson: vi.fn(),
}));

import { getActorResponseForHandle } from "#lib/util/activitypub_server_util.ts";
import { fetchExternalJson } from "#lib/util/secure_fetch_util.ts";
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

it("keeps asset thumbnail URLs and legacy filenames usable for federated profiles", async () => {
    vi.mocked(getActorResponseForHandle).mockResolvedValueOnce({ actor: {
        id: "actor0000000001", is_local: false, iri: "https://b.example/api/v1/activitypub/user/hiker",
        preferred_username: "hiker", domain: "b.example",
    } } as any);
    vi.mocked(fetchExternalJson).mockResolvedValueOnce({ ok: true, status: 200, json: { hits: [
        { id: "empty", thumbnail: "", iri: "" },
        { id: "absent", iri: "" },
        { id: "absolute", thumbnail: "https://b.example/photo.jpg", iri: "" },
        { id: "other-origin", thumbnail: "https://a.example/photo.jpg", iri: "" },
        { id: "other-protocol", thumbnail: "http://b.example/photo.jpg", iri: "" },
        { id: "other-port", thumbnail: "https://b.example:8443/photo.jpg", iri: "" },
        { id: "asset", thumbnail: "/api/v1/files/assets/photo/image.jpg", iri: "" },
        { id: "legacy", thumbnail: "image.jpg", iri: "" },
    ] } } as any);
    const response = await POST(profileTrailsEvent({ q: "", options: {} }, vi.fn()));
    expect(response.status).toBe(200);
    const result = await response.json();
    expect(result.hits.map((hit: { thumbnail: string }) => hit.thumbnail)).toEqual([
        "", "", "https://b.example/photo.jpg", "", "", "", "https://b.example/api/v1/files/assets/photo/image.jpg",
        "https://b.example/api/v1/files/trails/legacy/image.jpg",
    ]);
});

it("returns a bare linked filename for an older peer's profile thumbnail mapping", async () => {
    const search = vi.fn(async () => ({ hits: [{ id: "trail0000000001", thumbnail: "/api/v1/files/assets/asset0000000001/image.png" }] }));
    const response = await POST(profileTrailsEvent({ q: "", options: {} }, search));
    const { hits: [trail] } = await response.json();
    expect(trail.thumbnail).toBe("image.png");
    expect(`https://a.example/api/v1/files/trails/${trail.id}/${trail.thumbnail}`).toBe(
        "https://a.example/api/v1/files/trails/trail0000000001/image.png",
    );
});
