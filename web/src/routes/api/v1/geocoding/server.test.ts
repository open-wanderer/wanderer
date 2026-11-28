import { once } from "node:events";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GET as reverse } from "./reverse/+server";
import { GET as search } from "./search/+server";

const secret = "configured-test-secret";
const privateEnv = vi.hoisted(() => ({ POCKETBASE_PROXY_SECRET: undefined as string | undefined }));
vi.mock("$app/env/private", () => privateEnv);

const featureCollection = {
    type: "FeatureCollection",
    licence: "Test fixture",
    features: [{
        type: "Feature",
        geometry: { type: "Point", coordinates: [2, 1] },
        properties: { name: "Berg", address: { town: "Berg" } },
    }],
};
const failure = { message: "Nominatim request failed" };

function requestEvent(endpoint: string, query: string, init?: RequestInit, getClientAddress?: () => string) {
    const url = new URL(`https://wanderer.test/api/v1/geocoding/${endpoint}${query}`);
    const eventFetch = vi.fn();
    const event = {
        url,
        request: new Request(url, init),
        locals: { pb: new PocketBase("https://pocketbase.test/backend/") },
        fetch: eventFetch,
        getClientAddress,
    } as unknown as RequestEvent;
    privateEnv.POCKETBASE_PROXY_SECRET = secret;
    return { event, eventFetch };
}

function mockFetch(response = Response.json(featureCollection)) {
    const fetch = vi.fn(async (_url: URL, _options: RequestInit) => response);
    vi.stubGlobal("fetch", fetch);
    return fetch;
}

afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
});

describe.each([
    {
        endpoint: "search",
        get: search,
        query: "?q=Berg+%26+湖&q=ignored&limit=0x10&limit=invalid&lat=1&lon=2",
        expected: { q: "Berg & 湖", limit: "0x10" },
    },
    {
        endpoint: "reverse",
        get: reverse,
        query: "?lat=%2BInfinity&lon=-1e3&lat=invalid&lon=ignored&q=ignored&limit=2",
        expected: { lat: "+Infinity", lon: "-1e3" },
    },
])("public geocoding $endpoint", ({ endpoint, get, query, expected }) => {
    it("sends only accepted first query values and fixed server credentials to PocketBase", async () => {
        const { event, eventFetch } = requestEvent(endpoint,
            query + "&format=json&addressdetails=0&url=https://untrusted.test/&path=/another&headers=ignored",
            { headers: {
                Authorization: "Bearer visitor-token",
                Cookie: "pb_auth=visitor-cookie",
                "Accept-Language": "fr",
                "X-Forwarded-For": "192.0.2.1",
                "X-Internal-Secret": "visitor-secret",
            } },
            () => "203.0.113.7",
        );
        event.locals.pb.authStore.save("sdk-visitor-token");
        const fetch = mockFetch(Response.json(featureCollection, {
            status: 201,
            headers: { "Content-Type": "application/geo+json", "Set-Cookie": "unexpected=private" },
        }));

        const response = await get(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual(featureCollection);
        expect(response.headers.get("content-type")).toBe("application/json");
        expect(response.headers.has("set-cookie")).toBe(false);
        expect(eventFetch).not.toHaveBeenCalled();
        expect(fetch).toHaveBeenCalledOnce();
        const [url, options] = fetch.mock.calls[0];
        expect(url.origin).toBe("https://pocketbase.test");
        expect(url.pathname).toBe(`/backend/geocoding/${endpoint}`);
        expect([...url.searchParams]).toEqual(Object.entries(expected));
        expect(options.method).toBe("GET");
        expect(options.credentials).toBe("omit");
        expect(options.redirect).toBe("error");
        expect([...new Headers(options.headers)]).toEqual([
            ["x-forwarded-for", "203.0.113.7"], ["x-internal-secret", secret],
        ]);
        expect(options.signal).toBeInstanceOf(AbortSignal);
    });

    it("does not forward a caller-supplied address when the adapter cannot resolve one", async () => {
        const { event } = requestEvent(endpoint, query, { headers: { "X-Forwarded-For": "192.0.2.1" } },
            () => { throw new Error("Address unavailable"); });
        const fetch = mockFetch();

        const response = await get(event);

        expect(response.status).toBe(200);
        expect(new Headers(fetch.mock.calls[0][1].headers).has("x-forwarded-for")).toBe(false);
    });

    it.each([200, 204])("keeps empty backend result (HTTP %s) as a successful empty JSON object for anonymous callers", async status => {
        const { event } = requestEvent(endpoint, query);
        const fetch = mockFetch(status === 204 ? new Response(null, { status }) : Response.json({}));

        const response = await get(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual({});
        expect(fetch).toHaveBeenCalledOnce();
    });

    it.each([
        [400, '{"message":"private query detail", "detail":{"url":"https://private.test"}}'],
        [401, '{"message":"wrong internal secret", "detail":"private-test-token"}'],
        [404, '{"error":"private service detail"}'],
        [429, "wait for https://private.test"],
        [503, "private service failure"],
    ])("preserves backend HTTP %s without reflecting its error body or headers", async (status, body) => {
        const { event } = requestEvent(endpoint, query);
        mockFetch(new Response(body, { status, headers: { Location: "https://private.test" } }));

        const response = await get(event);

        expect(response.status).toBe(status);
        expect(await response.json()).toEqual(failure);
        expect(response.headers.has("location")).toBe(false);
    });

    it.each([
        "<html>private url</html>",
        '{"error":"private url"}',
        '{"error":"Unable to geocode: https://private.test"}',
        '{"error":"Unable to geocode","detail":"https://private.test"}',
        '{"error":"Unable to geocode","type":"FeatureCollection","features":[]}',
        '{"message":"private url","detail":{"secret":"test-private"}}',
        "[]",
        "null",
    ])("does not reflect unexpected successful backend responses", async body => {
        const { event } = requestEvent(endpoint, query);
        mockFetch(new Response(body));

        const response = await get(event);

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual(failure);
    });

    it("does not log or reflect exceptions from the internal request", async () => {
        const { event } = requestEvent(endpoint, query);
        const error = vi.spyOn(console, "error").mockImplementation(() => {});
        const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
        vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("https://private.test", {
            cause: new Error("private-test-token"),
        })));

        const response = await get(event);

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual(failure);
        expect(error).not.toHaveBeenCalled();
        expect(warn).not.toHaveBeenCalled();
    });

    it("does not contact a redirect target on the PocketBase origin", async () => {
        const requests: string[] = [];
        const backend = createServer((req, res) => {
            requests.push(req.url ?? "");
            if (req.url?.startsWith("/geocoding/")) {
                res.writeHead(302, { Location: "/never-follow" });
            }
            res.end();
        }).listen(0, "127.0.0.1");
        try {
            await once(backend, "listening");
            const { port } = backend.address() as AddressInfo;
            const { event } = requestEvent(endpoint, query);
            event.locals.pb = new PocketBase(`http://127.0.0.1:${port}`);

            const response = await get(event);

            expect(response.status).toBe(502);
            expect(await response.json()).toEqual(failure);
            expect(requests).toHaveLength(1);
            expect(requests[0]).toMatch(new RegExp(`^/geocoding/${endpoint}\\?`));
        } finally {
            backend.closeAllConnections();
            await new Promise<void>(resolve => backend.close(() => resolve()));
        }
    });

    it("propagates a public request abort to the backend fetch", async () => {
        const controller = new AbortController();
        const { event } = requestEvent(endpoint, query, { signal: controller.signal });
        const fetch = vi.fn((_url: URL, { signal }: RequestInit) => new Promise<Response>((_resolve, reject) => {
            signal?.addEventListener("abort", () => reject(signal.reason), { once: true });
        }));
        vi.stubGlobal("fetch", fetch);

        const pending = get(event);
        controller.abort();
        const response = await pending;

        expect(fetch.mock.calls[0][1].signal?.aborted).toBe(true);
        expect(response.status).toBe(502);
        expect(await response.json()).toEqual(failure);
    });

    it("fails closed without a configured secret, even if the caller supplies one", async () => {
        const { event } = requestEvent(endpoint, query, { headers: { "X-Internal-Secret": secret } });
        privateEnv.POCKETBASE_PROXY_SECRET = undefined;
        const fetch = mockFetch();

        const response = await get(event);

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual(failure);
        expect(fetch).not.toHaveBeenCalled();
    });
});

describe("reverse geocoding without a matching address", () => {
    it.each([200, 202])("keeps the exact no-address result from a successful response (HTTP %s)", async status => {
        const noResult = { error: "Unable to geocode" };
        const { event } = requestEvent("reverse", "?lat=0&lon=0");
        mockFetch(Response.json(noResult, { status }));

        const response = await reverse(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual(noResult);
    });

    it("does not accept the reverse no-address envelope for search", async () => {
        const { event } = requestEvent("search", "?q=unknown");
        mockFetch(Response.json({ error: "Unable to geocode" }));

        const response = await search(event);

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual(failure);
    });
});

describe("public geocoding query contract", () => {
    it.each([
        ["search", search, "", "Missing query parameter: q"],
        ["search", search, "?q=&q=valid", "Missing query parameter: q"],
        ["search", search, "?q=valid&limit=NaN&limit=2", "Invalid query parameter: limit"],
        ["reverse", reverse, "?lat=1", "Missing query parameter: lat or lon"],
        ["reverse", reverse, "?lat=&lat=1&lon=2", "Missing query parameter: lat or lon"],
        ["reverse", reverse, "?lat=NaN&lat=1&lon=2", "Invalid query parameter: lat or lon"],
        ["reverse", reverse, "?lat=1&lon=1,2", "Invalid query parameter: lat or lon"],
    ] as const)("%s rejects invalid query %# before contacting PocketBase", async (endpoint, get, query, message) => {
        const { event } = requestEvent(endpoint, query);
        const fetch = mockFetch();

        const response = await get(event);

        expect(response.status).toBe(400);
        expect(await response.json()).toEqual({ message });
        expect(fetch).not.toHaveBeenCalled();
    });

    it.each([
        ["search", search, "?q=+&limit=", { q: " " }],
        ["search", search, "?q=Berg&limit=+", { q: "Berg", limit: " " }],
        ["reverse", reverse, "?lat=+&lon=0b10", { lat: " ", lon: "0b10" }],
    ] as const)("%s preserves previously accepted whitespace and numeric spellings", async (endpoint, get, query, expected) => {
        const { event } = requestEvent(endpoint, query);
        const fetch = mockFetch();

        const response = await get(event);

        expect(response.status).toBe(200);
        expect([...fetch.mock.calls[0][0].searchParams]).toEqual(Object.entries(expected));
    });
});
