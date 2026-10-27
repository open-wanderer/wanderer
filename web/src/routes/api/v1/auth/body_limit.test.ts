import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";

vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: undefined }));
vi.mock("$app/env/private", () => ({ POCKETBASE_PROXY_SECRET: "proxy-secret", BODY_SIZE_LIMIT: undefined }));

import { POST as login } from "./login/+server";
import { POST as inbox } from "../activitypub/user/[handle]/inbox/+server";
import { INBOX_BODY_LIMIT, LOGIN_BODY_LIMIT } from "#lib/server/request_body.ts";

function fixture(path: string, body: BodyInit, headers?: HeadersInit) {
    const init: RequestInit & { duplex: "half" } = { method: "POST", body, headers, duplex: "half" };
    const request = new Request("https://local.example" + path, init);
    const authWithPassword = vi.fn().mockResolvedValue({ token: "accepted" });
    const send = vi.fn().mockResolvedValue(true);
    const event = { request, url: new URL(request.url), fetch: vi.fn(), locals: { pb: { collection: () => ({ authWithPassword }), send } } } as unknown as RequestEvent;
    return { event, authWithPassword, send };
}

describe("anonymous body parsing", () => {
    it.each([
        ["login", "/api/v1/auth/login", LOGIN_BODY_LIMIT, login],
        ["inbox", "/api/v1/activitypub/user/alice/inbox", INBOX_BODY_LIMIT, inbox],
    ] as const)("rejects an oversized %s before authentication or forwarding", async (_, path, limit, handler) => {
        const body = new ReadableStream<Uint8Array>({ start(controller) { controller.enqueue(new TextEncoder().encode(JSON.stringify({ actor: "https://remote.example/alice", username: "alice", password: "password123", padding: "x".repeat(limit) }))); controller.close(); } });
        const { event, authWithPassword, send } = fixture(path, body);
        await expect(handler(event)).rejects.toMatchObject({ status: 413, body: { message: "request_body_too_large" } });
        expect(authWithPassword).not.toHaveBeenCalled();
        expect(send).not.toHaveBeenCalled();
    });

    it("retains normal password authentication", async () => {
        const { event, authWithPassword } = fixture("/api/v1/auth/login", JSON.stringify({ username: "alice", password: "password123" }));
        expect((await login(event)).status).toBe(200);
        expect(authWithPassword).toHaveBeenCalledWith("alice", "password123");
    });

    it("forwards the exact bounded activity body and signature", async () => {
        const raw = '{ "actor": "https://remote.example/alice", "type": "Like", "name": "\u00e4" }\n';
        const { event, send } = fixture("/api/v1/activitypub/user/alice/inbox", raw, { Signature: "original" });
        expect((await inbox(event)).status).toBe(200);
        expect(send).toHaveBeenCalledWith("/activitypub/activity/process", expect.objectContaining({ body: raw, headers: expect.objectContaining({ signature: "original", "X-Forwarded-Path": event.url.pathname, "X-Internal-Secret": "proxy-secret" }) }));
    });
});
