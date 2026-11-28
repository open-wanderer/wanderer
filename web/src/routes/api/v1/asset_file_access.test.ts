import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { GET as assetFile } from "./assets/[id]/file/+server.ts";
import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import { GET as storedFile } from "./files/[collection]/[record]/[file]/+server.ts";

const id = "hcykc33z7upq76u";
const share = "a".repeat(32);

function fileEvent(token: string, status = 200) {
    const fetch = vi.fn().mockResolvedValue(new Response(status === 304 ? null : status === 200 ? "photo" : "denied", {
        status,
        headers: { "Cache-Control": "private, max-age=3600" },
    }));
    const event = {
        params: { id, collection: "assets", record: id, file: "photo.jpg" },
        url: new URL(`https://a.example/photo?share=${share}&thumb=100x100`),
        request: new Request("https://a.example/photo"),
        locals: {
            pb: {
                authStore: { token },
                buildURL: (path: string) => `https://b.example/${path}`,
            },
        },
        fetch,
    } as unknown as RequestEvent;
    return { event, fetch };
}

describe.each([
    ["linked photo", assetFile, `assets/${id}/file`],
    ["stored photo", storedFile, `api/files/assets/${id}/photo.jpg`],
] as const)("%s proxy access", (_name, handler, path) => {
    it.each(["", "owner-token"])("forwards share and user authorization (%s)", async (token) => {
        const { event, fetch } = fileEvent(token);
        const response = await responseOverHttp(await handler(event));

        expect(response.status).toBe(200);
        expect(await response.text()).toBe("photo");
        expect(response.headers.get("Cache-Control")).toBe("private, max-age=3600");
        expect(fetch).toHaveBeenCalledOnce();
        const [url, options] = fetch.mock.calls[0];
        expect(new URL(url).pathname).toBe(`/${path}`);
        expect(new URL(url).searchParams.get("share")).toBe(share);
        expect(new URL(url).searchParams.get("thumb")).toBe("100x100");
        expect(options.headers).toEqual(token ? { Authorization: `Bearer ${token}` } : {});
    });

    it("checks a new request after its share was revoked", async () => {
        const { event, fetch } = fileEvent("");
        expect((await handler(event)).status).toBe(200);
        fetch.mockResolvedValueOnce(new Response("denied", { status: 404 }));
        expect((await handler(event)).status).toBe(404);
        expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("forwards validators and preserves a bodyless 304 response", async () => {
        const { event, fetch } = fileEvent("owner-token", 304);
        event.request.headers.set("If-None-Match", '"photo-v1"');
        event.request.headers.set("If-Modified-Since", "Sun, 11 Oct 2026 10:00:00 GMT");
        const response = await responseOverHttp(await handler(event));
        expect(response.status).toBe(304);
        expect(response.body).toBeNull();
        expect(response.headers.get("Cache-Control")).toBe("private, max-age=3600");
        expect(fetch.mock.calls[0][1].headers).toEqual({
            Authorization: "Bearer owner-token", "If-None-Match": '"photo-v1"',
            "If-Modified-Since": "Sun, 11 Oct 2026 10:00:00 GMT",
        });
    });
});

it("preserves denial of stored photo access after a share is revoked", async () => {
    const { event } = fileEvent("", 404);
    expect((await storedFile(event)).status).toBe(404);
});

it("preserves denial of linked photo access after a share is revoked", async () => {
    const { event } = fileEvent("", 404);
    const response = await assetFile(event);
    expect(response.status).toBe(404);
    expect(await response.text()).toBe("denied");
});


it("rejects invalid share tokens before contacting the backend", async () => {
    const { event, fetch } = fileEvent("");
    event.url.searchParams.set("share", "invalid");
    await expect(assetFile(event)).rejects.toMatchObject({ status: 400 });
    expect(fetch).not.toHaveBeenCalled();
});
