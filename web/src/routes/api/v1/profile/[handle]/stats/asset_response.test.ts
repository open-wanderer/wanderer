import type { RequestEvent } from "@sveltejs/kit";
import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import PocketBase from "pocketbase";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("#lib/util/activitypub_server_util.ts", () => ({
    getActorResponseForHandle: vi.fn(async () => ({ actor: { id: "actor0000000001", is_local: true } })),
}));

import { GET } from "./+server.ts";

const logID = "log000000000001";
const share = "a".repeat(32);

function setup(count: number, token = "owner-token", shareToken?: string, proofStatus = 200) {
    const links = Array.from({ length: count }, (_, index) => {
        const id = `asset${String(index).padStart(10, "0")}`;
        return {
            id: `links${String(index).padStart(10, "0")}`, summit_log: logID, asset: id,
            created: new Date(Date.UTC(2026, 9, 1) + index).toISOString(),
            expand: { asset: {
                id, collectionId: "assetcollect001", collectionName: "assets", type: "photo",
                author: "actor0000000001", storage_mode: "copy", file: `photo${index}.jpg`,
                metadata: { caption: "An ordinary photo caption" },
            } },
        };
    });
    const log = {
        id: logID, collectionId: "summit_logs", collectionName: "summit_logs",
        author: "actor0000000001", trail: "trail0000000001", date: "2026-10-01",
        // Empty PocketBase back-relations are omitted on the actual wire.
        ...(count ? { expand: { summit_log_assets_via_summit_log: links.slice(0, 1000) } } : {}),
    };
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input));
        const collection = url.pathname.match(/^\/api\/collections\/([^/]+)\/records$/)?.[1];
        const send = (body: unknown, status = 200) => new Response(JSON.stringify(body), {
            status, headers: { "Content-Type": "application/json" },
        });
        const page = Number(url.searchParams.get("page") ?? 1);
        const perPage = Number(url.searchParams.get("perPage") ?? 500);
        const list = (items: unknown[]) => send({
            page, perPage, totalItems: items.length, totalPages: Math.ceil(items.length / perPage),
            items: items.slice((page - 1) * perPage, page * perPage),
        });
        if (collection === "subcategories" || collection === "trails") return list([]);
        const authorization = new Headers(init?.headers).get("Authorization");
        expect(authorization === "owner-token" || url.searchParams.get("share") === share).toBe(true);
        if (collection === "summit_logs") {
            expect(url.searchParams.get("filter")).toContain("author='actor0000000001'");
            expect(url.searchParams.get("expand")).toContain("summit_log_assets_via_summit_log.asset");
            return list([log]);
        }
        if (collection === "summit_log_assets") {
            expect(url.searchParams.get("filter")).toBe(`summit_log = "${logID}"`);
            expect(url.searchParams.get("expand")).toBe("asset");
            expect(url.searchParams.get("sort")).toBe("+created,+id");
            expect(perPage).toBe(500);
            if (proofStatus !== 200) return send({ message: "Unavailable" }, proofStatus);
            return list(links);
        }
        throw new Error(`Unexpected PocketBase collection: ${collection}`);
    });
    vi.stubGlobal("fetch", fetch);
    const pb = new PocketBase("https://a.example");
    pb.authStore.save(token);
    const url = new URL("https://a.example/api/v1/profile/@hiker/stats?startDate=2026-10-01");
    if (shareToken) url.searchParams.set("share", shareToken);
    const event = { params: { handle: "@hiker" }, url, request: new Request(url), locals: { pb }, fetch } as unknown as RequestEvent;
    return { event, fetch, links };
}

afterEach(() => vi.unstubAllGlobals());

describe("profile statistics photo responses", () => {
    it.each([
        { token: "owner-token", shareToken: undefined },
        { token: "", shareToken: share },
    ])("completes 1110 linked photos with the same authorization ($token)", async ({ token, shareToken }) => {
        const { event, fetch } = setup(1110, token, shareToken);
        const response = await responseOverHttp(await GET(event));
        expect(response.status).toBe(200);
        expect(response.headers.get("Cache-Control")).toBe("private, no-store");
        expect(response.headers.get("Vary")).toBe("Cookie, Authorization");
        const body = await response.json();
        expect(body).toHaveLength(1);
        expect(body[0].source).toBe("summit_log");
        expect(body[0].photos).toHaveLength(1110);
        expect(body[0].photos[1109]).toBe(`/api/v1/files/assetcollect001/asset0000001109/photo1109.jpg${shareToken ? `?share=${shareToken}` : ""}`);
        expect(body[0].expand).not.toHaveProperty("summit_log_assets_via_summit_log");
        expect(body[0].expand).not.toHaveProperty("assets_via_summit_log");
        expect(JSON.stringify(body).length).toBeLessThan(120_000);
        const pages = fetch.mock.calls.filter(([url]) => String(url).includes("/summit_log_assets/records"));
        expect(pages).toHaveLength(3);
        for (const [url, init] of pages) {
            expect(new URL(String(url)).searchParams.get("share")).toBe(shareToken ?? null);
            expect(new Headers(init?.headers).get("Authorization")).toBe(token || null);
        }
    });

    it("reports empty photos only after an authorized independent empty query", async () => {
        const { event, fetch } = setup(0);
        const response = await responseOverHttp(await GET(event));
        expect(response.status).toBe(200);
        const [activity] = await response.json();
        expect(activity.photos).toEqual([]);
        expect(activity.expand).not.toHaveProperty("summit_log_assets_via_summit_log");
        expect(activity.expand).not.toHaveProperty("assets_via_summit_log");
        expect(fetch.mock.calls.filter(([url]) => String(url).includes("/summit_log_assets/records"))).toHaveLength(1);
    });

    it("preserves absent photos when the independent empty query is unavailable", async () => {
        const { event, fetch } = setup(0, "owner-token", undefined, 503);
        const response = await responseOverHttp(await GET(event));
        expect(response.status).toBe(200);
        const [activity] = await response.json();
        expect(activity).not.toHaveProperty("photos");
        expect(activity.expand).not.toHaveProperty("summit_log_assets_via_summit_log");
        expect(fetch.mock.calls.filter(([url]) => String(url).includes("/summit_log_assets/records"))).toHaveLength(1);
    });
});

it("keeps canonical statistics links only when explicitly requested", async () => {
    const { event } = setup(2);
    event.url.searchParams.set("expand", "summit_log_assets_via_summit_log.asset");
    const response = await responseOverHttp(await GET(event));
    const [activity] = await response.json();
    expect(activity.photos).toHaveLength(2);
    expect(activity.expand.summit_log_assets_via_summit_log).toHaveLength(2);
});
