import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { GET as assetFile } from "./assets/[id]/file/+server";
import { GET as storedFile } from "./files/[collection]/[record]/[file]/+server";

const id = "hcykc33z7upq76u";
const share = "a".repeat(32);

function fileEvent(token: string, status = 200) {
    const fetch = vi.fn().mockResolvedValue(new Response(status === 200 ? "photo" : "denied", {
        status,
        headers: { "Cache-Control": "private, no-cache, must-revalidate" },
    }));
    const event = {
        params: { id, collection: "assets", record: id, file: "photo.jpg" },
        url: new URL(`https://wanderer.example/photo?share=${share}&thumb=100x100`),
        locals: {
            pb: {
                authStore: { token },
                buildURL: (path: string) => `https://db.example/${path}`,
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
        const response = await handler(event);

        expect(response.status).toBe(200);
        expect(await response.text()).toBe("photo");
        expect(response.headers.get("Cache-Control")).toBe("private, no-cache, must-revalidate");
        expect(fetch).toHaveBeenCalledOnce();
        const [url, options] = fetch.mock.calls[0];
        expect(new URL(url).pathname).toBe(`/${path}`);
        expect(new URL(url).searchParams.get("share")).toBe(share);
        expect(new URL(url).searchParams.get("thumb")).toBe("100x100");
        expect(options.headers).toEqual(token ? { Authorization: `Bearer ${token}` } : {});
    });
});

it("preserves denial of stored photo access after a share is revoked", async () => {
    const { event } = fileEvent("", 404);
    expect((await storedFile(event)).status).toBe(404);
});

it("preserves denial of linked photo access after a share is revoked", async () => {
    const { event } = fileEvent("", 404);
    await expect(assetFile(event)).rejects.toMatchObject({ status: 404 });
});
