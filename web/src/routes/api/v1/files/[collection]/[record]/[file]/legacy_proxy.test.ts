import type { RequestEvent } from "@sveltejs/kit";
import { expect, it, vi } from "vitest";
import { GET } from "./+server.ts";

it.each(["track.gpx", "old-photo.jpg"])("uses the authoritative file endpoint once without metadata scans for %s", async (file) => {
    const collection = vi.fn(() => { throw new Error("unexpected collection lookup"); });
    const fetch = vi.fn().mockResolvedValue(new Response("authorized file", { headers: { ETag: "fresh" } }));
    const share = "a".repeat(32);
    const event = {
        params: { collection: "trails", record: "trail0000000001", file },
        url: new URL(`https://a.example/file?share=${share}`),
        request: new Request("https://a.example/file", { headers: { "If-None-Match": "old" } }),
        locals: { pb: { collection, authStore: { token: "reader" }, buildURL: (path: string) => `https://b.example/${path}` } }, fetch,
    } as unknown as RequestEvent;
    const response = await GET(event);
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("authorized file");
    expect(collection).not.toHaveBeenCalled();
    expect(fetch).toHaveBeenCalledOnce();
    expect(fetch).toHaveBeenCalledWith(`https://b.example/api/files/trails/trail0000000001/${file}?share=${share}`, {
        headers: { Authorization: "Bearer reader", "If-None-Match": "old" }, signal: event.request.signal,
    });
});
