import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { describe, expect, it, vi } from "vitest";
import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import { PUT } from "./+server.ts";

const actor = "caller000000001";
const owner = "owners000000001";
const trail = "trail0000000001";
const photo = "photo0000000001";

function targetEvent(kind: "waypoint" | "summit_log" | "both", role: "author" | "editor" | "former-editor" | "trail-owner" | "author-editor") {
    const share = vi.fn().mockResolvedValue(role === "editor" || role === "author-editor" ? [{ id: "share" }] : []);
    const create = vi.fn().mockResolvedValue({ id: "link00000000001" });
    const collection = vi.fn((name: string) => {
        if (name === "trails") return { getOne: vi.fn().mockResolvedValue({ id: trail, author: role === "trail-owner" ? actor : owner }) };
        if (name === "waypoints") return { getOne: vi.fn().mockResolvedValue({ id: "point0000000001", author: actor, trail }) };
        if (name === "summit_logs") return { getOne: vi.fn().mockResolvedValue({ id: "entry0000000001", author: role === "author" || role === "author-editor" ? actor : owner, trail }) };
        if (name === "trail_share") return { getFullList: share };
        if (name === "assets") return { getOne: vi.fn().mockResolvedValue({ id: photo, type: "photo", author: actor }) };
        return { getFullList: vi.fn().mockResolvedValue([]), create };
    });
    const body = new FormData();
    body.append("trail", trail);
    body.append("assetIds", photo);
    if (kind !== "summit_log") body.append("waypoint", "point0000000001");
    if (kind !== "waypoint") body.append("summit_log", "entry0000000001");
    const request = new Request("https://a.example/api/v1/assets", { method: "PUT", body });
    const event = { request, locals: { user: { actor }, pb: { collection, filter: (value: string) => value } } } as unknown as RequestEvent;
    return { event, create, share };
}

describe("photo target permissions", () => {
    it("does not derive waypoint photo rights from the waypoint author", async () => {
        const { event, create } = targetEvent("waypoint", "former-editor");
        const response = await responseOverHttp(await PUT(event));
        expect(response.status).toBe(403);
        expect(create).not.toHaveBeenCalled();
    });

    it.each(["editor", "trail-owner"] as const)("permits a waypoint photo through its current %s", async (role) => {
        const { event, create } = targetEvent("waypoint", role);
        expect((await PUT(event)).status).toBe(200);
        expect(create).toHaveBeenCalledWith({ asset: photo, waypoint: "point0000000001" }, { requestKey: null });
    });

    it("does not grant summit-log photo edits through a trail edit share", async () => {
        const { event, create, share } = targetEvent("summit_log", "editor");
        expect((await PUT(event)).status).toBe(403);
        expect(create).not.toHaveBeenCalled();
        expect(share).not.toHaveBeenCalled();
    });

    it.each(["author", "trail-owner"] as const)("permits summit-log photos for its %s", async (role) => {
        const { event, create, share } = targetEvent("summit_log", role);
        expect((await PUT(event)).status).toBe(200);
        expect(create).toHaveBeenCalledWith({ asset: photo, summit_log: "entry0000000001" }, { requestKey: null });
        expect(share).not.toHaveBeenCalled();
    });

    it("permits an entry author after access to its private trail ends", async () => {
        const paths: string[] = [];
        const fetch = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            const url = new URL(String(input));
            paths.push(url.pathname);
            if (url.pathname.includes("/trails/records/")) return Response.json({ message: "Not found" }, { status: 404 });
            if (url.pathname.endsWith("/summit_logs/records/entry0000000001")) return Response.json({ id: "entry0000000001", author: actor, trail });
            if (url.pathname.endsWith(`/assets/records/${photo}`)) return Response.json({ id: photo, type: "photo", author: actor });
            expect(url.pathname).toBe("/api/collections/summit_log_assets/records");
            return options?.method === "POST"
                ? Response.json({ id: "link00000000001", asset: photo, summit_log: "entry0000000001" })
                : Response.json({ page: 1, perPage: 500, totalItems: 0, totalPages: 0, items: [] });
        });
        vi.stubGlobal("fetch", fetch);
        try {
            const body = new FormData();
            body.append("trail", trail);
            body.append("summit_log", "entry0000000001");
            body.append("assetIds", photo);
            const event = {
                request: new Request("https://a.example/api/v1/assets", { method: "PUT", body }),
                locals: { user: { actor }, pb: new PocketBase("https://b.example") },
            } as unknown as RequestEvent;
            const response = await responseOverHttp(await PUT(event));
            expect(response.status).toBe(200);
            expect(await response.json()).toEqual([expect.objectContaining({ id: photo })]);
            expect(paths).not.toContain(`/api/collections/trails/records/${trail}`);
            expect(paths).toContain("/api/collections/summit_log_assets/records");
        } finally {
            vi.unstubAllGlobals();
        }
    });

    it.each([{ role: "author", status: 403 }, { role: "author-editor", status: 200 }] as const)("checks both child target permissions ($role)", async ({ role, status }) => {
        const { event, create } = targetEvent("both", role);
        expect((await PUT(event)).status).toBe(status);
        expect(create).toHaveBeenCalledTimes(status === 200 ? 2 : 0);
    });
});
