import type { Handle, RequestEvent } from "@sveltejs/kit";
import { expect, it, vi } from "vitest";

vi.mock("$env/dynamic/public", () => ({ env: { PUBLIC_POCKETBASE_URL: "http://backend.example" } }));
vi.mock("$env/dynamic/private", () => ({ env: { MEILI_URL: "http://search.example" } }));
vi.mock("svelte-i18n", () => ({ locale: { set: vi.fn() } }));
vi.mock("@sveltejs/kit/hooks", () => ({ sequence: (...handlers: Handle[]): Handle => async ({ event, resolve }) => {
    const next = (index: number, current: RequestEvent): ReturnType<Handle> => index === handlers.length
        ? resolve(current)
        : handlers[index]({ event: current, resolve: (input) => next(index + 1, input) });
    return next(0, event);
} }));

import { handle } from "./hooks.server";
import { LOGIN_BODY_LIMIT } from "$lib/server/request_body";

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
