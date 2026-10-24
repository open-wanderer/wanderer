import { env } from "$env/dynamic/private";
import type { RequestEvent } from "@sveltejs/kit";
import { afterEach, describe, expect, it, vi } from "vitest";
import { INBOX_BODY_LIMIT, LOGIN_BODY_LIMIT, REQUEST_BODY_LIMIT, UPLOAD_BODY_LIMIT, limitRequestBodies, readBoundedRequestBody } from "./request_body";

afterEach(() => { delete env.BODY_SIZE_LIMIT; vi.restoreAllMocks(); });

function event(request: Request): RequestEvent {
    return { request, url: new URL(request.url) } as RequestEvent;
}

function streamedRequest(path: string, bytes: number, headers?: HeadersInit) {
    let remaining = bytes;
    const cancelled = vi.fn();
    const stream = new ReadableStream<Uint8Array>({
        pull(controller) {
            if (!remaining) return controller.close();
            const size = Math.min(16 * 1024, remaining);
            controller.enqueue(new Uint8Array(size));
            remaining -= size;
        },
        cancel: cancelled,
    });
    const init: RequestInit & { duplex: "half" } = { method: "POST", body: stream, duplex: "half", headers };
    return { request: new Request("https://local.example" + path, init), cancelled, remaining: () => remaining };
}

describe("request body limits", () => {
    it.each([
        ["/api/v1/auth/login", LOGIN_BODY_LIMIT],
        ["/api/v1/activitypub/user/alice/inbox", INBOX_BODY_LIMIT],
        ["/api/v1/search/multi", REQUEST_BODY_LIMIT],
    ])("rejects oversized chunked bodies at %s before resolution", async (path, limit) => {
        const { request, cancelled, remaining } = streamedRequest(path as string, (limit as number) + 64 * 1024);
        const resolve = vi.fn(async () => new Response("resolved"));
        const response = await limitRequestBodies({ event: event(request), resolve });
        expect(response.status).toBe(413);
        expect(await response.json()).toEqual({ message: "request_body_too_large" });
        expect(resolve).not.toHaveBeenCalled();
        expect(cancelled).toHaveBeenCalledOnce();
        expect(remaining()).toBeGreaterThan(0);
    });

    it("checks declared length without consuming the request", async () => {
        const request = new Request("https://local.example/api/v1/auth/login", { method: "POST", body: "small", headers: { "content-length": String(LOGIN_BODY_LIMIT + 1) } });
        const response = await limitRequestBodies({ event: event(request), resolve: vi.fn() });
        expect(response.status).toBe(413);
        expect(request.bodyUsed).toBe(false);
    });

    it("passes internal sub-requests while bounding the same client body", async () => {
        const body = JSON.stringify({ encoded_polyline: "x".repeat(REQUEST_BODY_LIMIT) });
        const request = () => new Request("https://local.example/api/v1/valhalla/height", { method: "POST", body });
        const resolve = vi.fn(async (input: RequestEvent) => {
            expect(await input.request.text()).toBe(body);
            return new Response("accepted");
        });
        const internal = { ...event(request()), isSubRequest: true };
        expect((await limitRequestBodies({ event: internal, resolve })).status).toBe(200);
        expect(resolve).toHaveBeenCalledOnce();

        const response = await limitRequestBodies({ event: event(request()), resolve });
        expect(response.status).toBe(413);
        expect(await response.json()).toEqual({ message: "request_body_too_large" });
        expect(resolve).toHaveBeenCalledOnce();
    });

    it("counts actual bytes when Content-Length understates the body", async () => {
        const { request, cancelled } = streamedRequest("/api/v1/auth/login", LOGIN_BODY_LIMIT + 1, { "content-length": "1" });
        await expect(readBoundedRequestBody(request, LOGIN_BODY_LIMIT)).rejects.toMatchObject({ status: 413 });
        expect(cancelled).toHaveBeenCalledOnce();
    });

    it("preserves signed UTF-8 bytes and headers", async () => {
        const raw = '{ "actor": "https://remote.example/a", "content": "\u00e4\u2603", "extra": 1 }\n';
        const request = new Request("https://local.example/api/v1/activitypub/user/alice/inbox", { method: "POST", body: raw, headers: { signature: "original-signature", digest: "original-digest" } });
        const resolve = vi.fn(async (input: RequestEvent) => {
            expect(await input.request.text()).toBe(raw);
            expect(input.request.headers.get("signature")).toBe("original-signature");
            expect(input.request.headers.get("digest")).toBe("original-digest");
            return new Response("accepted");
        });
        expect((await limitRequestBodies({ event: event(request), resolve })).status).toBe(200);
    });

    it("leaves multipart uploads unread until resolution", async () => {
        const { request, remaining } = streamedRequest("/api/v1/trail/form", UPLOAD_BODY_LIMIT, { "content-type": "multipart/form-data; boundary=test", "content-length": String(UPLOAD_BODY_LIMIT) });
        const put = new Request(request, { method: "PUT" });
        const resolve = vi.fn(async () => new Response("Unauthorized", { status: 401 }));
        expect((await limitRequestBodies({ event: event(put), resolve })).status).toBe(401);
        expect(remaining()).toBeGreaterThan(UPLOAD_BODY_LIMIT - 128 * 1024);
        expect(resolve).toHaveBeenCalledOnce();
    });

    it("rejects a declared multipart body above the upload limit", async () => {
        const request = new Request("https://local.example/api/v1/trail/upload", { method: "POST", body: "small", headers: { "content-type": "multipart/form-data; boundary=test", "content-length": String(UPLOAD_BODY_LIMIT + 1) } });
        const resolve = vi.fn();
        expect((await limitRequestBodies({ event: event(request), resolve })).status).toBe(413);
        expect(resolve).not.toHaveBeenCalled();
    });

    it("bounds actual multipart bytes when the upload parser reads them", async () => {
        env.BODY_SIZE_LIMIT = "64K";
        const { request, cancelled } = streamedRequest("/api/v1/trail/upload", 128 * 1024, { "content-type": "multipart/form-data; boundary=test" });
        const resolve = vi.fn(async (input: RequestEvent) => {
            const reader = input.request.body!.getReader();
            while (!(await reader.read()).done) { /* drain without retaining */ }
            return new Response("resolved");
        });
        expect((await limitRequestBodies({ event: event(request), resolve })).status).toBe(413);
        expect(cancelled).toHaveBeenCalledOnce();
    });
});


it("uses the adapter setting for the upload route limit", async () => {
    for (const [setting, status] of [["64M", 413], ["128M", 200]] as const) {
        env.BODY_SIZE_LIMIT = setting;
        const request = new Request("https://local.example/api/v1/trail/upload", { method: "POST", body: "small", headers: { "content-type": "multipart/form-data; boundary=test", "content-length": String(70 * 1024 * 1024) } });
        expect((await limitRequestBodies({ event: event(request), resolve: async () => new Response("accepted") })).status).toBe(status);
    }
});

it.each([
    [undefined, UPLOAD_BODY_LIMIT],
    ["2G", UPLOAD_BODY_LIMIT],
    ["200M", 200 * 1024 * 1024],
    ["Infinity", Infinity],
    ["invalid", UPLOAD_BODY_LIMIT],
] as const)("handles upload configuration %s without failing the request", async (setting, limit) => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => {});
    if (setting === undefined) delete env.BODY_SIZE_LIMIT;
    else env.BODY_SIZE_LIMIT = setting;
    for (const [length, status] of Number.isFinite(limit)
        ? [[limit, 200], [limit + 1, 413]]
        : [[UPLOAD_BODY_LIMIT + 1, 200], [4 * UPLOAD_BODY_LIMIT, 200]]) {
        const request = new Request("https://local.example/api/v1/trail/upload", {
            method: "POST", body: "small",
            headers: { "content-type": "multipart/form-data; boundary=test", "content-length": String(length) },
        });
        const response = await limitRequestBodies({ event: event(request), resolve: async () => new Response("accepted") });
        expect(response.status).toBe(status);
    }
    expect(warning).toHaveBeenCalledTimes(setting === "invalid" ? 2 : 0);
});
