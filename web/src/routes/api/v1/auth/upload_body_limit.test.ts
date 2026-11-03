import { isHttpError, type RequestEvent } from "@sveltejs/kit";
import { expect, it, vi } from "vitest";
import { limitRequestBodies } from "#lib/server/request_body.ts";
import { PUT as createTrail } from "../trail/form/+server";
import { PUT as createLog } from "../summit-log/form/+server";
import { PUT as createList } from "../list/form/+server";
import { POST as uploadWaypointPhotos } from "../waypoint/[id]/file/+server";
import { POST as uploadAvatar } from "../user/[id]/file/+server";
import { POST as uploadTrailFile } from "../trail/[id]/file/+server";
import { POST as uploadListFile } from "../list/[id]/file/+server";
import { POST as uploadLogFile } from "../summit-log/[id]/file/+server";
import { POST as updateTrail } from "../trail/form/[id]/+server";
import { POST as updateList } from "../list/form/[id]/+server";
import { POST as updateLog } from "../summit-log/form/[id]/+server";
import { PUT as uploadTrail } from "../trail/upload/+server";

it.each([
    ["/api/v1/trail/form", "PUT", "gpx", createTrail],
    ["/api/v1/summit-log/form", "PUT", "gpx", createLog],
    ["/api/v1/list/form", "PUT", "avatar", createList],
    ["/api/v1/waypoint/abcdefghijklmn0/file", "POST", "photos", uploadWaypointPhotos],
    ["/api/v1/user/abcdefghijklmn0/file", "POST", "avatar", uploadAvatar],
] as const)("preserves a multi-MiB file through the real upload route %s", async (path, method, field, route) => {
    const form = new FormData();
    form.set(field, new Blob([new Uint8Array(3 * 1024 * 1024)], { type: field === "gpx" ? "application/gpx+xml" : "image/png" }), field + (field === "gpx" ? ".gpx" : ".png"));
    const record = { id: "abcdefghijklmn0", photos: [] };
    const persist = vi.fn(async (...args: unknown[]) => {
        const data = args.find((value) => value instanceof FormData) as FormData;
        const file = data.get(field) as File;
        expect(file.size).toBe(3 * 1024 * 1024);
        return { ...record };
    });
    const request = new Request("https://local.example" + path, { method, body: form });
    const event = { request, url: new URL(request.url), params: { id: record.id }, locals: { user: { id: "owner0000000001" }, pb: { collection: () => ({ create: persist, update: persist, getOne: async () => ({ ...record }) }) } } } as unknown as RequestEvent;
    const response = await limitRequestBodies({ event, resolve: (input) => route(input) });
    expect(response.status).toBe(200);
    expect(persist).toHaveBeenCalledOnce();
});

it.each([
    ["trail", uploadTrailFile],
    ["list", uploadListFile],
    ["summit-log", uploadLogFile],
] as const)("rejects an anonymous legacy %s upload before reading or parsing its body", async (kind, route) => {
    let reads = 0;
    const body = new ReadableStream<Uint8Array>({
        pull(controller) { reads++; controller.enqueue(new Uint8Array(16 * 1024)); controller.close(); },
    }, { highWaterMark: 0 });
    const request = new Request(`https://local.example/api/v1/${kind}/abcdefghijklmn0/file`, {
        method: "POST", body, duplex: "half",
        headers: { "content-type": "multipart/form-data; boundary=test", "content-length": String(1024 ** 3) },
    } as RequestInit);
    const parse = vi.spyOn(Request.prototype, "formData");
    const persist = vi.fn();
    try {
        const event = { request, url: new URL(request.url), params: { id: "abcdefghijklmn0" }, locals: { pb: { collection: persist } } } as unknown as RequestEvent;
        const response = await limitRequestBodies({ event, resolve: (input) => route(input) });
        expect(response.status).toBe(401);
        expect(await response.json()).toEqual({ message: "Unauthorized" });
        expect(parse).not.toHaveBeenCalled();
        expect(reads).toBe(0);
        expect(persist).not.toHaveBeenCalled();
    } finally { parse.mockRestore(); }
});


it.each([
    ["/api/v1/trail/form", "PUT", createTrail],
    ["/api/v1/summit-log/form", "PUT", createLog],
    ["/api/v1/waypoint/abcdefghijklmn0/file", "POST", uploadWaypointPhotos],
] as const)("accepts several default-sized photos through %s", async (path, method, route) => {
    const form = new FormData();
    for (let i = 0; i < 3; i++) form.append("photos", new Blob([new Uint8Array(20 * 1024 * 1024)], { type: "image/png" }), `photo-${i}.png`);
    const record = { id: "abcdefghijklmn0", photos: [] };
    const persist = vi.fn(async (...args: unknown[]) => {
        const data = args.find((value) => value instanceof FormData) as FormData;
        expect(data.getAll("photos").map((file) => (file as File).size)).toEqual(Array(3).fill(20 * 1024 * 1024));
        return { ...record };
    });
    const request = new Request("https://local.example" + path, { method, body: form });
    const event = { request, url: new URL(request.url), params: { id: record.id }, locals: { user: { id: "owner0000000001" }, pb: { collection: () => ({ create: persist, update: persist, getOne: async () => ({ ...record }) }) } } } as unknown as RequestEvent;
    const response = await limitRequestBodies({ event, resolve: (input) => route(input) });
    expect(response.status).toBe(200);
    expect(persist).toHaveBeenCalledOnce();
});

it.each([
    ["/api/v1/trail/form", "PUT", createTrail],
    ["/api/v1/summit-log/form", "PUT", createLog],
    ["/api/v1/list/form", "PUT", createList],
    ["/api/v1/waypoint/abcdefghijklmn0/file", "POST", uploadWaypointPhotos],
    ["/api/v1/user/abcdefghijklmn0/file", "POST", uploadAvatar],
] as const)("rejects an anonymous upload without parsing its body at %s", async (path, method, route) => {
    let reads = 0;
    const body = new ReadableStream<Uint8Array>({ pull(controller) { reads++; controller.enqueue(new Uint8Array(16 * 1024)); controller.close(); } }, { highWaterMark: 0 });
    const request = new Request("https://local.example" + path, { method, body, duplex: "half", headers: { "content-type": "multipart/form-data; boundary=test" } } as RequestInit);
    const parse = vi.spyOn(Request.prototype, "formData");
    const persist = vi.fn();
    try {
        const event = { request, url: new URL(request.url), params: { id: "abcdefghijklmn0" }, locals: { pb: { collection: () => ({ create: persist, update: persist }) } } } as unknown as RequestEvent;
        const response = await limitRequestBodies({ event, resolve: (input) => route(input) });
        expect(response.status).toBe(401);
        expect(response.headers.get("content-type")).toContain("application/json");
        expect(await response.json()).toEqual({ message: "Unauthorized" });
        expect(parse).not.toHaveBeenCalled();
        expect(reads).toBe(0);
        expect(persist).not.toHaveBeenCalled();
    } finally { parse.mockRestore(); }
});

it.each([
    ["/api/v1/trail/form/abcdefghijklmn0", "POST", updateTrail],
    ["/api/v1/list/form/abcdefghijklmn0", "POST", updateList],
    ["/api/v1/summit-log/form/abcdefghijklmn0", "POST", updateLog],
    ["/api/v1/trail/upload", "PUT", uploadTrail],
] as const)("rejects an anonymous upload before consuming its body at %s", async (path, method, route) => {
    let reads = 0;
    const body = new ReadableStream<Uint8Array>({
        pull(controller) { reads++; controller.enqueue(new Uint8Array(16 * 1024)); controller.close(); },
    }, { highWaterMark: 0 });
    const request = new Request("https://local.example" + path, {
        method, body, duplex: "half",
        headers: { "content-type": "multipart/form-data; boundary=test", "content-length": String(1024 ** 3) },
    } as RequestInit);
    const parse = vi.spyOn(Request.prototype, "formData");
    const persist = vi.fn();
    try {
        const event = { request, url: new URL(request.url), params: { id: "abcdefghijklmn0" }, locals: { pb: { collection: persist } } } as unknown as RequestEvent;
        const response = await limitRequestBodies({ event, resolve: async (input) => {
            try { return await route(input); }
            catch (cause) {
                if (isHttpError(cause)) return Response.json(cause.body, { status: cause.status });
                throw cause;
            }
        } });
        expect(response.status).toBe(401);
        expect(await response.json()).toEqual({ message: "Unauthorized" });
        expect(parse).not.toHaveBeenCalled();
        expect(reads).toBe(0);
        expect(persist).not.toHaveBeenCalled();
    } finally { parse.mockRestore(); }
});
