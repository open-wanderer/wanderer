import { createServer, type Server, type ServerResponse } from "node:http";
import type { AddressInfo } from "node:net";
import type { RequestEvent } from "@sveltejs/kit";
import type { Handle } from "@sveltejs/kit/hooks";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const publicEnv = vi.hoisted(() => ({ PUBLIC_POCKETBASE_URL: "", PUBLIC_PRIVATE_INSTANCE: undefined, PUBLIC_DISABLE_SIGNUP: undefined }));
vi.mock("$app/env/public", () => publicEnv);
vi.mock("$app/env/private", () => ({ MEILI_URL: "http://127.0.0.1:7700", ORIGIN: undefined }));
vi.mock("svelte-i18n", () => ({ locale: { set: vi.fn() } }));
// Isolate the auth hook from sequence's framework-owned request context.
vi.mock("@sveltejs/kit/hooks", () => ({ sequence: (...handlers: Handle[]) => handlers[3] }));

import { handle } from "./hooks.server";
import { proxyRegionArchive } from "#lib/server/region_archive.ts";

const user = { id: "session-user", collectionName: "users", collectionId: "users" };
const token = `header.${Buffer.from(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + 3600 })).toString("base64url")}.signature`;
const sessionCookie = `pb_auth=${encodeURIComponent(JSON.stringify({ token, record: user }))}`;

const BACKEND_PATHS = {
    apiToken: "/auth/token",
    searchToken: "/search/token",
    refresh: "/api/collections/users/auth-refresh",
    settings: "/api/collections/settings/records",
    actor: "/api/collections/activitypub_actors/records",
};

describe("auth hook with a failing backend", () => {
    let server: Server;
    let frozen: string[];
    let dropped: string[];
    let failing: Record<string, number>;
    let hung: ServerResponse[];
    let budget: AbortController;

    beforeAll(async () => {
        server = createServer(async (req, res) => {
            for await (const _ of req) { /* drain */ }
            const path = req.url!.split("?")[0];
            if (frozen.includes(path)) {
                hung.push(res);
                // Expire the hook's budget only once the backend holds the request.
                budget.abort(new DOMException("The operation timed out.", "TimeoutError"));
                return;
            }
            if (dropped.includes(path)) {
                req.socket.destroy();
                return;
            }
            res.setHeader("content-type", "application/json");
            if (failing[path]) {
                res.writeHead(failing[path]);
                res.end(JSON.stringify({ status: failing[path], message: "Refused by the test", data: {} }));
            } else if (path === BACKEND_PATHS.apiToken || path === BACKEND_PATHS.refresh) {
                res.end(JSON.stringify({ token, record: user }));
            } else if (path === BACKEND_PATHS.searchToken) {
                res.end(JSON.stringify({ token: "search-token" }));
            } else {
                res.end(JSON.stringify({ items: [{ id: "row", user: user.id }] }));
            }
        });
        await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
        publicEnv.PUBLIC_POCKETBASE_URL = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
    });

    afterAll(() => {
        server.closeAllConnections();
        return new Promise<void>((resolve, reject) => server.close(err => err ? reject(err) : resolve()));
    });

    beforeEach(() => {
        frozen = [];
        dropped = [];
        failing = {};
        hung = [];
        // The real budget is 10 s; the tests keep the real abort path but not the wait.
        budget = new AbortController();
        vi.spyOn(AbortSignal, "timeout").mockImplementation(() => budget.signal);
    });

    afterEach(() => {
        vi.restoreAllMocks();
        hung.forEach(res => res.destroy());
    });

    function request({ path = "/api/v1/trail", cookie, meiliCookie, authorization }: {
        path?: string; cookie?: string; meiliCookie?: string; authorization?: string;
    }) {
        const url = new URL(path, "http://wanderer.example");
        const headers: Record<string, string> = {};
        if (cookie) headers.cookie = cookie;
        if (authorization) headers.Authorization = authorization;
        const event = {
            url,
            request: new Request(url, { headers }),
            route: { id: path },
            cookies: { get: vi.fn(() => meiliCookie), set: vi.fn(), delete: vi.fn() },
            getClientAddress: () => "192.0.2.1",
            fetch,
            locals: {},
        } as unknown as RequestEvent;
        return { event, resolve: vi.fn(async () => new Response("resolved")) };
    }

    const session = { cookie: sessionCookie, meiliCookie: `search-token|${user.id}|1` };

    // No response may authenticate or change cookies: neither the auth hook's
    // own calls nor the export at its end.
    function expectUntouched(event: RequestEvent, resolve: ReturnType<typeof vi.fn>, response: Response | undefined) {
        expect(resolve).not.toHaveBeenCalled();
        expect(event.locals.user).toBeUndefined();
        expect(event.locals.settings).toBeUndefined();
        expect(event.cookies.delete).not.toHaveBeenCalled();
        expect(event.cookies.set).not.toHaveBeenCalled();
        expect(response?.headers.get("set-cookie") ?? null).toBeNull();
    }

    const failures: [string, string, Parameters<typeof request>[0]][] = [
        ["API token check", BACKEND_PATHS.apiToken, { authorization: "Bearer wanderer_key_valid", meiliCookie: `search-token|${user.id}|1` }],
        ["anonymous search token", BACKEND_PATHS.searchToken, {}],
        ["session search token", BACKEND_PATHS.searchToken, { cookie: sessionCookie }],
        ["session refresh", BACKEND_PATHS.refresh, session],
        ["settings lookup", BACKEND_PATHS.settings, session],
        ["actor lookup", BACKEND_PATHS.actor, session],
    ];

    it.each(failures)("answers 503 without touching cookies on a frozen backend during the %s", async (_name, path, options) => {
        frozen = [path];
        const { event, resolve } = request(options);

        const response = await handle({ event, resolve });

        expect(hung).toHaveLength(1);
        expect(AbortSignal.timeout).toHaveBeenCalledExactlyOnceWith(10_000);
        expect(response.status).toBe(503);
        expect(await response.json()).toEqual({ message: "backend_unavailable" });
        expectUntouched(event, resolve, response);
    });

    // A garbage collection can stop undici forwarding the abort through event.fetch.
    it.each([
        ["API token check", { authorization: "Bearer wanderer_key_valid", meiliCookie: `search-token|${user.id}|1` }],
        ["search token", {}],
    ])("answers 503 even if the abort never reaches event.fetch during the %s", async (_name, options) => {
        const { event, resolve } = request(options);
        (event as { fetch: unknown }).fetch = vi.fn(() => {
            budget.abort(new DOMException("The operation timed out.", "TimeoutError"));
            return new Promise(() => {});
        });

        const response = await handle({ event, resolve });

        expect(response.status).toBe(503);
        expectUntouched(event, resolve, response);
    });

    it.each(failures)("answers 503 without touching cookies on a dropped connection during the %s", async (_name, path, options) => {
        dropped = [path];
        const { event, resolve } = request(options);

        const response = await handle({ event, resolve });

        expect(response.status).toBe(503);
        expectUntouched(event, resolve, response);
    });

    it.each(failures)("answers 503 without touching cookies on a backend 502 during the %s", async (_name, path, options) => {
        failing = { [path]: 502 };
        const { event, resolve } = request(options);

        const response = await handle({ event, resolve });

        expect(response.status).toBe(503);
        expectUntouched(event, resolve, response);
    });

    it.each([400, 404, 429])("does not treat a %i from the refresh as a token rejection", async (status) => {
        failing = { [BACKEND_PATHS.refresh]: status };
        const { event, resolve } = request(session);

        const response = await handle({ event, resolve });

        expect(response.status).toBe(503);
        expectUntouched(event, resolve, response);
    });

    it("throws a 503 error page for page routes", async () => {
        dropped = [BACKEND_PATHS.refresh];
        const { event, resolve } = request({ ...session, path: "/settings" });

        await expect(handle({ event, resolve })).rejects.toMatchObject({ status: 503 });

        expectUntouched(event, resolve, undefined);
    });

    it.each([401, 403])("clears the session when the backend rejects the token with a %i", async (status) => {
        failing = { [BACKEND_PATHS.refresh]: status };
        const { event, resolve } = request(session);

        const response = await handle({ event, resolve });

        expect(response.status).toBe(200);
        expect(resolve).toHaveBeenCalledOnce();
        expect(event.locals.user).toBeNull();
        expect(event.locals.pb.authStore.token).toBe("");
        expect(event.cookies.delete).toHaveBeenCalledWith("meilisearch_token", { path: "/" });
        const cookie = response.headers.get("set-cookie");
        expect(cookie).toContain("pb_auth=");
        expect(cookie).toContain("Expires=Thu, 01 Jan 1970");
        expect(cookie).not.toContain(user.id);
    });

    it("keeps a verified session and its cookie", async () => {
        const { event, resolve } = request(session);

        const response = await handle({ event, resolve });

        expect(event.locals.user).toMatchObject({ id: user.id, actor: "row" });
        expect(event.cookies.delete).not.toHaveBeenCalled();
        expect(response.headers.get("set-cookie")).toContain(user.id);
    });

    describe("region archive download on a frozen backend", () => {
        const archive = "/regions/monaco/download";
        const download = (event: RequestEvent) => vi.fn(() => proxyRegionArchive(event, archive, "monaco-vector.pmtiles"));

        afterEach(() => vi.useRealTimers());

        it("answers 504 once the proxy gives up, when the hook needs no backend call", async () => {
            frozen = [archive];
            const { event } = request({ path: "/api/v1/regions/monaco/download", meiliCookie: "search-token|public|1" });
            const resolve = download(event);
            vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });

            const pending = handle({ event, resolve });
            await vi.waitFor(() => expect(hung).toHaveLength(1));
            await vi.advanceTimersByTimeAsync(30_000);
            const response = await pending;

            expect(resolve).toHaveBeenCalledOnce();
            expect(response.status).toBe(504);
            expect(await response.json()).toEqual({ message: "Region archive backend did not respond" });
        });

        it.each([
            ["session", BACKEND_PATHS.refresh, session],
            ["API token", BACKEND_PATHS.apiToken, { authorization: "Bearer wanderer_key_valid", meiliCookie: `search-token|${user.id}|1` }],
            ["anonymous without a search token", BACKEND_PATHS.searchToken, {}],
        ])("answers 503 from the hook before the proxy starts for %s", async (_name, path, options) => {
            frozen = [path, archive];
            const { event } = request({ ...options, path: "/api/v1/regions/monaco/download" });
            const resolve = download(event);

            const response = await handle({ event, resolve });

            expect(response.status).toBe(503);
            expect(hung).toHaveLength(1);
            expectUntouched(event, resolve, response);
        });
    });

    describe("with a forged pb_auth cookie", () => {
        const b64 = (value: object) => Buffer.from(JSON.stringify(value)).toString("base64url");
        const victim = { id: "victim", collectionName: "users", collectionId: "users", email: "victim@example.org" };
        const forged = (forgedToken: string) => `pb_auth=${encodeURIComponent(JSON.stringify({ token: forgedToken, record: victim }))}`;
        const hour = Math.floor(Date.now() / 1000) + 3600;

        // A token that passes the SDK's local expiry check, one that does not, and none.
        it.each([
            ["an unsigned future-dated token", `${b64({ alg: "none" })}.${b64({ exp: hour, id: victim.id })}.`],
            ["an expired token", `${b64({ alg: "none" })}.${b64({ exp: 1, id: victim.id })}.`],
            ["garbage", "garbage"],
            ["no token", ""],
        ])("gets no access during an outage with %s", async (_name, forgedToken) => {
            dropped = Object.values(BACKEND_PATHS);
            const { event, resolve } = request({ cookie: forged(forgedToken), meiliCookie: `search-token|${victim.id}|1`, path: "/api/v1/settings" });

            const response = await handle({ event, resolve });

            expect(response.status).toBe(503);
            expectUntouched(event, resolve, response);
        });

        it("gets no access while the backend hangs", async () => {
            frozen = Object.values(BACKEND_PATHS);
            const forgedToken = `${b64({ alg: "none" })}.${b64({ exp: hour, id: victim.id })}.`;
            const { event, resolve } = request({ cookie: forged(forgedToken), meiliCookie: `search-token|${victim.id}|1` });

            const response = await handle({ event, resolve });

            expect(response.status).toBe(503);
            expectUntouched(event, resolve, response);
        });
    });
});
