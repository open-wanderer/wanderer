import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import type { Handle, RequestEvent } from "@sveltejs/kit";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const publicEnv = vi.hoisted(() => ({ PUBLIC_POCKETBASE_URL: "", PUBLIC_PRIVATE_INSTANCE: "true" }));
vi.mock("$env/dynamic/public", () => ({ env: publicEnv }));
vi.mock("$env/dynamic/private", () => ({ env: { MEILI_URL: "http://127.0.0.1:7700" } }));
vi.mock("svelte-i18n", () => ({ locale: { set: vi.fn() } }));
// Isolate the auth hook from sequence's framework-owned request context.
vi.mock("@sveltejs/kit/hooks", () => ({ sequence: (...handlers: Handle[]) => handlers[2] }));

import { handle } from "./hooks.server";

describe("API token authentication", () => {
    let server: Server;
    let status: number;
    let message: string;
    let requests: { path: string; body: string }[];
    const user = { id: "token-user", collectionName: "users", collectionId: "users" };
    const token = `header.${Buffer.from(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + 3600 })).toString("base64url")}.signature`;

    beforeAll(async () => {
        server = createServer(async (req, res) => {
            let body = "";
            for await (const chunk of req) body += chunk;
            requests.push({ path: req.url!, body });
            res.setHeader("content-type", "application/json");
            if (req.url === "/auth/token") {
                res.writeHead(status);
                res.end(JSON.stringify(status === 200 ? { token, record: user } : { status, message, data: {} }));
            } else if (req.url === "/api/collections/users/auth-refresh") {
                res.end(JSON.stringify({ token, record: user }));
            } else if (req.url?.startsWith("/api/collections/settings/records")) {
                res.end(JSON.stringify({ items: [{ id: "settings", user: user.id }] }));
            } else if (req.url?.startsWith("/api/collections/activitypub_actors/records")) {
                res.end(JSON.stringify({ items: [{ id: "actor", user: user.id }] }));
            } else {
                res.writeHead(500);
                res.end(JSON.stringify({ message: "Unexpected backend request" }));
            }
        });
        await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
        publicEnv.PUBLIC_POCKETBASE_URL = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
    });

    afterAll(() => new Promise<void>((resolve, reject) => server.close(err => err ? reject(err) : resolve())));

    beforeEach(() => {
        requests = [];
        status = 404;
        message = "Invalid or revoked API token.";
    });

    function request(path = "/api/v1/api-token", authenticated = false, authorization: string | undefined = "Bearer wanderer_key_bogus") {
        const url = new URL(path, "http://wanderer.example");
        const event = {
            url,
            request: new Request(url, { headers: authorization ? { Authorization: authorization } : {} }),
            route: { id: path },
            cookies: {
                get: vi.fn(() => `search-token|${authenticated ? user.id : "public"}|1`),
                set: vi.fn(), delete: vi.fn(),
            },
            fetch,
            locals: {},
        } as unknown as RequestEvent;
        const resolve = vi.fn(async () => new Response("resolved"));
        return { event, resolve };
    }

    it.each(["/api/v1/api-token", "/api/v1/trail/upload", "/api/v1/category"])("returns JSON 401 for an invalid or revoked token at %s", async path => {
        const { event, resolve } = request(path);
        const response = await handle({ event, resolve });

        expect(response.status).toBe(401);
        expect(response.headers.get("content-type")).toBe("application/json");
        expect(response.headers.get("www-authenticate")).toBe('Bearer error="invalid_token"');
        expect(await response.json()).toEqual({ message: "invalid_token" });
        expect(resolve).not.toHaveBeenCalled();
        expect(requests).toEqual([{ path: "/auth/token", body: JSON.stringify({ api_token: "wanderer_key_bogus" }) }]);
        expect(event.locals.user).toBeUndefined();
        expect(event.cookies.set).not.toHaveBeenCalled();
    });

    it.each([
        [500, "Backend unavailable"],
        [404, "The requested resource wasn't found."],
        [400, "Failed to read request data"],
    ])("preserves server errors for backend %i: %s", async (backendStatus, backendMessage) => {
        status = backendStatus;
        message = backendMessage;
        const { event, resolve } = request();
        await expect(handle({ event, resolve })).rejects.toMatchObject({ status: 500 });
        expect(resolve).not.toHaveBeenCalled();
    });

    it("keeps a valid token authenticated", async () => {
        status = 200;
        const { event, resolve } = request("/api/v1/api-token", true, "Bearer wanderer_key_valid");
        const response = await handle({ event, resolve });
        expect(response.status).toBe(200);
        expect(await response.text()).toBe("resolved");
        expect(resolve).toHaveBeenCalledOnce();
        expect(event.locals.user).toMatchObject({ id: user.id, actor: "actor" });
        expect(requests[0]).toEqual({ path: "/auth/token", body: JSON.stringify({ api_token: "wanderer_key_valid" }) });
    });

    it("preserves server errors when the backend cannot be reached", async () => {
        const { event, resolve } = request();
        event.fetch = vi.fn().mockRejectedValue(new TypeError("fetch failed"));
        await expect(handle({ event, resolve })).rejects.toMatchObject({ status: 500 });
        expect(resolve).not.toHaveBeenCalled();
        expect(requests).toEqual([]);
    });

    it("keeps public requests without a token accessible", async () => {
        const { event, resolve } = request("/api/v1/category", false, "");
        expect((await handle({ event, resolve })).status).toBe(200);
        expect(requests).toEqual([]);
        expect(resolve).toHaveBeenCalledOnce();
    });
});
