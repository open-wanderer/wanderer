import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { afterEach, expect, it, vi } from "vitest";
import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import { GET } from "./+server.ts";

afterEach(() => vi.unstubAllGlobals());

it("returns metadata-only list detail without trails or asset-link queries", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input));
        expect(url.pathname).toBe("/remote/list/list00000000001");
        expect(url.searchParams.get("expand")).toBe("author");
        return Response.json({ id: "list00000000001", name: "Day trips", expand: { author: { id: "actor0000000001" } } });
    });
    vi.stubGlobal("fetch", fetch);
    const pb = new PocketBase("https://a.example");
    const collection = vi.spyOn(pb, "collection");
    const url = new URL("https://a.example/api/v1/list/list00000000001?expand=author");
    const event = { url, params: { id: "list00000000001" }, fetch, request: new Request(url), locals: { pb } } as unknown as RequestEvent;
    const response = await responseOverHttp(await GET(event));
    expect(response.status).toBe(200);
    const body = await response.json();
    expect(body.expand.author.id).toBe("actor0000000001");
    expect(body.expand).not.toHaveProperty("trails");
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(collection).not.toHaveBeenCalled();
});
