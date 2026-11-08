import type { RequestEvent } from "@sveltejs/kit";
import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import { beforeEach, describe, expect, it, vi } from "vitest";

const lookup = vi.hoisted(() => ({ actor: vi.fn(), remote: vi.fn() }));
vi.mock("#lib/util/activitypub_server_util.ts", () => ({ getActorResponseForHandle: lookup.actor }));
vi.mock("#lib/util/secure_fetch_util.ts", async (original) => ({
    ...await original<typeof import("#lib/util/secure_fetch_util.ts")>(),
    fetchExternalJson: lookup.remote,
}));

import { GET as profileFeed } from "./feed/+server.ts";
import { GET as profileStats } from "./stats/+server.ts";
import { GET as ownFeed } from "../../feed/+server.ts";

const share = "a".repeat(32);
const assetID = "photo0000000001";
const filePath = `/api/v1/files/assetcollect001/${assetID}/shared.jpg`;

function photoRecord(kind: "trail" | "summit_log", origin = "") {
    const id = kind === "trail" ? "trail0000000001" : "entry0000000001";
    const relation = kind === "trail" ? "trail_assets_via_trail" : "summit_log_assets_via_summit_log";
    return {
        id, name: "Shared activity", author: "actor0000000001", date: "2026-10-01", thumbnail: 0,
        photos: [`${origin}${filePath}?share=${share}`],
        expand: { [relation]: [{
            id: "links0000000001", [kind]: id, asset: assetID,
            expand: { asset: {
                id: assetID, collectionId: "assetcollect001", collectionName: "assets",
                type: "photo", storage_mode: "copy", author: "actor0000000001", file: "shared.jpg",
            } },
        }] },
    };
}

function eventFor(path: string, payload: unknown, remote: boolean) {
    lookup.actor.mockResolvedValue({ actor: {
        id: "actor0000000001", is_local: !remote, preferred_username: "hiker",
        iri: `${remote ? "https://b.example" : "https://a.example"}/api/v1/activitypub/user/hiker`,
    } });
    lookup.remote.mockResolvedValue({ ok: true, status: 200, json: payload });
    const collection = vi.fn(() => ({ getList: vi.fn().mockResolvedValue(payload) }));
    const url = new URL(`https://a.example${path}?share=${share}`);
    return {
        params: { handle: remote ? "@hiker@b.example" : "@hiker" },
        request: new Request(url), url, locals: { pb: { collection } },
    } as unknown as RequestEvent;
}

function feedWith(item: ReturnType<typeof photoRecord>, type: "trail" | "summit_log") {
    return { page: 1, perPage: 20, totalItems: 1, totalPages: 1, items: [{
        id: "feed00000000001", item: item.id, type, expand: { item },
    }] };
}

beforeEach(() => { lookup.actor.mockReset(); lookup.remote.mockReset(); });

describe("profile photo responses retain their access query", () => {
    it.each(["trail", "summit_log"] as const)("retains origin and share in remote profile feed %s photos", async (kind) => {
        const item = photoRecord(kind, "https://b.example");
        const event = eventFor("/api/v1/profile/@hiker@b.example/feed", feedWith(item, kind), true);
        const response = await responseOverHttp(await profileFeed(event));
        expect(response.status).toBe(200);
        const body = await response.json();
        expect(body.items[0].expand.item.photos).toEqual([`https://b.example${filePath}?share=${share}`]);
        const upstream = new URL(lookup.remote.mock.calls[0][0]);
        expect(upstream.origin).toBe("https://b.example");
        expect(upstream.searchParams.get("share")).toBe(share);
    });

    it.each([false, true])("retains remote statistics photos and requested links (expand=%s)", async (requested) => {
        const activity = { ...photoRecord("summit_log", "https://b.example"), source: "summit_log" };
        const event = eventFor("/api/v1/profile/@hiker@b.example/stats", [activity], true);
        if (requested) event.url.searchParams.set("expand", "summit_log_assets_via_summit_log.asset");
        const response = await responseOverHttp(await profileStats(event));
        expect(response.status).toBe(200);
        const [body] = await response.json();
        expect(body.photos).toEqual([`https://b.example${filePath}?share=${share}`]);
        if (requested) expect(body.expand.summit_log_assets_via_summit_log).toHaveLength(1);
        else expect(body.expand).not.toHaveProperty("summit_log_assets_via_summit_log");
        expect(new URL(lookup.remote.mock.calls[0][0]).searchParams.get("share")).toBe(share);
        expect(response.headers.get("Cache-Control")).toBe("private, no-store");
    });

    it.each([
        { kind: "trail", profile: false }, { kind: "summit_log", profile: false },
        { kind: "trail", profile: true }, { kind: "summit_log", profile: true },
    ] as const)("retains relative URLs and share in local feed $kind (profile=$profile)", async ({ kind, profile }) => {
        const item = photoRecord(kind);
        const event = eventFor(profile ? "/api/v1/profile/@hiker/feed" : "/api/v1/feed", feedWith(item, kind), false);
        const response = await responseOverHttp(await (profile ? profileFeed : ownFeed)(event));
        expect(response.status).toBe(200);
        expect((await response.json()).items[0].expand.item.photos).toEqual([`${filePath}?share=${share}`]);
        expect(lookup.remote).not.toHaveBeenCalled();
    });
});
