import type { RequestEvent } from "@sveltejs/kit";
import type { Handle } from "@sveltejs/kit/hooks";
import { describe, expect, it, vi } from "vitest";

vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "http://backend.example", PUBLIC_PRIVATE_INSTANCE: undefined, PUBLIC_DISABLE_SIGNUP: undefined, PUBLIC_MAP_MAX_POLYLINES: undefined }));
vi.mock("$app/env/private", () => ({ MEILI_URL: "http://search.example", ORIGIN: undefined, BODY_SIZE_LIMIT: undefined }));
vi.mock("svelte-i18n", () => ({ locale: { set: vi.fn() } }));
vi.mock("@sveltejs/kit/hooks", () => ({ sequence: (...handlers: Handle[]): Handle => async ({ event, resolve }) => {
    const next = async (index: number, current: RequestEvent): Promise<Response> => index === handlers.length
        ? resolve(current)
        : handlers[index]({ event: current, resolve: (input) => next(index + 1, input) });
    return next(0, event);
} }));

import { handle } from "./hooks.server";
import { TRACK_BODY_LIMIT, LOGIN_BODY_LIMIT, REQUEST_BODY_LIMIT } from "#lib/server/request_body.ts";
import { POST as convert } from "./routes/api/v1/trail/convert/+server";

it("rejects an anonymous chunked body before auth and route execution in the installed hook", async () => {
    const bytes = new TextEncoder().encode(JSON.stringify({ username: "alice", password: "password123", extra: "x".repeat(LOGIN_BODY_LIMIT) }));
    const body = new ReadableStream<Uint8Array>({ start(controller) { controller.enqueue(bytes); controller.close(); } });
    const url = new URL("https://local.example/api/v1/auth/login");
    const init: RequestInit & { duplex: "half" } = { method: "POST", body, duplex: "half", headers: { "content-type": "application/json" } };
    const backendFetch = vi.fn().mockResolvedValue(new Response("{}", { status: 500 }));
    const event = { url, request: new Request(url, init), route: { id: "/api/v1/auth/login" },
        cookies: { get: () => "search-token|public|1", set: vi.fn(), delete: vi.fn() }, fetch: backendFetch, locals: {} } as unknown as RequestEvent;
    const resolve = vi.fn(async () => new Response("parsed"));
    const response = await handle({ event, resolve });
    expect(response.status).toBe(413);
    expect(resolve).not.toHaveBeenCalled();
    expect(backendFetch).not.toHaveBeenCalled();
});

describe("POST /api/v1/trail/convert through the installed hook", () => {
    // Pads a real track so that the request is over the 1 MiB default limit.
    function gpxFile(size: number) {
        const track = `<?xml version="1.0"?><gpx version="1.1" creator="t"><trk><trkseg><trkpt lat="47" lon="11"/><trkpt lat="47.001" lon="11.001"/></trkseg></trk></gpx>`;
        return new Blob([track, " ".repeat(size - track.length)], { type: "application/gpx+xml" });
    }

    function convertThroughHandle(request: Request) {
        const event = { url: new URL(request.url), request, route: { id: "/api/v1/trail/convert" },
            cookies: { get: () => `search-token|public|1|${Date.now() + 3_600_000}`, set: vi.fn(), delete: vi.fn() },
            fetch: vi.fn().mockResolvedValue(new Response("{}", { status: 500 })), locals: {} } as unknown as RequestEvent;
        const resolve = vi.fn(async (input: RequestEvent) => {
            input.locals.pb.authStore.save("token", { id: "user", collectionName: "users" } as never);
            return convert(input);
        });
        return { resolve, response: handle({ event, resolve }) };
    }

    function multipart(file: Blob) {
        const body = new FormData();
        body.set("file", file, "track.gpx");
        return new Request("https://local.example/api/v1/trail/convert", { method: "POST", body });
    }

    it("converts an upload above the default body limit", async () => {
        const size = 2 * REQUEST_BODY_LIMIT;
        const { response } = convertThroughHandle(multipart(gpxFile(size)));

        expect((await response).status).toBe(200);
    });

    it("takes a raw GPX body above the default body limit", async () => {
        const { response } = convertThroughHandle(new Request("https://local.example/api/v1/trail/convert", {
            method: "POST", body: await gpxFile(2 * REQUEST_BODY_LIMIT).text(),
        }));

        expect((await response).status).toBe(200);
    });

    it("rejects a declared body above the conversion limit before the route", async () => {
        const { resolve, response } = convertThroughHandle(new Request("https://local.example/api/v1/trail/convert", {
            method: "POST", body: "x", headers: { "content-length": String(TRACK_BODY_LIMIT + 1) },
        }));

        expect((await response).status).toBe(413);
        expect(resolve).not.toHaveBeenCalled();
    });

    it("stops a chunked body above the conversion limit", async () => {
        const chunk = new Uint8Array(1024 * 1024);
        let sent = 0;
        const body = new ReadableStream<Uint8Array>({
            pull(controller) {
                sent += chunk.byteLength;
                controller.enqueue(chunk);
            },
        });
        const init: RequestInit & { duplex: "half" } = { method: "POST", body, duplex: "half" };
        const { response } = convertThroughHandle(new Request("https://local.example/api/v1/trail/convert", init));

        expect((await response).status).toBe(413);
        expect(sent).toBeLessThan(TRACK_BODY_LIMIT + 8 * chunk.byteLength);
    });
});

it("limits the body of /api/v1/trail/upload like /trail/convert", async () => {
    const url = new URL("https://local.example/api/v1/trail/upload");
    const request = new Request(url, {
        method: "PUT", body: "x",
        headers: { "content-type": "multipart/form-data; boundary=test", "content-length": String(TRACK_BODY_LIMIT + 1) },
    });
    const event = { url, request, route: { id: "/api/v1/trail/upload" },
        cookies: { get: () => `search-token|public|1|${Date.now() + 3_600_000}`, set: vi.fn(), delete: vi.fn() },
        fetch: vi.fn(), locals: {} } as unknown as RequestEvent;
    const resolve = vi.fn(async () => new Response("parsed"));

    expect((await handle({ event, resolve })).status).toBe(413);
    expect(resolve).not.toHaveBeenCalled();
});
