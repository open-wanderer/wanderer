import type { RequestEvent } from "@sveltejs/kit";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const config = vi.hoisted(() => ({ baseUrl: "https://nominatim.openstreetmap.org" }));

vi.mock("$app/environment", () => ({ version: "test" }));
vi.mock("$lib/server/url", () => ({ resolveBaseUrl: () => config.baseUrl }));

import { fetchNominatim } from "./nominatim";

describe("fetchNominatim rate limiting", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(1_000_000);
    });

    afterEach(() => vi.useRealTimers());

    it("spaces concurrent public Nominatim requests by at least one second", async () => {
        const requestTimes: number[] = [];
        const event = {
            fetch: vi.fn(async () => {
                requestTimes.push(Date.now());
                return new Response(null, { status: 200 });
            }),
        } as unknown as RequestEvent;

        const requests = Promise.all([
            fetchNominatim(event, "search", new URLSearchParams({ q: "one" })),
            fetchNominatim(event, "search", new URLSearchParams({ q: "two" })),
            fetchNominatim(event, "search", new URLSearchParams({ q: "three" })),
        ]);
        await vi.runAllTimersAsync();
        await requests;

        expect(requestTimes).toHaveLength(3);
        expect(requestTimes[1] - requestTimes[0]).toBeGreaterThanOrEqual(1000);
        expect(requestTimes[2] - requestTimes[1]).toBeGreaterThanOrEqual(1000);
    });

    it("retries a 429 after honoring Retry-After", async () => {
        const requestTimes: number[] = [];
        let attempts = 0;
        const event = {
            fetch: vi.fn(async () => {
                requestTimes.push(Date.now());
                attempts += 1;
                return attempts === 1
                    ? new Response(null, { status: 429, headers: { "Retry-After": "2" } })
                    : new Response(null, { status: 200 });
            }),
        } as unknown as RequestEvent;

        const request = fetchNominatim(event, "reverse", new URLSearchParams({ lat: "1", lon: "2" }));
        await vi.runAllTimersAsync();
        const response = await request;

        expect(response.status).toBe(200);
        expect(attempts).toBe(2);
        expect(requestTimes[1] - requestTimes[0]).toBeGreaterThanOrEqual(2000);
    });

    it("returns a 429 without waiting when Retry-After is too long", async () => {
        const event = {
            fetch: vi.fn(async () => new Response(null, { status: 429, headers: { "Retry-After": "3600" } })),
        } as unknown as RequestEvent;

        const request = fetchNominatim(event, "search", new URLSearchParams({ q: "busy" }));
        await vi.runAllTimersAsync();
        const response = await request;

        expect(response.status).toBe(429);
        expect(event.fetch).toHaveBeenCalledTimes(1);
    });

    it("aborts a request that does not answer in time and retries it", async () => {
        let attempts = 0;
        const event = {
            fetch: vi.fn((_url: string, init: RequestInit) => {
                attempts += 1;
                if (attempts > 1) {
                    return Promise.resolve(new Response(null, { status: 200 }));
                }
                return new Promise<Response>((_resolve, reject) => {
                    init.signal?.addEventListener("abort", () => reject(new Error("aborted")));
                });
            }),
        } as unknown as RequestEvent;

        const request = fetchNominatim(event, "search", new URLSearchParams({ q: "slow" }));
        await vi.runAllTimersAsync();
        const response = await request;

        expect(response.status).toBe(200);
        expect(attempts).toBe(2);
    });

    it("does not let a hanging request block the queue", async () => {
        const event = {
            fetch: vi.fn((_url: string, init: RequestInit) =>
                new Promise<Response>((_resolve, reject) => {
                    init.signal?.addEventListener("abort", () => reject(new Error("aborted")));
                }),
            ),
        } as unknown as RequestEvent;
        const next = {
            fetch: vi.fn(async () => new Response(null, { status: 200 })),
        } as unknown as RequestEvent;

        const hanging = fetchNominatim(event, "search", new URLSearchParams({ q: "hang" }));
        const hangingResult = hanging.catch((error: Error) => error);
        const following = fetchNominatim(next, "search", new URLSearchParams({ q: "next" }));
        await vi.runAllTimersAsync();

        expect(await hangingResult).toBeInstanceOf(Error);
        expect((await following).status).toBe(200);
    });
});
