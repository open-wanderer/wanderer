import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRateLimiter } from "./rate_limit";

describe("createRateLimiter", () => {
    beforeEach(() => { vi.useFakeTimers(); });
    afterEach(() => { vi.useRealTimers(); });

    it("allows the limit and refuses the next request with the seconds to wait", () => {
        const limit = createRateLimiter(2, 60_000);

        expect(limit("a")).toBeUndefined();
        expect(limit("a")).toBeUndefined();
        vi.advanceTimersByTime(20_000);
        expect(limit("a")).toBe(40);
    });

    it("counts each key on its own", () => {
        const limit = createRateLimiter(1, 60_000);

        limit("a");

        expect(limit("a")).toBe(60);
        expect(limit("b")).toBeUndefined();
    });

    it("starts over after the window", () => {
        const limit = createRateLimiter(1, 60_000);
        limit("a");
        limit("a");

        vi.advanceTimersByTime(60_000);

        expect(limit("a")).toBeUndefined();
    });

    it("keeps working past the number of tracked keys", () => {
        const limit = createRateLimiter(1, 60_000);

        for (let i = 0; i < 25_000; i++) limit(`key${i}`);

        expect(limit("fresh")).toBeUndefined();
        expect(limit("fresh")).toBe(60);
    });
});
