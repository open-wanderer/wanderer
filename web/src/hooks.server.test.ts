import { createServer, type Server, type IncomingHttpHeaders } from "node:http";
import type { AddressInfo } from "node:net";
import type { RequestEvent } from "@sveltejs/kit";
import type { Handle } from "@sveltejs/kit/hooks";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const publicEnv = vi.hoisted(() => ({ PUBLIC_POCKETBASE_URL: "", PUBLIC_PRIVATE_INSTANCE: "true", PUBLIC_DISABLE_SIGNUP: undefined }));
vi.mock("$app/env/public", () => publicEnv);
vi.mock("$app/env/private", () => ({ MEILI_URL: "http://127.0.0.1:7700", ORIGIN: undefined }));
vi.mock("svelte-i18n", () => ({ locale: { set: vi.fn() } }));
// Isolate the auth hook from sequence's framework-owned request context.
vi.mock("@sveltejs/kit/hooks", () => ({ sequence: (...handlers: Handle[]) => handlers[3] }));

import { handle } from "./hooks.server";

// Tests swap request-scoped functions on the event.
type TestEvent = { -readonly [K in keyof RequestEvent]: RequestEvent[K] };

describe("API token authentication", () => {
    let server: Server;
    let status: number;
    let message: string;
    let requests: { path: string; body: string; headers: IncomingHttpHeaders }[];
    const user = { id: "token-user", collectionName: "users", collectionId: "users" };
    const token = `header.${Buffer.from(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + 3600 })).toString("base64url")}.signature`;

    beforeAll(async () => {
        server = createServer(async (req, res) => {
            let body = "";
            for await (const chunk of req) body += chunk;
            requests.push({ path: req.url!, body, headers: req.headers });
            res.setHeader("content-type", "application/json");
            if (req.url === "/auth/token") {
                res.writeHead(status);
                res.end(JSON.stringify(status === 200 ? { token, record: user } : { status, message, data: {} }));
            } else if (req.url === "/search/token") {
                res.end(JSON.stringify({ token: "search-token" }));
            } else if (req.url === "/api/collections/users/auth-refresh") {
                res.end(JSON.stringify({ token, record: user }));
            } else if (req.url?.startsWith("/api/collections/settings/records")) {
                res.end(JSON.stringify({ items: [{ id: "settings", user: user.id }] }));
            } else if (req.method === "POST" && req.url === "/api/collections/users/records") {
                res.end(body);
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
        status = 401;
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
            getClientAddress: () => "192.0.2.1",
            fetch,
            locals: {},
        } as unknown as TestEvent;
        const resolve = vi.fn(async () => new Response("resolved"));
        return { event, resolve };
    }

    it.each([
        ["/api/v1/api-token", "Invalid or revoked API token."],
        ["/api/v1/trail/upload", "Invalid or revoked API token."],
        ["/api/v1/category", "Invalid or revoked API token."],
        ["/api/v1/api-token", "Key has expired."],
    ])("returns JSON 401 for a rejected token at %s: %s", async (path, backendMessage) => {
        message = backendMessage;
        const { event, resolve } = request(path);
        const response = await handle({ event, resolve });

        expect(response.status).toBe(401);
        expect(response.headers.get("content-type")).toBe("application/json");
        expect(response.headers.get("www-authenticate")).toBe('Bearer error="invalid_token"');
        expect(await response.json()).toEqual({ message: "invalid_token" });
        expect(resolve).not.toHaveBeenCalled();
        expect(requests).toEqual([expect.objectContaining({ path: "/auth/token", body: JSON.stringify({ api_token: "wanderer_key_bogus" }) })]);
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
        expect(requests[0]).toMatchObject({ path: "/auth/token", body: JSON.stringify({ api_token: "wanderer_key_valid" }) });
    });

    it.each(["192.0.2.1", "2001:db8::1"])("forwards only the resolved address %s on every request", async (address) => {
        status = 200;
        const { event } = request("/api/v1/api-token", true, "Bearer wanderer_key_valid");
        event.getClientAddress = () => address;
        event.request.headers.set("X-Forwarded-For", "198.51.100.1");
        event.request.headers.set("X-Real-IP", "198.51.100.2");
        vi.mocked(event.cookies.get).mockReturnValue("");
        const callerFetch = vi.fn(fetch);
        await handle({ event, resolve: async () => {
            // Interleave another visitor before reusing the first visitor's client.
            const other = request("/api/v1/category", false, "");
            other.event.getClientAddress = () => "192.0.2.2";
            await handle({ ...other, resolve: async () => {
                await other.event.locals.pb.collection("settings").getList(1, 1);
                return new Response("resolved");
            } });
            await event.locals.pb.collection("settings").getList(1, 1, {
                headers: { "x-forwarded-for": "198.51.100.3", "X-Caller": "preserved" },
                fetch: callerFetch,
                cache: "no-store",
            });
            return new Response("resolved");
        } });

        expect(requests.map(({ path }) => path.split("?")[0])).toEqual([
            "/auth/token", "/search/token", "/api/collections/users/auth-refresh",
            "/api/collections/settings/records", "/api/collections/activitypub_actors/records",
            "/api/collections/settings/records", "/api/collections/settings/records",
        ]);
        expect(requests.map(({ headers }) => headers["x-forwarded-for"])).toEqual([
            address, address, address, address, address, "192.0.2.2", address,
        ]);
        expect(requests.every(({ headers }) => headers["x-real-ip"] === undefined)).toBe(true);
        expect(requests.at(-1)?.headers["x-caller"]).toBe("preserved");
        expect(callerFetch).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ cache: "no-store" }));
    });

    it("omits X-Forwarded-For when getClientAddress throws", async () => {
        status = 200;
        const { event, resolve } = request("/api/v1/api-token", true, "Bearer wanderer_key_valid");
        event.getClientAddress = () => {
            throw new Error("ADDRESS_HEADER was specified, but the incoming request does not include a x-forwarded-for header");
        };
        vi.mocked(event.cookies.get).mockReturnValue("");
        const response = await handle({ event, resolve });

        expect(response.status).toBe(200);
        expect(await response.text()).toBe("resolved");
        expect(resolve).toHaveBeenCalledOnce();
        expect(requests.every(({ headers }) => headers["x-forwarded-for"] === undefined)).toBe(true);

        const { options } = await event.locals.pb.beforeSend!("http://pb", {
            headers: { "Content-Type": "application/json", Authorization: token, "X-Forwarded-For": "198.51.100.1" },
        });
        expect(options?.headers).toEqual({
            "Content-Type": "application/json", Authorization: token,
        });
    });

    it("keeps SDK-managed headers and JSON bodies intact when forwarding", async () => {
        status = 200;
        const { event } = request("/api/v1/user", true, "Bearer wanderer_key_valid");
        vi.mocked(event.cookies.get).mockReturnValue("");
        const record = { username: "Test", email: "test@test.de" };
        await handle({ event, resolve: async () => {
            await event.locals.pb.collection("users").create(record);
            return new Response("resolved");
        } });

        const created = requests.find(({ path }) => path === "/api/collections/users/records");
        expect(created?.body).toBe(JSON.stringify(record));
        expect(created?.headers).toMatchObject({
            "content-type": "application/json",
            "authorization": token,
            "accept-language": expect.any(String),
            "x-forwarded-for": "192.0.2.1",
        });

        // The SDK reads headers by plain-object key after beforeSend.
        const { options } = await event.locals.pb.beforeSend!("http://pb", {
            headers: { "Content-Type": "application/json", Authorization: token, "X-FORWARDED-FOR": "198.51.100.1" },
        });
        expect(Object.getPrototypeOf(options?.headers)).toBe(Object.prototype);
        expect(options?.headers).toEqual({
            "Content-Type": "application/json", Authorization: token, "X-Forwarded-For": "192.0.2.1",
        });
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
