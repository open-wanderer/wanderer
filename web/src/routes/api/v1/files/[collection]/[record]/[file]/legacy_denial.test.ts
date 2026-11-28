import type { RequestEvent } from "@sveltejs/kit";
import { expect, it, vi } from "vitest";
import { GET } from "./+server.ts";

it.each([200, 404])("passes the backend file status %s through", async (fileStatus) => {
    const fetch = vi.fn().mockResolvedValue(new Response("file response", { status: fileStatus }));
    const event = {
        params: { collection: "trails", record: "trail0000000001", file: "track.gpx" },
        url: new URL("https://a.example/file"), request: new Request("https://a.example/file"),
        locals: { pb: {
            authStore: { token: "reader" }, buildURL: (path: string) => `https://b.example/${path}`,
        } }, fetch,
    } as unknown as RequestEvent;
    const response = await GET(event);
    expect(response.status).toBe(fileStatus);
    expect(await response.text()).toBe("file response");
    expect(fetch).toHaveBeenCalledOnce();
});
